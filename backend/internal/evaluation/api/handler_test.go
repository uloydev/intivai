package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	evalapp "github.com/intivai/backend/internal/evaluation/application"
	iamapi "github.com/intivai/backend/internal/iam/api"
	iamapp "github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

func seedInterviewWithEval(t *testing.T, pool *gorm.DB, orgID, appID, jobID, candID, ivID string) {
	t.Helper()
	ctx := context.Background()
	err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgID, "t", "eh" + orgID[:8]}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgID, "Go Engineer", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgID, "Jane Doe", "jane@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgID, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		raw, _ := json.Marshal(struct {
			Questions []map[string]any `json:"questions"`
			Answers   []map[string]any `json:"answers"`
		}{
			Questions: []map[string]any{{"idx": 1, "content": "Tell me about Go", "category": "technical"}},
			Answers:   []map[string]any{{"idx": 1, "content": "Five years of Go", "answered_at": "2026-08-10T00:00:00Z"}},
		})
		eval, _ := json.Marshal(map[string]any{"overall_score": 80, "recommendation": "proceed"})
		return tx.Exec(`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx, context_version, evaluation, created_at)
			VALUES ($1,$2,'chat','completed',$3,1,0,$4,NOW())`, ivID, appID, raw, eval).Error
	})
	if err != nil {
		t.Fatal(err)
	}
}

// seedHandlerScenario — org + job + candidate + passed application + completed
// interview with evaluation. Returns handler + real auth token.
func seedHandlerScenario(t *testing.T) (*fiber.App, string, uuid.UUID, uuid.UUID) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	orgID := uuid.New()
	appID, jobID, candID, ivID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	seedInterviewWithEval(t, pool, orgID.String(), appID.String(), jobID.String(), candID.String(), ivID.String())
	_ = ctx

	svc := evalapp.NewEvaluationService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), scrrepo.NewPostgresApplicationRepo(pool),
		cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool), nil)
	handler := NewEvaluationHandler(svc)
	jwt := auth.NewJWTProvider("test-secret-eval-handler")

	app := fiber.New()
	app.Get("/interviews/:id", iamapi.AuthMiddleware(jwt), handler.GetInterview)
	app.Get("/candidates/:id/report", iamapi.AuthMiddleware(jwt), handler.GetCandidateReport)

	token, err := jwt.Issue(uuid.New(), orgID, "admin", iamapp.TokenTypeAuth, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	return app, token, ivID, candID
}

func doAuthed(app *fiber.App, token, path string) *http.Response {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(r, -1)
	if err != nil {
		return nil
	}
	return resp
}

func TestGetInterviewHandler(t *testing.T) {
	app, token, ivID, _ := seedHandlerScenario(t)

	resp := doAuthed(app, token, "/interviews/"+ivID.String())
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("status = %v, want 200", resp.StatusCode)
	}
	var out struct {
		Data struct {
			InterviewID uuid.UUID       `json:"interview_id"`
			Status      string          `json:"status"`
			Candidate   json.RawMessage `json:"candidate"`
			Evaluation  json.RawMessage `json:"evaluation"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Data.InterviewID != ivID || out.Data.Status != "completed" || len(out.Data.Evaluation) == 0 {
		t.Fatalf("detail = %+v", out.Data)
	}
	if len(out.Data.Candidate) == 0 {
		t.Fatal("candidate context missing")
	}
}

func TestGetInterviewHandlerRejectsBadIDAndAnon(t *testing.T) {
	app, token, _, _ := seedHandlerScenario(t)

	if resp := doAuthed(app, token, "/interviews/not-a-uuid"); resp == nil || resp.StatusCode != 400 {
		t.Fatalf("bad id status = %v, want 400", resp.StatusCode)
	}
	r := httptest.NewRequest(http.MethodGet, "/interviews/"+uuid.NewString(), nil)
	if resp, _ := app.Test(r, -1); resp.StatusCode != 401 {
		t.Fatalf("anonymous status = %v, want 401", resp.StatusCode)
	}
}

func TestGetCandidateReportHandler(t *testing.T) {
	app, token, _, candID := seedHandlerScenario(t)

	resp := doAuthed(app, token, "/candidates/"+candID.String()+"/report")
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("status = %v, want 200", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Candidate struct {
				Name string `json:"name"`
			} `json:"candidate"`
			Interviews []json.RawMessage `json:"interviews"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Data.Candidate.Name == "" || len(out.Data.Interviews) != 1 {
		t.Fatalf("report = %+v", out.Data)
	}
}
