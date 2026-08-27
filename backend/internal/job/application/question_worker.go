package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	"github.com/intivai/backend/internal/job/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// TaskGenerateQuestionSet — enqueued when a job becomes published, exactly
// like TaskGenerateRubric. Question generation is a PARALLEL concern to the
// rubric: separate task, separate worker, separate failure surface, so a
// rubric outage never costs a job its questions (or vice versa).
const TaskGenerateQuestionSet = "generate_question_set"

// questionSetErrorMaxLen keeps the persisted failure reason short and
// operator-readable (provider errors can carry huge payload dumps).
const questionSetErrorMaxLen = 300

// questionSetCount — how many questions the LLM is asked for. The interview
// itself still applies its own per-interview limit.
const questionSetCount = 8

type GenerateQuestionSetPayload struct {
	JobID string `json:"job_id"`
	OrgID string `json:"org_id"`
	// UpdatedAt is the job's row version at enqueue time (J7). The worker
	// writes conditionally on it: if a republish/edit bumped the job since
	// enqueue, this generation is stale and must not overwrite the newer
	// cycle's set.
	UpdatedAt int64 `json:"updated_at_nano"`
}

// QuestionWorker generates the job's interview question set ONCE, at publish
// time (decision D5). Generating per interview would add provider latency to
// the candidate hot path and hand two candidates different questions for the
// same job — the stored set is the consistency guarantee.
type QuestionWorker struct {
	pool      *gorm.DB
	repo      domain.JobRepository
	llmClient *llm.Client
	log       zerolog.Logger
	now       func() time.Time
}

func NewQuestionWorker(pool *gorm.DB, repo domain.JobRepository, llmClient *llm.Client, log zerolog.Logger) *QuestionWorker {
	return &QuestionWorker{pool: pool, repo: repo, llmClient: llmClient, log: log, now: func() time.Time { return time.Now().UTC() }}
}

func (w *QuestionWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGenerateQuestionSet, w.handle)
}

func (w *QuestionWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p GenerateQuestionSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	id, err := uuid.Parse(p.JobID)
	if err != nil {
		return asynq.SkipRetry
	}

	var job *domain.Job
	var stored json.RawMessage
	err = db.RunInTx(ctx, w.pool, p.OrgID, func(txCtx context.Context) error {
		var txErr error
		job, txErr = w.repo.GetByID(txCtx, id)
		if txErr != nil {
			// Missing row also covers the cross-tenant case: RLS hides another
			// org's job, so the task can never be retried into existence.
			if errors.Is(txErr, domain.ErrNotFound) {
				return asynq.SkipRetry
			}
			return txErr
		}
		stored, txErr = LoadQuestionSet(txCtx, id)
		return txErr
	})
	if err != nil {
		return err
	}

	// Idempotency guard, result-gated like the rubric worker: a usable set
	// already on the row means a retry must not spend another LLM call and
	// must not rewrite what candidates are already being asked.
	if _, ok := domain.SelectQuestionSet(stored); ok {
		return nil
	}

	// Publish is the only trigger. A draft (or unpublished-again) job gets no
	// generation — nothing is interviewing against it.
	if !job.IsPublished {
		return nil
	}

	questions, err := w.generate(ctx, job)
	if err != nil {
		return w.fail(ctx, p, id, err)
	}

	set := domain.NewQuestionSet(domain.QuestionSourceLLM, questions, w.now())
	// Marshal validates: an invalid set cannot reach the column, so a failed
	// generation always leaves question_set untouched (no partial persistence).
	raw, err := set.Marshal()
	if err != nil {
		return w.fail(ctx, p, id, err)
	}

	return db.RunInTx(ctx, w.pool, p.OrgID, func(txCtx context.Context) error {
		// Column-scoped write: a full-row Update() would clobber any recruiter
		// edit made while the LLM call was running. CAS on the enqueue-time
		// version (J7): a republish/edit mid-generation stamps a newer
		// updated_at, and this stale task must not overwrite the newer cycle.
		// Payloads without a version (legacy/unsigned tasks) fall back to the
		// row version loaded in the pre-generation tx: the CAS still holds.
		version := p.UpdatedAt
		if version == 0 {
			version = job.UpdatedAt.UnixNano()
		}
		stale, err := storeQuestionSet(txCtx, id, raw, version)
		if err != nil {
			return err
		}
		if stale {
			w.log.Info().Str("job_id", p.JobID).Int64("task_version", version).
				Msg("question set generation skipped: job advanced since enqueue")
		}
		return nil
	})
}

