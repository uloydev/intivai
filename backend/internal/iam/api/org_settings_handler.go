package api

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamapp "github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

// Candidate Q&A cap bounds (B4/D3): migration 030 declares the bare column,
// bounds live here so the handler rejects outside 0..50 before touching the DB.
const (
	minCandidateQALimit = 0
	maxCandidateQALimit = 50
)

// OrgSettingsHandler owns org-level settings endpoints (e.g. candidate Q&A cap).
type OrgSettingsHandler struct {
	repo iamdomain.IAMRepository
}

func NewOrgSettingsHandler(repo iamdomain.IAMRepository) *OrgSettingsHandler {
	return &OrgSettingsHandler{repo: repo}
}

type candidateQALimitRequest struct {
	CandidateQALimit *int `json:"candidate_qa_limit"`
}

// UpdateCandidateQALimit handles PUT /api/v1/orgs/:orgId/settings/candidate-qa-limit.
// Org-admin only; orgId is pinned to the actor's org (tenant isolation).
func (h *OrgSettingsHandler) UpdateCandidateQALimit(c *fiber.Ctx) error {
	actor, err := RequireActor(c)
	if err != nil {
		return httpapi.Error(c, err)
	}
	orgID, err := uuid.Parse(c.Params("orgId"))
	if err != nil || orgID != actor.OrgID {
		return httpapi.Error(c, sharederr.NewDomainError("FORBIDDEN", "org mismatch"))
	}
	if err := iamapp.Authorize(actor, iamdomain.RoleAdmin); err != nil {
		return httpapi.Error(c, err)
	}
	var req candidateQALimitRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	if req.CandidateQALimit == nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "candidate_qa_limit is required"))
	}
	if *req.CandidateQALimit < minCandidateQALimit || *req.CandidateQALimit > maxCandidateQALimit {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_QA_LIMIT",
			fmt.Sprintf("candidate_qa_limit must be %d..%d", minCandidateQALimit, maxCandidateQALimit)))
	}
	if err := h.repo.UpdateOrgCandidateQALimit(c.UserContext(), orgID, *req.CandidateQALimit); err != nil {
		if errors.Is(err, iamdomain.ErrNotFound) {
			return httpapi.Error(c, sharederr.NewNotFoundError("org", orgID.String()))
		}
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, fiber.Map{"candidate_qa_limit": *req.CandidateQALimit})
}
