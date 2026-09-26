package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	billingapp "github.com/intivai/backend/internal/billing/application"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	iamapi "github.com/intivai/backend/internal/iam/api"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type BillingHandler struct {
	svc *billingapp.BillingService
}

func NewBillingHandler(svc *billingapp.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

type CheckoutRequest struct {
	Plan       string `json:"plan"`
	Credits    int    `json:"credits,omitempty"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

type PortalRequest struct {
	ReturnURL string `json:"return_url"`
}

// GetSummary — GET /billing/summary
func (h *BillingHandler) GetSummary(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	summary, err := h.svc.GetSummary(c.UserContext(), actor.OrgID)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, summary)
}

// CreateCheckout — POST /billing/checkout
func (h *BillingHandler) CreateCheckout(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	var req CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid checkout payload"))
	}
	plan := billingdomain.Plan(strings.ToLower(strings.TrimSpace(req.Plan)))
	session, err := h.svc.CreateCheckout(c.UserContext(), actor.OrgID, plan, req.Credits, req.SuccessURL, req.CancelURL)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, session)
}

// CreatePortal — POST /billing/portal
func (h *BillingHandler) CreatePortal(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	var req PortalRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid portal payload"))
	}
	url, err := h.svc.CreatePortal(c.UserContext(), actor.OrgID, req.ReturnURL)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, fiber.Map{"url": url})
}
