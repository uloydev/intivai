package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// ConfirmReview record-repo: captures the 7-field payload for assertions.
// Embedded nil interface: only ConfirmReview is invoked by this flow.
type stubConfirmRepo struct {
	cvdomain.CandidateRepository
	token      string
	structured []byte
	name       string
	email      string
	orgID      uuid.UUID
	candID     uuid.UUID
}

func (s *stubConfirmRepo) ConfirmReview(ctx context.Context, token string, structured []byte, name, email string) (uuid.UUID, uuid.UUID, error) {
	s.token, s.structured, s.name, s.email = token, structured, name, email
	if s.orgID == uuid.Nil {
		return uuid.Nil, uuid.Nil, nil
	}
	return s.orgID, s.candID, nil
}

type stubConfirmAppRepo struct {
	scrdomain.ApplicationRepository
	apps []*scrdomain.Application
}

func (s *stubConfirmAppRepo) ByCandidate(ctx context.Context, orgID, candidateID uuid.UUID) ([]*scrdomain.Application, error) {
	return s.apps, nil
}

type stubConfirmStore struct{}

func (s *stubConfirmStore) Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	return nil
}
func (s *stubConfirmStore) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (s *stubConfirmStore) Delete(ctx context.Context, path string) error { return nil }

type stubConfirmEnq struct {
	count int
}

func (s *stubConfirmEnq) Enqueue(ctx context.Context, task string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.count++
	return &asynq.TaskInfo{ID: "t1"}, nil
}

func newConfirmApp(repo *stubConfirmRepo, apps []*scrdomain.Application) *fiber.App {
	app := fiber.New()
	// The confirm service fan-out runs RunInTx against the service pool; a
	// pre-attached in-context tx makes RunInTx reuse it (nil pool untouched).
	app.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(db.WithTx(context.Background(), &gorm.DB{}))
		return c.Next()
	})
	h := NewCVHandler(cvapp.NewCVService(repo, &stubConfirmAppRepo{apps: apps}, &stubConfirmStore{}, &stubConfirmEnq{}, nil), 10)
	app.Post("/api/v1/public/candidate-review/:token/confirm", h.ConfirmProfile)
	return app
}

func doConfirm(app *fiber.App, token, body string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/candidate-review/"+token+"/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		return nil
	}
	return resp
}

func TestConfirmProfileHandlerAcceptsSevenFieldPayload(t *testing.T) {
	orgID, candID := uuid.New(), uuid.New()
	repo := &stubConfirmRepo{orgID: orgID, candID: candID}
	app := newConfirmApp(repo, nil)

	body := `{"name":"Jane Doe","email":"jane@x.io","skills":["Go","SQL"],"experience_years":5,"education":"MSc","certifications":["AWS"],"summary":"backend engineer"}`
	resp := doConfirm(app, "tok-ok", body)
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %v, want 200", resp.StatusCode)
	}
	var payload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	if payload["data"].(map[string]any)["status"] != "confirmed" {
		t.Fatalf("response = %v, want status confirmed", payload)
	}

	if repo.token != "tok-ok" || repo.name != "Jane Doe" || repo.email != "jane@x.io" {
		t.Fatalf("repo got token=%q name=%q email=%q", repo.token, repo.name, repo.email)
	}
	var rd struct {
		Skills          []string `json:"skills"`
		ExperienceYears float64  `json:"experience_years"`
		Education       string   `json:"education"`
		Certifications  []string `json:"certifications"`
		Summary         string   `json:"summary"`
	}
	if err := json.Unmarshal(repo.structured, &rd); err != nil {
		t.Fatalf("structured not json: %v", err)
	}
	if len(rd.Skills) != 2 || rd.ExperienceYears != 5 || rd.Education != "MSc" || len(rd.Certifications) != 1 || rd.Summary != "backend engineer" {
		t.Fatalf("resume fields lost: %+v", rd)
	}
	var m map[string]any
	_ = json.Unmarshal(repo.structured, &m)
	if _, ok := m["name"]; ok {
		t.Fatal("name leaked into cv_structured (identity column, not resume dimension)")
	}
}

func TestConfirmProfileHandlerBadJSON(t *testing.T) {
	app := newConfirmApp(&stubConfirmRepo{orgID: uuid.New(), candID: uuid.New()}, nil)
	resp := doConfirm(app, "tok", `{"name":`)
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %v, want 400", resp.StatusCode)
	}
}

func TestConfirmProfileHandlerValidation(t *testing.T) {
	app := newConfirmApp(&stubConfirmRepo{orgID: uuid.New(), candID: uuid.New()}, nil)

	cases := []struct {
		name string
		body string
	}{
		{"empty name", `{"name":"","email":"j@x.io","skills":[],"experience_years":0,"education":"","certifications":[],"summary":""}`},
		{"invalid email", `{"name":"Jane","email":"not-an-email","skills":[],"experience_years":0,"education":"","certifications":[],"summary":""}`},
		{"oversized summary", `{"name":"Jane","email":"j@x.io","skills":[],"experience_years":0,"education":"","certifications":[],"summary":"` + strings.Repeat("x", 2001) + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doConfirm(app, "tok", tc.body)
			if resp == nil || resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %v, want 400", resp.StatusCode)
			}
			var payload map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&payload)
			if payload["code"] != "CANDIDATE_PROFILE_INVALID" {
				t.Fatalf("code = %v, want CANDIDATE_PROFILE_INVALID", payload["code"])
			}
		})
	}
}

func TestConfirmProfileHandlerNotFound(t *testing.T) {
	// uuid.Nil org (stub default) → service maps to NotFoundError → 404.
	app := newConfirmApp(&stubConfirmRepo{}, nil)
	body := `{"name":"Jane","email":"j@x.io","skills":[],"experience_years":0,"education":"","certifications":[],"summary":""}`
	resp := doConfirm(app, "tok-bad", body)
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %v, want 404", resp.StatusCode)
	}
}
