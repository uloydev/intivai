package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// questionStubProvider — the deterministic mock provider pattern used by the
// rubric worker tests: call counting makes idempotency observable, and the
// fixture can be swapped to exercise malformed/biased output.
type questionStubProvider struct {
	calls    int
	fail     bool
	fixture  any
	failWith error
}

func (s *questionStubProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, errors.New("unused")
}
func (s *questionStubProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (s *questionStubProvider) StructuredOutput(ctx context.Context, req llm.StructuredRequest) (any, error) {
	s.calls++
	if s.fail {
		if s.failWith != nil {
			return nil, s.failWith
		}
		return nil, llm.ErrUpstream
	}
	if s.fixture != nil {
		return s.fixture, nil
	}
	return &questionSetSchema{Questions: []questionSchema{
		{
			Category:    "technical",
			Prompt:      "Walk me through the last Go service you took to production.",
			Context:     "This role owns Go services end to end, so shipping history is the signal we need.",
			Expectation: "Concrete architecture, the failure modes handled, and the tradeoffs accepted.",
			Skill:       "go",
			Priority:    1,
		},
		{
			Category:    "situational",
			Prompt:      "Errors spike halfway through a release. What is your first move?",
			Context:     "On-call is shared across the team, so incident instincts matter from day one.",
			Expectation: "Stop the bleeding, communicate, then chase root cause with data.",
			Priority:    2,
		},
	}}, nil
}
func (s *questionStubProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (s *questionStubProvider) CountTokens(text string) int { return 0 }

func newTestQuestionWorker(pool *gorm.DB, prov llm.Provider) *QuestionWorker {
	client := llm.NewClient(prov, nil, nil, 1)
	return NewQuestionWorker(pool, jobrepo.NewPostgresJobRepo(pool), client, zerolog.Nop())
}

func requireQuestionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

// seedPublishedJob — publish is the generation trigger, so the fixture job is
// published from the start.
func seedPublishedJob(t *testing.T, pool *gorm.DB) (orgID string, jobID uuid.UUID) {
	t.Helper()
	orgID = uuid.NewString()
	jobID = uuid.New()
	err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", "q"+orgID[:8]).Error; err != nil {
			return err
		}
		return tx.Exec(
			`INSERT INTO jobs (id, org_id, title, description, required_skills, min_experience, status, is_published, created_at)
			 VALUES ($1,$2,'Go Engineer','Own our Go services','["Go","Kubernetes"]',3,'active',TRUE,NOW())`,
			jobID, orgID).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	return orgID, jobID
}

func questionTask(orgID string, jobID uuid.UUID) *asynq.Task {
	payload, _ := json.Marshal(GenerateQuestionSetPayload{JobID: jobID.String(), OrgID: orgID})
	return asynq.NewTask(TaskGenerateQuestionSet, payload)
}

func readQuestionSet(t *testing.T, pool *gorm.DB, orgID string, jobID uuid.UUID) json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		var err error
		raw, err = LoadQuestionSet(tctx, jobID)
		return err
	})
	if err != nil {
		t.Fatalf("load question set: %v", err)
	}
	return raw
}

func readQuestionSetError(t *testing.T, pool *gorm.DB, orgID string, jobID uuid.UUID) string {
	t.Helper()
	var msg string
	err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		var err error
		msg, err = LoadQuestionSetError(tctx, jobID)
		return err
	})
	if err != nil {
		t.Fatalf("load question set error: %v", err)
	}
	return msg
}