// generate asks the provider for a descriptive set and validates it strictly.
// Every returned error is retryable by design: a bad or missing response must
// leave the job exactly as it was, and the next attempt costs one call.
func (w *QuestionWorker) generate(ctx context.Context, job *domain.Job) ([]domain.Question, error) {
	sys := fmt.Sprintf(`You are an expert technical interviewer.
Generate %d interview questions for the job below. Every question must be
SELF-EXPLANATORY: the candidate is told why it is asked and what a good answer
shows.

Rules:
- "category" is exactly one of: %q, %q, %q.
- "prompt" is the question the candidate hears (one sentence, no preamble).
- "context" explains why this question is asked FOR THIS JOB.
- "expectation" states what a strong answer demonstrates.
- "skill" is the related required skill when the question targets one, else "".
- "priority": 1 = probes a gap in the requirements, 2 = verifies claimed depth, 3 = general.
- NEVER ask about age, marital or family status, religion, politics, gender,
  ethnicity, disability or any protected characteristic.
- Cover the required skills first, then behavioral and situational judgement.

Output JSON exactly matching this schema:
{
  "questions": [
    {"category": "string", "prompt": "string", "context": "string", "expectation": "string", "skill": "string", "priority": 1}
  ]
}`, questionSetCount, domain.QuestionCategoryTechnical, domain.QuestionCategoryBehavioral, domain.QuestionCategorySituational)

	type questionInput struct {
		Title            string   `json:"title"`
		Description      string   `json:"description"`
		RequiredSkills   []string `json:"required_skills"`
		MinExperience    int      `json:"min_experience"`
		Responsibilities []string `json:"responsibilities,omitempty"`
		Requirements     []string `json:"requirements,omitempty"`
	}
	req, err := json.Marshal(questionInput{
		Title:            job.Title,
		Description:      job.Description,
		RequiredSkills:   job.RequiredSkills,
		MinExperience:    job.MinExperience,
		Responsibilities: job.Responsibilities,
		Requirements:     job.Requirements,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal question input: %w", err)
	}

	out, err := w.llmClient.StructuredOutput(ctx, llm.StructuredRequest{
		OrgID:  job.OrgID.String(),
		System: sys,
		User:   string(req),
		Schema: &questionSetSchema{},
	})
	if err != nil {
		return nil, err
	}
	return decodeLLMQuestions(out)
}

// questionSetSchema / questionSchema — the provider-facing shape. Kept separate
// from domain.Question so provider quirks (casing, extra fields, absent
// values) are normalised in one place before validation.
type questionSetSchema struct {
	Questions []questionSchema `json:"questions"`
}

type questionSchema struct {
	Category    string `json:"category"`
	Prompt      string `json:"prompt"`
	Context     string `json:"context"`
	Expectation string `json:"expectation"`
	Skill       string `json:"skill"`
	Priority    int    `json:"priority"`
}

// decodeLLMQuestions normalises, bias-filters and validates provider output.
//
// Failure semantics (per plan): malformed JSON, an unexpected shape, an empty
// set or a set whose every question is biased are all reported as errors so
// asynq retries — nothing is persisted in between.
func decodeLLMQuestions(out any) ([]domain.Question, error) {
	schema, err := coerceQuestionSchema(out)
	if err != nil {
		return nil, err
	}
	if len(schema.Questions) == 0 {
		return nil, fmt.Errorf("%w: provider returned no questions", domain.ErrQuestionSetInvalid)
	}

	questions := make([]domain.Question, 0, len(schema.Questions))
	for i, q := range schema.Questions {
		candidate := domain.Question{
			Category:    domain.NormalizeQuestionCategory(q.Category),
			Prompt:      strings.TrimSpace(q.Prompt),
			Context:     strings.TrimSpace(q.Context),
			Expectation: strings.TrimSpace(q.Expectation),
			Skill:       strings.ToLower(strings.TrimSpace(q.Skill)),
			Priority:    q.Priority,
		}
		if err := candidate.Validate(); err != nil {
			return nil, fmt.Errorf("question %d: %w", i, err)
		}
		// Bias filter gates EVERY LLM-authored field BEFORE storage (I6): the
		// provider writes the framing too, so a protected-class probe in
		// Context/Expectation/Skill is as unacceptable as one in the prompt.
		// The interview generator gates again at read time (defense in depth).
		//
		// Semantics: FILTER the individual question, fail only when nothing
		// valid remains (below) — one bad field must not kill the whole set
		// and burn retries over content we can simply drop.
		if gensvc.IsBiased(candidate.Prompt) ||
			gensvc.IsBiased(candidate.Context) ||
			gensvc.IsBiased(candidate.Expectation) ||
			gensvc.IsBiased(candidate.Skill) {
			continue
		}
		questions = append(questions, candidate)
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("%w: every generated question was bias-filtered", domain.ErrQuestionSetInvalid)
	}
	return questions, nil
}

// coerceQuestionSchema accepts both the typed schema pointer (providers that
// unmarshal into Schema) and any generic JSON shape, without ever guessing at
// missing fields.
func coerceQuestionSchema(out any) (*questionSetSchema, error) {
	switch v := out.(type) {
	case nil:
		return nil, fmt.Errorf("%w: provider returned no payload", domain.ErrQuestionSetInvalid)
	case *questionSetSchema:
		if v == nil {
			return nil, fmt.Errorf("%w: provider returned no payload", domain.ErrQuestionSetInvalid)
		}
		return v, nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable payload: %v", domain.ErrQuestionSetInvalid, err)
	}
	var schema questionSetSchema
	if err := json.Unmarshal(b, &schema); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrQuestionSetInvalid, err)
	}
	return &schema, nil
}

