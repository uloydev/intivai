package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	iamapp "github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
)

type stubJobRepo struct {
	jobs             map[uuid.UUID]*jobdomain.Job
	update           int
	getByID          int
	getByIDForUpdate int
}

func (r *stubJobRepo) Create(ctx context.Context, job *jobdomain.Job) error {
	r.jobs[job.ID] = job
	return nil
}
func (r *stubJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*jobdomain.Job, error) {
	r.getByID++
	j, ok := r.jobs[id]
	if !ok {
		return nil, jobdomain.ErrNotFound
	}
	return j, nil
}
func (r *stubJobRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*jobdomain.Job, error) {
	r.getByIDForUpdate++
	return r.GetByID(ctx, id)
}
func (r *stubJobRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*jobdomain.Job, error) {
	return nil, nil
}
func (r *stubJobRepo) List(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return nil, nil
}
func (r *stubJobRepo) ListActive(ctx context.Context, orgID uuid.UUID) ([]*jobdomain.Job, error) {
	return nil, nil
}
func (r *stubJobRepo) Update(ctx context.Context, job *jobdomain.Job) error {
	r.jobs[job.ID] = job
	r.update++
	// Simulate the row's ON UPDATE timestamp: every transition bumps it.
	job.UpdatedAt = time.Now().UTC()
	return nil
}
func (r *stubJobRepo) UpdateRubric(ctx context.Context, id uuid.UUID, rubric json.RawMessage) error {
	return nil
}

// recordingQueue captures enqueued tasks incl. their TaskID option so tests
// can assert on deterministic-ID behaviour without Redis.
type recordingQueue struct {
	tasks []capturedEnqueue
}

type capturedEnqueue struct {
	taskType string
	taskID   string // "" when no TaskID option was passed
}

func taskIDFromOptions(opts []asynq.Option) string {
	for _, o := range opts {
		type typed interface {
			Type() asynq.OptionType
			Value() interface{}
		}
		if t, ok := o.(typed); ok && t.Type() == asynq.TaskIDOpt {
			if v, ok := t.Value().(string); ok {
				return v
			}
		}
	}
	return ""
}

func (q *recordingQueue) Enqueue(ctx context.Context, jobType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	q.tasks = append(q.tasks, capturedEnqueue{taskType: jobType, taskID: taskIDFromOptions(opts)})
	return &asynq.TaskInfo{}, nil
}

func TestUpdateWeightsLockedAfterPublish(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	job.IsPublished = true
	repo.jobs[job.ID] = job

	svc := NewJobService(repo, nil)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}

	weights := map[string]float64{
		"skills_match": 0.6, "experience_years": 0.2, "semantic_match": 0.1,
		"education": 0.05, "certifications": 0.05,
	}
	_, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, ScoringWeights: weights})
	if err == nil {
		t.Fatal("published job accepted weight edit")
	}
	var de *sharederrors.DomainError
	if !errors.As(err, &de) || de.Code != "JOB_WEIGHTS_LOCKED" {
		t.Fatalf("err = %v, want JOB_WEIGHTS_LOCKED", err)
	}
	if repo.update != 0 {
		t.Fatal("repo.Update must not run on a locked edit")
	}
}

// Finding I9 / D8: the weights-freeze check is check-then-act against the
// job's publish state. Without a row lock, a concurrent publish toggle can
// land between the read and the Update and the freeze passes stale. The
// update path must therefore read the row FOR UPDATE inside the tenant tx.
func TestUpdatePathReadsJobRowForUpdate(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	repo.jobs[job.ID] = job

	svc := NewJobService(repo, nil)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}

	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID}); err != nil {
		t.Fatal(err)
	}
	if repo.getByIDForUpdate != 1 {
		t.Fatalf("GetByIDForUpdate calls = %d, want 1 — freeze check races concurrent publishes", repo.getByIDForUpdate)
	}
}

// Finding I8 / D7: the deterministic TaskID `generate_question_set:<jobID>`
// stays occupied in asynq's archived set (~90d) after retry exhaustion, and an
// enqueue conflict is mapped to success — so unpublish→republish silently
// never regenerates. The ID must therefore embed the job's updated_at (bumped
// on every transition), while the worker keeps its result-gated idempotency.
func TestRepublishEnqueuesFreshQuestionSetTaskID(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	repo.jobs[job.ID] = job

	q := &recordingQueue{}
	svc := NewJobService(repo, q)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}
	publish := true

	// First publish.
	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, IsPublished: &publish}); err != nil {
		t.Fatal(err)
	}
	// Unpublish (recruiter edits the JD).
	unpublish := false
	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, IsPublished: &unpublish}); err != nil {
		t.Fatal(err)
	}
	// Republish — must enqueue a FRESH task id even though one was already
	// minted for this job (archived-task conflict would swallow it silently).
	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, IsPublished: &publish}); err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, task := range q.tasks {
		if task.taskType == TaskGenerateQuestionSet {
			ids = append(ids, task.taskID)
		}
	}
	if len(ids) < 2 {
		t.Fatalf("question-set enqueues = %d, want >= 3 transitions recorded %v", len(ids), q.tasks)
	}
	last := ids[len(ids)-1]
	prev := ids[len(ids)-2]
	if last == "" {
		t.Fatal("republish enqueue carried no TaskID option")
	}
	if last == prev {
		t.Fatalf("republish reused occupied TaskID %q — regeneration is silently swallowed", last)
	}
}