// Happy path: publish -> worker generates -> versioned descriptive set stored
// on the job, and the exact same set is retrievable afterwards.
func TestQuestionWorkerHappyPathStoresDescriptiveSet(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	prov := &questionStubProvider{}
	worker := newTestQuestionWorker(pool, prov)

	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if prov.calls != 1 {
		t.Fatalf("llm calls = %d, want 1", prov.calls)
	}

	raw := readQuestionSet(t, pool, orgID, jobID)
	set, ok := jobdomain.SelectQuestionSet(raw)
	if !ok {
		t.Fatalf("stored set unusable: %s", raw)
	}
	if set.Version != jobdomain.QuestionSetVersion || set.Source != jobdomain.QuestionSourceLLM {
		t.Fatalf("set metadata = v%d/%s", set.Version, set.Source)
	}
	if len(set.Questions) != 2 {
		t.Fatalf("questions = %d, want 2", len(set.Questions))
	}
	for _, q := range set.Questions {
		if q.Prompt == "" || q.Context == "" || q.Expectation == "" {
			t.Fatalf("stored question not self-explanatory: %+v", q)
		}
	}
	if msg := readQuestionSetError(t, pool, orgID, jobID); msg != "" {
		t.Fatalf("error state set on success: %q", msg)
	}
}

// INVARIANT (D5): the set is generated ONCE PER JOB, not per candidate. A
// re-run of the task must not call the LLM again and must not rewrite the set,
// so every candidate interviewed for this job hears identical questions.
func TestQuestionSetIsGeneratedOncePerJobNotPerCandidate(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	prov := &questionStubProvider{}
	worker := newTestQuestionWorker(pool, prov)

	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	first := readQuestionSet(t, pool, orgID, jobID)

	// Second candidate applies -> same job, task replayed: no regeneration.
	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("replay handle: %v", err)
	}
	second := readQuestionSet(t, pool, orgID, jobID)

	if prov.calls != 1 {
		t.Fatalf("llm calls = %d, want 1 (set is per job, generated once)", prov.calls)
	}
	if string(first) != string(second) {
		t.Fatalf("stored set mutated between candidates:\n%s\n%s", first, second)
	}

	set, ok := jobdomain.SelectQuestionSet(second)
	if !ok {
		t.Fatalf("set unusable: %s", second)
	}
	a := gensvc.GenerateQuestions(gensvc.CandidateProfile{Skills: []string{"Go"}}, gensvc.JobRequirements{Title: "Go Engineer"}, 5, set)
	b := gensvc.GenerateQuestions(gensvc.CandidateProfile{Skills: []string{"Rust"}}, gensvc.JobRequirements{Title: "Go Engineer"}, 5, set)
	if len(a) == 0 || len(a) != len(b) {
		t.Fatalf("per-candidate divergence at interview start: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("question %d differs per candidate: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// Failure path: LLM error -> retryable, error state recorded, set ABSENT
// (never partially persisted) and the deterministic templates still carry the
// interview at start time.
func TestQuestionWorkerLLMFailureLeavesNoSetAndTemplatesStillWork(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	prov := &questionStubProvider{fail: true}
	worker := newTestQuestionWorker(pool, prov)

	err := worker.handle(context.Background(), questionTask(orgID, jobID))
	if err == nil {
		t.Fatal("expected an error so asynq retries")
	}
	if errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("provider outage must stay retryable, got %v", err)
	}

	raw := readQuestionSet(t, pool, orgID, jobID)
	if _, ok := jobdomain.SelectQuestionSet(raw); ok {
		t.Fatalf("set persisted despite failure: %s", raw)
	}
	if len(raw) != 0 && string(raw) != "null" {
		t.Fatalf("partial persistence: %s", raw)
	}
	if msg := readQuestionSetError(t, pool, orgID, jobID); msg == "" {
		t.Fatal("terminal error state not recorded on the job")
	}

	// Interview start with no usable set: deterministic templates take over.
	set, _ := jobdomain.SelectQuestionSet(raw)
	qs := gensvc.GenerateQuestions(
		gensvc.CandidateProfile{Skills: []string{"Go"}},
		gensvc.JobRequirements{Title: "Go Engineer", RequiredSkills: []string{"Go", "Kubernetes"}},
		5, set)
	if len(qs) == 0 {
		t.Fatal("fallback templates produced no questions — interview cannot start")
	}
	for _, q := range qs {
		if q.Context == "" || q.Expectation == "" {
			t.Fatalf("fallback question not self-explanatory: %+v", q)
		}
	}

	// Recovery: a later retry succeeds and clears the error state.
	prov.fail = false
	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
	if _, ok := jobdomain.SelectQuestionSet(readQuestionSet(t, pool, orgID, jobID)); !ok {
		t.Fatal("set missing after recovery")
	}
	if msg := readQuestionSetError(t, pool, orgID, jobID); msg != "" {
		t.Fatalf("error state not cleared after recovery: %q", msg)
	}
}

// Malformed LLM JSON: retryable failure, nothing persisted (the plan forbids
// partial persistence of an unvalidated set).
func TestQuestionWorkerMalformedOutputIsRetryableAndPersistsNothing(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	prov := &questionStubProvider{fixture: &questionSetSchema{Questions: []questionSchema{
		{Category: "technical", Prompt: "Prompt without any framing"},
	}}}
	worker := newTestQuestionWorker(pool, prov)

	err := worker.handle(context.Background(), questionTask(orgID, jobID))
	if err == nil {
		t.Fatal("expected malformed output to fail")
	}
	if errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("malformed output must stay retryable, got %v", err)
	}
	raw := readQuestionSet(t, pool, orgID, jobID)
	if len(raw) != 0 && string(raw) != "null" {
		t.Fatalf("partial persistence of malformed set: %s", raw)
	}
}

