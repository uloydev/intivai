package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
)

// mockLLM satisfies llm.Provider for RequestHuman tests (unused).
type mockLLM struct{}

func (mockLLM) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) { return nil, nil }
func (mockLLM) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan string, error) {
	return nil, nil
}
func (mockLLM) StructuredOutput(_ context.Context, _ llm.StructuredRequest) (any, error) {
	return nil, nil
}
func (mockLLM) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
func (mockLLM) CountTokens(_ string) int                             { return 0 }

// TestRequestHuman_Endpoint verifies the POST /candidate/interviews/:id/request-human
// handler returns correct status codes and persists human_requested = true.
func TestRequestHuman_Endpoint(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}

	// Seed minimal data: org + job + candidate + passed application.
	ctx := t.Context()
	orgID := "rh" + time.Now().Format("150405")
	orgUUID := uuid.New()
	appID, jobID, candID := uuid.New(), uuid.New(), uuid.New()

	err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, []any{orgUUID, "t", orgID}},
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, orgUUID, "Go Eng", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, orgUUID, "Jane", "j@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, orgUUID, candID, jobID}},
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

	jwtProvider := auth.NewJWTProvider("test-rh-secret")
	svc := ivapp.NewInterviewService(pool,
		ivrepo.NewPostgresInterviewRepo(pool), ivrepo.NewPostgresTokenRepo(pool), ivrepo.NewPostgresQuestionBank(pool),
		scrrepo.NewPostgresApplicationRepo(pool), cvrepo.NewPostgresCandidateRepo(pool), jobrepo.NewPostgresJobRepo(pool),
		jobrepo.NewPostgresCandidateContextRepo(pool),
		ctxrepo.NewPostgresContextRepo(pool), nil, jwtProvider, ivdomain.SystemClock(), nil, nil, zerolog.Nop())

	handler := NewChatHandler(svc, mockLLM{}, jwtProvider, zerolog.Nop())
	app := fiber.New()
	app.Post("/candidate/interviews/:id/request-human", handler.RequestHuman)

	// Helper: create an interview + consent + issue a ticket.
	createInterview := func() (string, string) {
		t.Helper()
		created, err := svc.CreateInterview(ctx, actorWith(orgUUID, "admin"), ivapp.CreateInterviewCommand{ApplicationID: appID, QuestionCount: 3})
		if err != nil {
			t.Fatalf("create interview: %v", err)
		}
		if err := svc.GiveConsent(ctx, created.InterviewID, created.Token); err != nil {
			t.Fatalf("consent: %v", err)
		}
		return created.InterviewID.String(), created.Token
	}

	t.Run("invalid interview ID returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"invitation_token": "tok"})
		req := httptest.NewRequest("POST", "/candidate/interviews/not-a-uuid/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})

	t.Run("missing invitation_token returns 400", func(t *testing.T) {
		interviewID, _ := createInterview()
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/candidate/interviews/"+interviewID+"/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})

	t.Run("invalid token returns 400", func(t *testing.T) {
		interviewID, _ := createInterview()
		body, _ := json.Marshal(map[string]string{"invitation_token": "totally-invalid-token"})
		req := httptest.NewRequest("POST", "/candidate/interviews/"+interviewID+"/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})

	t.Run("valid token returns 200 and sets human_requested", func(t *testing.T) {
		interviewID, token := createInterview()
		body, _ := json.Marshal(map[string]string{"invitation_token": token})
		req := httptest.NewRequest("POST", "/candidate/interviews/"+interviewID+"/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		// Verify human_requested = true in the database.
		var requested bool
		err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
			tx, _ := db.TxFrom(tctx)
			return tx.Raw(`SELECT human_requested FROM interviews WHERE id = $1`, interviewID).Row().Scan(&requested)
		})
		if err != nil {
			t.Fatal(err)
		}
		if !requested {
			t.Fatal("human_requested = false, want true")
		}
	})

	// Regression: the Chat page only holds the WS ticket (not the invitation
	// token) — RequestHuman must accept it and still set the flag.
	t.Run("ws ticket token returns 200 and sets human_requested", func(t *testing.T) {
		interviewID, token := createInterview()
		ivUUID, err := uuid.Parse(interviewID)
		if err != nil {
			t.Fatal(err)
		}
		ticket, err := svc.IssueTicket(ctx, ivapp.IssueTicketCommand{InterviewID: ivUUID, InvitationToken: token})
		if err != nil {
			t.Fatalf("issue ticket: %v", err)
		}

		body, _ := json.Marshal(map[string]string{"invitation_token": ticket.Ticket})
		req := httptest.NewRequest("POST", "/candidate/interviews/"+interviewID+"/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var requested bool
		err = db.RunInTx(ctx, pool, orgUUID.String(), func(tctx context.Context) error {
			tx, _ := db.TxFrom(tctx)
			return tx.Raw(`SELECT human_requested FROM interviews WHERE id = $1`, interviewID).Row().Scan(&requested)
		})
		if err != nil {
			t.Fatal(err)
		}
		if !requested {
			t.Fatal("human_requested = false, want true")
		}
	})

	// Cross-interview ticket must be rejected.
	t.Run("ticket for another interview returns 400", func(t *testing.T) {
		idA, _ := createInterview()
		idB, tokenB := createInterview()
		ivB, err := uuid.Parse(idB)
		if err != nil {
			t.Fatal(err)
		}
		ticket, err := svc.IssueTicket(ctx, ivapp.IssueTicketCommand{InterviewID: ivB, InvitationToken: tokenB})
		if err != nil {
			t.Fatalf("issue ticket: %v", err)
		}

		body, _ := json.Marshal(map[string]string{"invitation_token": ticket.Ticket})
		req := httptest.NewRequest("POST", "/candidate/interviews/"+idA+"/request-human", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})
}
