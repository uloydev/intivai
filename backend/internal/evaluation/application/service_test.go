package application

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/evaluation/domain"
	iamapp "github.com/intivai/backend/internal/iam/application"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
)

func seedEvalScenario(t *testing.T) (*EvaluationService, string, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	appID, jobID, candID, ivID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "es" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go Engineer", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane Doe", "jane@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := json.Marshal(struct {
		Questions []struct {
			Idx      int    `json:"idx"`
			Content  string `json:"content"`
			Category string `json:"category"`
			Skill    string `json:"skill"`
		} `json:"questions"`
		Answers []struct {
			Idx        int    `json:"idx"`
			Content    string `json:"content"`
			AnsweredAt string `json:"answered_at"`
		} `json:"answers"`
	}{
		Questions: []struct {
			Idx      int    `json:"idx"`
			Content  string `json:"content"`
			Category string `json:"category"`
			Skill    string `json:"skill"`
		}{{Idx: 1, Content: "Tell me about Go", Category: "technical", Skill: "go"}},
		Answers: []struct {
			Idx        int    `json:"idx"`
			Content    string `json:"content"`
			AnsweredAt string `json:"answered_at"`
		}{{Idx: 1, Content: "I built services with Go for five years", AnsweredAt: "2026-08-10T00:00:00Z"}},
	})
	eval, _ := json.Marshal(domain.Report{OverallScore: 80, Recommendation: "proceed"})
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx, context_version, evaluation, created_at)
				VALUES ($1,$2,'chat','completed',$3,1,2,$4,NOW())`, []any{ivID, appID, raw, eval}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewEvaluationService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), scrrepo.NewPostgresApplicationRepo(pool),
		cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool))
	return svc, orgID, ivID, candID, appID
}

func TestInterviewDetail(t *testing.T) {
	svc, orgID, ivID, _, _ := seedEvalScenario(t)
	actor := evalActor(orgID, "admin")

	d, err := svc.InterviewDetail(context.Background(), actor, ivID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "completed" || d.ContextVersion != 2 || d.TotalQuestions != 1 {
		t.Fatalf("detail = %+v", d)
	}
	if len(d.Questions) != 1 || len(d.Answers) != 1 {
		t.Fatalf("questions/answers missing: %+v", d)
	}
	if d.Candidate == nil || d.Candidate.Name != "Jane Doe" || d.Job == nil || d.Job.Title != "Go Engineer" {
		t.Fatalf("candidate/job missing: %+v", d)
	}
	if len(d.Evaluation) == 0 {
		t.Fatal("evaluation missing")
	}
}

func TestInterviewDetailCrossOrgForbidden(t *testing.T) {
	svc, _, ivID, _, _ := seedEvalScenario(t)
	other := evalActor(uuid.NewString(), "admin")

	if _, err := svc.InterviewDetail(context.Background(), other, ivID); err == nil {
		t.Fatal("cross-org interview visible")
	}
}

func TestCandidateReport(t *testing.T) {
	svc, orgID, _, candID, _ := seedEvalScenario(t)
	actor := evalActor(orgID, "recruiter")

	r, err := svc.CandidateReport(context.Background(), actor, candID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Candidate.Name != "Jane Doe" {
		t.Fatalf("candidate = %+v", r.Candidate)
	}
	if len(r.Interviews) != 1 || r.Interviews[0].Status != "completed" || len(r.Interviews[0].Evaluation) == 0 {
		t.Fatalf("interviews = %+v", r.Interviews)
	}
}

func evalActor(orgID, role string) iamapp.AuthContext {
	return iamapp.AuthContext{OrgID: uuid.MustParse(orgID), Role: role}
}

func TestListInterviews(t *testing.T) {
	svc, orgID, ivID, candID, _ := seedEvalScenario(t)
	actor := evalActor(orgID, "admin")

	list, err := svc.ListInterviews(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	item := list[0]
	if item.InterviewID != ivID || item.CandidateID != candID {
		t.Fatalf("item = %+v", item)
	}
	if item.CandidateName != "Jane Doe" || item.JobTitle != "Go Engineer" {
		t.Fatalf("enrichment missing: %+v", item)
	}
	if len(item.Evaluation) == 0 {
		t.Fatal("evaluation missing from list row")
	}
}

func TestListInterviewsCrossOrgEmpty(t *testing.T) {
	svc, _, _, _, _ := seedEvalScenario(t)
	other := evalActor(uuid.NewString(), "admin")

	list, err := svc.ListInterviews(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("cross-org list len = %d, want 0", len(list))
	}
}