// fail records the failure on the job (operator-visible error state) and
// returns the error that decides asynq's next move: retry while the budget
// lasts, permanent skip once it is exhausted. question_set itself is never
// touched here — a failed generation leaves the fallback path intact.
func (w *QuestionWorker) fail(ctx context.Context, p GenerateQuestionSetPayload, id uuid.UUID, cause error) error {
	msg := safeQuestionSetError(cause)
	if uerr := db.RunInTx(ctx, w.pool, p.OrgID, func(txCtx context.Context) error {
		return storeQuestionSetError(txCtx, id, msg)
	}); uerr != nil {
		w.log.Error().Err(uerr).Str("job_id", p.JobID).Msg("question set failure state not persisted")
	}
	w.log.Error().Err(cause).Str("job_id", p.JobID).Msg("generate_question_set failed")

	if retriesExhausted(ctx) {
		// Terminal: error state stands, the job keeps falling back to the
		// deterministic templates at interview start.
		return asynq.SkipRetry
	}
	return fmt.Errorf("generate question set for job %s: %w", p.JobID, cause)
}

// retriesExhausted reports whether asynq has no attempts left. Outside a real
// asynq worker (unit tests) the counters are unknown and the answer is false,
// keeping failures retryable by default.
func retriesExhausted(ctx context.Context) bool {
	retried, ok := asynq.GetRetryCount(ctx)
	if !ok {
		return false
	}
	maxRetry, ok := asynq.GetMaxRetry(ctx)
	if !ok || maxRetry <= 0 {
		return false
	}
	return retried >= maxRetry
}

