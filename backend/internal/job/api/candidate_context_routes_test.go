package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamapi "github.com/intivai/backend/internal/iam/api"
	iamapp "github.com/intivai/backend/internal/iam/application"
	iamauth "github.com/intivai/backend/internal/iam/infrastructure/auth"
	jobapi "github.com/intivai/backend/internal/job/api"
	jobapp "github.com/intivai/backend/internal/job/application"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/pkg/db"
)

// Finding I2: the candidate-context endpoints must be registered on the
// authed group exactly like the other job routes (auth middleware + tenant tx,
// admin/recruiter enforced inside the service). An unauthenticated request
// proves REGISTRATION here:
//   - unregistered path  -> fiber router answers 404 before any middleware runs
//   - registered path    -> auth middleware answers 401 first
func TestCandidateContextRoutesAreRegisteredOnAuthedGroup(t *testing.T) {
	app := fiber.New()
	v1 := app.Group("/api/v1")
	authed := v1.Group("", iamapi.AuthMiddleware(iamauth.NewJWTProvider("test-secret")), func(c *fiber.Ctx) error {
		return c.Next()
	})

	candidateContextHandler := jobapi.NewCandidateContextHandler(nil)
	candidateContextHandler.RegisterRoutes(authed)

	cases := []struct{ method, path string }{
		{"GET", "/api/v1/jobs/6f9619ff-8b86-d011-b42d-00c04fc964ff/candidate-context"},
		{"PUT", "/api/v1/jobs/6f9619ff-8b86-d011-b42d-00c04fc964ff/candidate-context"},
		{"POST", "/api/v1/jobs/6f9619ff-8b86-d011-b42d-00c04fc964ff/candidate-context/suggest"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		if resp.StatusCode == fiber.StatusNotFound {
			t.Fatalf("%s %s returned 404 — route not wired on the authed group", tc.method, tc.path)
		}
	}
}

// The authed group's middleware chain runs BEFORE the handler: an
// unauthenticated request must be rejected with 401 (not leak to the handler).
func TestCandidateContextRoutesRequireAuth(t *testing.T) {
	app := fiber.New()
	v1 := app.Group("/api/v1")

	tokens := iamauth.NewJWTProvider("test-secret")
	authed := v1.Group("", iamapi.AuthMiddleware(tokens), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusTeapot)
	})
	jobapi.NewCandidateContextHandler(nil).RegisterRoutes(authed)

	req := httptest.NewRequest("PUT", "/api/v1/jobs/6f9619ff-8b86-d011-b42d-00c04fc964ff/candidate-context", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unauthenticated PUT got %d, want 401", resp.StatusCode)
	}
}

// staticTokens — auth middleware seam returning fixed claims (no JWT dance).
type staticTokens struct {
	claims *iamapp.Claims
}

func (s staticTokens) Issue(_ uuid.UUID, _ uuid.UUID, _, _ string, _ time.Duration, _ iamapp.TokenExtra) (string, error) {
	return "", errors.New("unused")
}
func (s staticTokens) Parse(_ string) (*iamapp.Claims, error) { return s.claims, nil }

// stubSuggestLLM — deterministic draft for the suggest endpoint.
type stubSuggestLLM struct{}

func (stubSuggestLLM) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "Remote-friendly. Compensation is competitive."}, nil
}
func (stubSuggestLLM) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan string, error) {
	return nil, errors.New("unused")
}
func (stubSuggestLLM) StructuredOutput(_ context.Context, _ llm.StructuredRequest) (interface{}, error) {
	return nil, errors.New("unused")
}
func (stubSuggestLLM) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("unused")
}
func (stubSuggestLLM) CountTokens(_ string) int { return 0 }

// seedCandidateContextJob removed — org+job seeding happens inline in the
// full-chain test below.

// Finding I2 end-to-end: through the SAME registration main.go uses
// (auth -> tenant tx -> handler), a recruiter can save and read the per-job
// candidate context, suggest returns an AI draft, and lower roles are
// rejected by the service's authorize rail.
func TestCandidateContextRoutesFullChain(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID, jobID := uuid.New(), uuid.New()
	if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return db.ErrNoTx
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "CC Co", "cc"+orgID.String()[:8]).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, is_published, created_at, updated_at)
			VALUES ($1,$2,'Go Engineer','Own services','active',FALSE,NOW(),NOW())`, jobID, orgID).Error
	}); err != nil {
		t.Fatal(err)
	}

	newApp := func(role string) *fiber.App {
		app := fiber.New()
		v1 := app.Group("/api/v1")
		authed := v1.Group("",
			iamapi.AuthMiddleware(staticTokens{claims: &iamapp.Claims{
				Subject: uuid.New(), OrgID: orgID, Role: role, Type: iamapp.TokenTypeAuth,
			}}),
			iamapi.TenantTxMiddleware(pool),
		)
		svc := jobapp.NewCandidateContextService(pool, jobrepo.NewPostgresCandidateContextRepo(pool), jobrepo.NewPostgresJobRepo(pool), stubSuggestLLM{})
		jobapi.NewCandidateContextHandler(svc).RegisterRoutes(authed)
		return app
	}

	do := func(app *fiber.App, method, path, body string) *http.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer static-test-token")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp
	}

	recruiterApp := newApp("recruiter")

	// Save.
	saveResp := do(recruiterApp, "PUT",
		"/api/v1/jobs/"+jobID.String()+"/candidate-context",
		`{"content":"Hybrid — 3 days onsite in Lisbon."}`)
	if saveResp.StatusCode != fiber.StatusOK {
		t.Fatalf("save got %d, want 200", saveResp.StatusCode)
	}
	var saved struct {
		Data struct {
			JobID   string `json:"job_id"`
			Content string `json:"content"`
			Version int    `json:"version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Data.JobID != jobID.String() || saved.Data.Version != 1 ||
		!strings.Contains(saved.Data.Content, "Lisbon") {
		t.Fatalf("save payload mismatch: %+v", saved.Data)
	}

	// Get.
	getResp := do(recruiterApp, "GET", "/api/v1/jobs/"+jobID.String()+"/candidate-context", "")
	if getResp.StatusCode != fiber.StatusOK {
		t.Fatalf("get got %d, want 200", getResp.StatusCode)
	}
	var fetched struct {
		Data struct {
			Content string `json:"content"`
			Version int    `json:"version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&fetched); err != nil {
		t.Fatal(err)
	}
	if fetched.Data.Version != 1 || !strings.Contains(fetched.Data.Content, "Lisbon") {
		t.Fatalf("get payload mismatch: %+v", fetched.Data)
	}

	// Suggest — AI draft returned, nothing persisted.
	sugResp := do(recruiterApp, "POST", "/api/v1/jobs/"+jobID.String()+"/candidate-context/suggest", "")
	if sugResp.StatusCode != fiber.StatusOK {
		t.Fatalf("suggest got %d, want 200", sugResp.StatusCode)
	}
	var suggested struct {
		Data struct {
			Draft string `json:"draft"`
		} `json:"data"`
	}
	if err := json.NewDecoder(sugResp.Body).Decode(&suggested); err != nil {
		t.Fatal(err)
	}
	if suggested.Data.Draft == "" {
		t.Fatal("suggest returned empty draft")
	}

	// Role rail: interviewers cannot touch recruiter-authored context.
	memberResp := do(newApp("interviewer"), "PUT",
		"/api/v1/jobs/"+jobID.String()+"/candidate-context",
		`{"content":"should not persist"}`)
	if memberResp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("interviewer PUT got %d, want 403", memberResp.StatusCode)
	}
}