func TestUpdateWeightsAllowedWhileDraft(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	repo.jobs[job.ID] = job

	svc := NewJobService(repo, nil)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}

	weights := map[string]float64{
		"skills_match": 0.6, "experience_years": 0.2, "semantic_match": 0.1,
		"education": 0.05, "certifications": 0.05,
	}
	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, ScoringWeights: weights}); err != nil {
		t.Fatal(err)
	}
	if repo.update != 1 {
		t.Fatalf("update calls = %d, want 1", repo.update)
	}
	if repo.jobs[job.ID].ScoringWeights["skills_match"] != 0.6 {
		t.Fatal("weights not persisted")
	}
}

func TestUpdateKeepsWeightsWhenOmitted(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	repo.jobs[job.ID] = job

	svc := NewJobService(repo, nil)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}

	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID}); err != nil {
		t.Fatal(err)
	}
	if repo.jobs[job.ID].ScoringWeights != nil {
		t.Fatal("omitted weights set something")
	}
}

// J8: enqueues must NOT execute inside the mutation path — they are deferred
// to after the transaction commits. The tenant-tx middleware attaches a hook
// registry to the request ctx; with a registry, registering a hook must not
// execute until RunAfterCommit fires after the commit succeeded.
func TestUpdateDefersEnqueuesUntilAfterCommit(t *testing.T) {
	ctx := db.WithAfterCommit(context.Background())
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	repo.jobs[job.ID] = job

	q := &recordingQueue{}
	svc := NewJobService(repo, q)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}
	publish := true

	if _, err := svc.Update(ctx, actor, UpdateJobCommand{JobID: job.ID, IsPublished: &publish}); err != nil {
		t.Fatal(err)
	}
	if len(q.tasks) != 0 {
		t.Fatalf("enqueues executed BEFORE commit: %d tasks, want 0 (deferred)", len(q.tasks))
	}
	db.RunAfterCommit(ctx)
	if len(q.tasks) == 0 {
		t.Fatal("enqueues never fired after RunAfterCommit")
	}
}

// INVARIANT (finding I14 / D9): a recorded generation failure must be
// operator-visible through the job GET surface, otherwise the only trace of a
// dead question set is a missing column nobody can see.
func TestGetExposesQuestionSetError(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	repo := &stubJobRepo{jobs: map[uuid.UUID]*jobdomain.Job{}}
	job, _ := jobdomain.NewJob(orgID, "t", "d", nil, 0)
	job.QuestionSetError = "question set invalid: every generated question was bias-filtered"
	repo.jobs[job.ID] = job

	svc := NewJobService(repo, nil)
	actor := iamapp.AuthContext{UserID: uuid.New(), OrgID: orgID, Role: string(iamdomain.RoleRecruiter)}

	result, err := svc.Get(ctx, actor, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.QuestionSetError != job.QuestionSetError {
		t.Fatalf("question_set_error = %q, want %q", result.QuestionSetError, job.QuestionSetError)
	}

	// Wire contract: snake_case key on the JSON payload (DTO rule).
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["question_set_error"]; !ok {
		t.Fatalf("json payload missing question_set_error key: %s", b)
	}

	// Healthy jobs carry no error state — the key stays absent.
	clean, _ := jobdomain.NewJob(orgID, "clean", "d", nil, 0)
	repo.jobs[clean.ID] = clean
	cleanResult, err := svc.Get(ctx, actor, clean.ID)
	if err != nil {
		t.Fatal(err)
	}
	cb, err := json.Marshal(cleanResult)
	if err != nil {
		t.Fatal(err)
	}
	var cleanPayload map[string]json.RawMessage
	if err := json.Unmarshal(cb, &cleanPayload); err != nil {
		t.Fatal(err)
	}
	if _, ok := cleanPayload["question_set_error"]; ok {
		t.Fatalf("healthy job leaked question_set_error key: %s", cb)
	}
}