func safeQuestionSetError(cause error) string {
	msg := strings.TrimSpace(cause.Error())
	if msg == "" {
		msg = "question set generation failed"
	}
	if len(msg) > questionSetErrorMaxLen {
		msg = msg[:questionSetErrorMaxLen]
	}
	return msg
}

// ---- column-scoped question_set access ----
//
// These live in the worker (not on JobRepository) on purpose: the Job
// aggregate does not carry the stored set yet, and the only writes are
// single-column ones that must not participate in the full-row Update.

// LoadQuestionSet returns the raw jobs.question_set payload for a job. Must be
// called inside a tenant transaction (db.RunInTx) so RLS applies. A NULL
// column yields a nil payload — feed it to domain.SelectQuestionSet to get the
// "use templates instead" decision.
func LoadQuestionSet(ctx context.Context, jobID uuid.UUID) (json.RawMessage, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	var raw []byte
	err := tx.WithContext(ctx).Raw(`SELECT question_set FROM jobs WHERE id = $1`, jobID).Row().Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// LoadQuestionSetError returns the recorded generation failure ("" = none).
func LoadQuestionSetError(ctx context.Context, jobID uuid.UUID) (string, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return "", db.ErrNoTx
	}
	var msg *string
	err := tx.WithContext(ctx).Raw(`SELECT question_set_error FROM jobs WHERE id = $1`, jobID).Row().Scan(&msg)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", nil
	}
	return *msg, nil
}

// storeQuestionSet writes the validated set and clears any previous failure.
// updatedAtNanos is the job row version stamped at enqueue (J7): the write is
// conditional on it so a stale generation (job republished/edited while the
// LLM call ran) fails the WHERE and cannot overwrite the newer cycle's set.
// Returns stale=true when the row advanced past the payload version.
func storeQuestionSet(ctx context.Context, jobID uuid.UUID, raw json.RawMessage, updatedAtNanos int64) (bool, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return false, db.ErrNoTx
	}
	res := tx.WithContext(ctx).Exec(
		`UPDATE jobs SET question_set = $1, question_set_error = NULL, updated_at = NOW() WHERE id = $2 AND updated_at = $3`,
		string(raw), jobID, time.Unix(0, updatedAtNanos))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 0, nil
}

// storeQuestionSetError records the failure reason WITHOUT touching
// question_set (no partial persistence of a rejected generation).
func storeQuestionSetError(ctx context.Context, jobID uuid.UUID, msg string) error {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return db.ErrNoTx
	}
	return tx.WithContext(ctx).Exec(
		`UPDATE jobs SET question_set_error = $1, updated_at = NOW() WHERE id = $2`, msg, jobID).Error
}

// enqueueQuestionSet — question-set generation is triggered by publish (D5).
// The TaskID embeds the job's row version (updated_at, D7 / finding I8): a
// purely deterministic ID stays occupied in asynq's archived set (~90d) after
// retry exhaustion, and the ErrTaskIDConflict below maps to success — so an
// unpublish→republish would silently never regenerate. updated_at is bumped by
// every transition, so each publish cycle mints a fresh ID. Result-gated
// idempotency lives in the worker (SelectQuestionSet guard): a stored set is
// still never regenerated or rewritten.
func questionSetTaskID(jobID uuid.UUID, updatedAt time.Time) string {
	return TaskGenerateQuestionSet + ":" + jobID.String() + ":" + strconv.FormatInt(updatedAt.UnixNano(), 10)
}

func (s *JobService) enqueueQuestionSet(ctx context.Context, orgID, jobID uuid.UUID, updatedAt time.Time, published bool) error {
	if s.q == nil || !published {
		return nil
	}
	_, err := s.q.Enqueue(ctx, TaskGenerateQuestionSet,
		GenerateQuestionSetPayload{JobID: jobID.String(), OrgID: orgID.String(), UpdatedAt: updatedAt.UnixNano()},
		asynq.TaskID(questionSetTaskID(jobID, updatedAt)), asynq.MaxRetry(5))
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil // this exact transition was already enqueued
	}
	return err
}
