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
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
)

type stubOrgRepo struct {
	iamdomain.IAMRepository
	org         *iamdomain.Org
	updateLimit int
	updateErr   error
}

func (s *stubOrgRepo) GetOrg(_ context.Context, id uuid.UUID) (*iamdomain.Org, error) {
	if s.org == nil {
		return nil, iamdomain.ErrNotFound
	}
	return s.org, nil
}

func (s *stubOrgRepo) UpdateOrgCandidateQALimit(_ context.Context, orgID uuid.UUID, limit int) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateLimit = limit
	q := limit
	if s.org != nil {
		s.org.CandidateQALimit = &q
	}
	return nil
}

func newOrgSettingsApp(repo iamdomain.IAMRepository, role string, orgID uuid.UUID) *fiber.App {
	app := fiber.New()
	h := NewOrgSettingsHandler(repo)
	app.Put("/api/v1/orgs/:orgId/settings/candidate-qa-limit",
		AuthMiddleware(stubTokens{claims: &application.Claims{
			Subject: uuid.New(), OrgID: orgID, Role: role, Type: application.TokenTypeAuth,
		}}),
		h.UpdateCandidateQALimit)
	return app
}

func doPutQALimit(app *fiber.App, orgID uuid.UUID, body string) (int, map[string]any) {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/"+orgID.String()+"/settings/candidate-qa-limit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer x")
	resp, err := app.Test(req, -1)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	return resp.StatusCode, payload
}

func TestOrgSettingsHandlerSetsQALimit(t *testing.T) {
	orgID := uuid.New()
	repo := &stubOrgRepo{org: &iamdomain.Org{}}
	app := newOrgSettingsApp(repo, string(iamdomain.RoleAdmin), orgID)

	status, payload := doPutQALimit(app, orgID, `{"candidate_qa_limit":25}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", status, payload)
	}
	if repo.updateLimit != 25 {
		t.Fatalf("repo received limit %d, want 25", repo.updateLimit)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data wrapper missing: %v", payload)
	}
	if data["candidate_qa_limit"] != float64(25) {
		t.Fatalf("candidate_qa_limit = %v, want 25", data["candidate_qa_limit"])
	}
	if repo.org.CandidateQALimit == nil || *repo.org.CandidateQALimit != 25 {
		t.Fatalf("org candidate limit = %v, want 25", repo.org.CandidateQALimit)
	}
}

func TestOrgSettingsHandlerQALimitBounds(t *testing.T) {
	orgID := uuid.New()
	repo := &stubOrgRepo{org: &iamdomain.Org{}}
	app := newOrgSettingsApp(repo, string(iamdomain.RoleAdmin), orgID)

	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"candidate_qa_limit":0}`, http.StatusOK},
		{`{"candidate_qa_limit":50}`, http.StatusOK},
		{`{"candidate_qa_limit":51}`, http.StatusBadRequest},
		{`{"candidate_qa_limit":-1}`, http.StatusBadRequest},
		{`{}`, http.StatusBadRequest},
		{`{"candidate_qa_limit":"10"}`, http.StatusBadRequest},
	} {
		status, payload := doPutQALimit(app, orgID, tc.body)
		if status != tc.status {
			t.Fatalf("body %s: status = %d, want %d (body %v)", tc.body, status, tc.status, payload)
		}
	}
}

func TestOrgSettingsHandlerRejectsNonAdmin(t *testing.T) {
	orgID := uuid.New()
	app := newOrgSettingsApp(&stubOrgRepo{}, string(iamdomain.RoleRecruiter), orgID)

	status, _ := doPutQALimit(app, orgID, `{"candidate_qa_limit":10}`)
	if status != http.StatusForbidden {
		t.Fatalf("recruiter: status = %d, want 403", status)
	}
}

func TestOrgSettingsHandlerRejectsOrgMismatch(t *testing.T) {
	app := newOrgSettingsApp(&stubOrgRepo{}, string(iamdomain.RoleAdmin), uuid.New())

	status, _ := doPutQALimit(app, uuid.New(), `{"candidate_qa_limit":10}`)
	if status != http.StatusForbidden {
		t.Fatalf("org mismatch: status = %d, want 403", status)
	}
}

func TestOrgSettingsHandlerNotFound(t *testing.T) {
	orgID := uuid.New()
	repo := &stubOrgRepo{updateErr: iamdomain.ErrNotFound}
	app := newOrgSettingsApp(repo, string(iamdomain.RoleAdmin), orgID)

	status, _ := doPutQALimit(app, orgID, `{"candidate_qa_limit":10}`)
	if status != http.StatusNotFound {
		t.Fatalf("missing org: status = %d, want 404", status)
	}
}