// Drafts are not generated for: publish is the only trigger, so a draft task
// is a no-op that never spends an LLM call.
func TestQuestionWorkerSkipsUnpublishedJob(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		return tx.Exec(`UPDATE jobs SET is_published = FALSE WHERE id = $1`, jobID).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	prov := &questionStubProvider{}
	worker := newTestQuestionWorker(pool, prov)
	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("draft handle: %v", err)
	}
	if prov.calls != 0 {
		t.Fatalf("llm invoked for a draft job: %d calls", prov.calls)
	}
}

// Tenant discipline mirrors the rubric worker: an org-B task naming an org-A
// job sees nothing through RLS, spends no LLM call, writes nothing.
func TestQuestionWorkerCrossOrgIsolation(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgA, jobID := seedPublishedJob(t, pool)
	orgB := uuid.NewString()
	err := db.RunInTx(context.Background(), pool, orgB, func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgB, "b", "qb"+orgB[:8]).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	prov := &questionStubProvider{}
	worker := newTestQuestionWorker(pool, prov)
	if err := worker.handle(context.Background(), questionTask(orgB, jobID)); err != nil && !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("cross-org task must be permanent-skip, got %v", err)
	}
	if prov.calls != 0 {
		t.Fatalf("llm invoked across tenant boundary: %d calls", prov.calls)
	}
	if raw := readQuestionSet(t, pool, orgA, jobID); len(raw) != 0 && string(raw) != "null" {
		t.Fatalf("cross-org write leaked: %s", raw)
	}
}

// ---- pure decode/validation unit tests (no database required) ----

func TestDecodeLLMQuestionsRejectsUnusableShapes(t *testing.T) {
	cases := []struct {
		name string
		out  any
	}{
		{"nil output", nil},
		{"wrong type", "not a question set"},
		{"no questions", &questionSetSchema{}},
		{"missing expectation", &questionSetSchema{Questions: []questionSchema{
			{Category: "technical", Prompt: "p", Context: "c"},
		}}},
		{"missing context", &questionSetSchema{Questions: []questionSchema{
			{Category: "technical", Prompt: "p", Expectation: "e"},
		}}},
		{"unknown category", &questionSetSchema{Questions: []questionSchema{
			{Category: "trivia", Prompt: "p", Context: "c", Expectation: "e"},
		}}},
		{"every question biased", &questionSetSchema{Questions: []questionSchema{
			{Category: "behavioral", Prompt: "How old are you?", Context: "c", Expectation: "e"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeLLMQuestions(tc.out); err == nil {
				t.Fatal("expected decode failure (retryable, nothing persisted)")
			}
		})
	}
}

// A provider that returns a decoded map (real providers do) must work too.
func TestDecodeLLMQuestionsAcceptsGenericJSONShape(t *testing.T) {
	out := map[string]any{"questions": []any{map[string]any{
		"category":    "Technical",
		"prompt":      "Describe a service you scaled.",
		"context":     "The role owns scaling decisions.",
		"expectation": "Numbers, bottlenecks found, and the fix.",
		"priority":    float64(1),
	}}}
	qs, err := decodeLLMQuestions(out)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(qs) != 1 {
		t.Fatalf("questions = %d, want 1", len(qs))
	}
	if qs[0].Category != jobdomain.QuestionCategoryTechnical {
		t.Fatalf("category not normalised: %q", qs[0].Category)
	}
}

// Bias filter runs on LLM output before persistence: clean questions survive,
// the biased one never reaches the stored set.
func TestDecodeLLMQuestionsDropsBiasedQuestions(t *testing.T) {
	qs, err := decodeLLMQuestions(&questionSetSchema{Questions: []questionSchema{
		{Category: "technical", Prompt: "Describe your last Go service.", Context: "c", Expectation: "e"},
		{Category: "behavioral", Prompt: "Are you married with children?", Context: "c", Expectation: "e"},
	}})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(qs) != 1 {
		t.Fatalf("questions = %d, want 1 (biased dropped)", len(qs))
	}
	if gensvc.IsBiased(qs[0].Prompt) {
		t.Fatalf("biased question persisted: %q", qs[0].Prompt)
	}
}

// INVARIANT (finding I6): EVERY LLM-authored text field is bias-gated, not
// just the prompt. The provider writes Context/Expectation/Skill too — a
// protected-class probe smuggled into the framing must never reach the stored
// set a candidate is asked from.
func TestDecodeLLMQuestionsBiasGatesEveryTextField(t *testing.T) {
	cases := []struct {
		name    string
		biased  func(questionSchema) string // returns the field carrying biased content
		content string
	}{
		{"prompt", func(q questionSchema) string { return q.Prompt }, "How old are you?"},
		{"context", func(q questionSchema) string { return q.Context }, "We ask about family plans to gauge commitment."},
		{"expectation", func(q questionSchema) string { return q.Expectation }, "A strong answer discusses their religion."},
		{"skill", func(q questionSchema) string { return q.Skill }, "political science"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qs, err := decodeLLMQuestions(&questionSetSchema{Questions: []questionSchema{
				{
					Category:    "technical",
					Prompt:      "Describe your last Go service.",
					Context:     "The role owns Go services end to end.",
					Expectation: "Concrete architecture and tradeoffs.",
					Skill:       "go",
				},
				func() questionSchema {
					q := questionSchema{
						Category:    "behavioral",
						Prompt:      "Describe a conflict you resolved.",
						Context:     "Cross-team work needs de-escalation skills.",
						Expectation: "Empathy, facts, and a durable resolution.",
						Skill:       "communication",
					}
					switch tc.name {
					case "prompt":
						q.Prompt = tc.content
					case "context":
						q.Context = tc.content
					case "expectation":
						q.Expectation = tc.content
					case "skill":
						q.Skill = tc.content
					}
					return q
				}(),
			}})
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(qs) != 1 {
				t.Fatalf("questions = %d, want 1 — biased %s field leaked into the stored set", len(qs), tc.name)
			}
			if got := tc.biased(questionSchema{
				Prompt:      qs[0].Prompt,
				Context:     qs[0].Context,
				Expectation: qs[0].Expectation,
				Skill:       qs[0].Skill,
			}); gensvc.IsBiased(got) {
				t.Fatalf("stored question still carries biased content in %s: %q", tc.name, got)
			}
		})
	}
}

// One bad field filters ONE question; the rest of the set survives. Failing
// the whole generation over a single filtered item would hand asynq a retry
// loop for content we can simply drop.
func TestDecodeLLMQuestionsKeepsCleanSiblingsOfBiasedQuestion(t *testing.T) {
	qs, err := decodeLLMQuestions(&questionSetSchema{Questions: []questionSchema{
		{Category: "technical", Prompt: "Describe your last Go service.", Context: "Scaling story.", Expectation: "Numbers.", Skill: "go"},
		{Category: "situational", Prompt: "Errors spike mid-release. First move?", Context: "Are you married matters here.", Expectation: "Triage order.", Priority: 2},
	}})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(qs) != 1 {
		t.Fatalf("questions = %d, want 1 (only the biased sibling filtered)", len(qs))
	}
}

// Bias filtering at the store boundary end-to-end: a biased CONTEXT field
// (not just the prompt) filters that question out and only the clean sibling
// reaches jobs.question_set (finding I6).
func TestQuestionWorkerFiltersBiasedContextBeforeStore(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	prov := &questionStubProvider{fixture: &questionSetSchema{Questions: []questionSchema{
		{Category: "technical", Prompt: "Describe your last Go service.", Context: "Scaling story.", Expectation: "Numbers.", Skill: "go"},
		{Category: "situational", Prompt: "Errors spike mid-release. First move?", Context: "Do you have children at home?", Expectation: "Triage order.", Priority: 2},
	}}}
	worker := newTestQuestionWorker(pool, prov)

	if err := worker.handle(context.Background(), questionTask(orgID, jobID)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	set, ok := jobdomain.SelectQuestionSet(readQuestionSet(t, pool, orgID, jobID))
	if !ok {
		t.Fatal("usable set missing after bias-filtered generation")
	}
	if len(set.Questions) != 1 {
		t.Fatalf("stored questions = %d, want 1 — biased framing reached the column", len(set.Questions))
	}
	if gensvc.IsBiased(set.Questions[0].Context + set.Questions[0].Prompt + set.Questions[0].Expectation + set.Questions[0].Skill) {
		t.Fatalf("biased content persisted: %+v", set.Questions[0])
	}
}

func TestSafeQuestionSetErrorTruncates(t *testing.T) {
	msg := safeQuestionSetError(errors.New(strings.Repeat("x", 1000)))
	if len(msg) == 0 || len(msg) > 300 {
		t.Fatalf("error message length = %d, want 1..300", len(msg))
	}
}

// J7: storeQuestionSet is CAS'd on the enqueue-time row version (updated_at).
// A generation whose payload version predates the current row — job
// edited/republished while the LLM call ran — must not overwrite the newer
// cycle's set.
func TestStoreQuestionSetStaleVersionDoesNotOverwrite(t *testing.T) {
	pool := requireQuestionTestDB(t)
	orgID, jobID := seedPublishedJob(t, pool)
	ctx := context.Background()

	var currentVersion time.Time
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Raw(`SELECT updated_at FROM jobs WHERE id = $1`, jobID).Scan(&currentVersion).Error
	}); err != nil {
		t.Fatal(err)
	}

	newerRaw := json.RawMessage(`{"version":2,"questions":[{"idx":1,"prompt":"fresh","category":"technical","skill":"Go","priority":1}]}`)
	// 1) Newer generator (matching version) writes first.
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		stale, err := storeQuestionSet(tctx, jobID, newerRaw, currentVersion.UnixNano())
		if err != nil {
			return err
		}
		if stale {
			t.Fatal("matching-version store reported stale")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// 2) Stale generator (older version) tries to overwrite — must be rejected.
	staleVersion := currentVersion.Add(-time.Hour)
	staleRaw := json.RawMessage(`{"version":1,"questions":[{"idx":1,"prompt":"stale","category":"technical","skill":"Go","priority":1}]}`)
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		stale, err := storeQuestionSet(tctx, jobID, staleRaw, staleVersion.UnixNano())
		if err != nil {
			return err
		}
		if !stale {
			t.Fatal("stale-version store overwrote a newer set (CAS missing)")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// 3) Row must still hold the newer set.
	got := readQuestionSet(t, pool, orgID, jobID)
	if !strings.Contains(string(got), "fresh") || strings.Contains(string(got), "stale") {
		t.Fatalf("question_set after stale attempt = %s, want the newer generation only", got)
	}
}
