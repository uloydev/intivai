package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	billingapi "github.com/intivai/backend/internal/billing/api"
	billingapp "github.com/intivai/backend/internal/billing/application"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	billinggw "github.com/intivai/backend/internal/billing/infrastructure/gateway"
	iamapp "github.com/intivai/backend/internal/iam/application"
	"github.com/stretchr/testify/require"
)

type mockBillingRepo struct {
	billing *billingdomain.OrgBilling
	usage   int
}

func (m *mockBillingRepo) GetOrgBilling(ctx context.Context, orgID uuid.UUID) (*billingdomain.OrgBilling, error) {
	return m.billing, nil
}

func (m *mockBillingRepo) UpdateSubscription(ctx context.Context, billing *billingdomain.OrgBilling) error {
	m.billing = billing
	return nil
}

func (m *mockBillingRepo) GetMonthlyUsage(ctx context.Context, orgID uuid.UUID, since *time.Time) (int, error) {
	return m.usage, nil
}

func (m *mockBillingRepo) DeductCredit(ctx context.Context, orgID uuid.UUID) (bool, error) {
	if m.billing.InterviewCredits > 0 {
		m.billing.InterviewCredits--
		return true, nil
	}
	return false, nil
}

func setupBillingApp(repo billingdomain.BillingRepository, actor *iamapp.AuthContext) (*fiber.App, *billingapi.BillingHandler) {
	app := fiber.New()
	gw := billinggw.NewNoopPaymentGateway()
	svc := billingapp.NewBillingService(nil, repo, gw)
	handler := billingapi.NewBillingHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		if actor != nil {
			c.Locals("auth", *actor)
		}
		return c.Next()
	})

	app.Get("/billing/summary", handler.GetSummary)
	app.Post("/billing/checkout", handler.CreateCheckout)
	app.Post("/billing/portal", handler.CreatePortal)

	return app, handler
}

func TestBillingHandler_GetSummary(t *testing.T) {
	orgID := uuid.New()
	actor := &iamapp.AuthContext{OrgID: orgID, UserID: uuid.New(), Role: "recruiter"}
	repo := &mockBillingRepo{
		billing: &billingdomain.OrgBilling{
			OrgID:            orgID,
			Plan:             billingdomain.PlanPro,
			PlanStatus:       "active",
			InterviewCredits: 5,
		},
		usage: 12,
	}

	t.Run("unauthorized without actor", func(t *testing.T) {
		app, _ := setupBillingApp(repo, nil)
		req := httptest.NewRequest(http.MethodGet, "/billing/summary", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("success", func(t *testing.T) {
		app, _ := setupBillingApp(repo, actor)
		req := httptest.NewRequest(http.MethodGet, "/billing/summary", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var out struct {
			Data struct {
				Plan         string `json:"plan"`
				MonthlyLimit int    `json:"monthly_limit"`
				MonthlyUsage int    `json:"monthly_usage"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		require.Equal(t, "pro", out.Data.Plan)
		require.Equal(t, 1000, out.Data.MonthlyLimit)
		require.Equal(t, 12, out.Data.MonthlyUsage)
	})
}

func TestBillingHandler_CreateCheckout(t *testing.T) {
	orgID := uuid.New()
	actor := &iamapp.AuthContext{OrgID: orgID, UserID: uuid.New(), Role: "recruiter"}
	repo := &mockBillingRepo{
		billing: &billingdomain.OrgBilling{
			OrgID:      orgID,
			Plan:       billingdomain.PlanFree,
			PlanStatus: "active",
		},
	}

	t.Run("invalid payload", func(t *testing.T) {
		app, _ := setupBillingApp(repo, actor)
		req := httptest.NewRequest(http.MethodPost, "/billing/checkout", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid plan", func(t *testing.T) {
		app, _ := setupBillingApp(repo, actor)
		payload := billingapi.CheckoutRequest{
			Plan:       "unlimited_super",
			SuccessURL: "https://intivai.com/success",
			CancelURL:  "https://intivai.com/cancel",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/billing/checkout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid checkout", func(t *testing.T) {
		app, _ := setupBillingApp(repo, actor)
		payload := billingapi.CheckoutRequest{
			Plan:       "pro",
			Credits:    10,
			SuccessURL: "https://intivai.com/success",
			CancelURL:  "https://intivai.com/cancel",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/billing/checkout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestBillingHandler_CreatePortal(t *testing.T) {
	orgID := uuid.New()
	actor := &iamapp.AuthContext{OrgID: orgID, UserID: uuid.New(), Role: "recruiter"}
	repo := &mockBillingRepo{
		billing: &billingdomain.OrgBilling{
			OrgID:            orgID,
			Plan:             billingdomain.PlanPro,
			PlanStatus:       "active",
			StripeCustomerID: "cus_12345",
		},
	}

	t.Run("unauthorized", func(t *testing.T) {
		app, _ := setupBillingApp(repo, nil)
		req := httptest.NewRequest(http.MethodPost, "/billing/portal", bytes.NewBufferString(`{"return_url":"https://intivai.com"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("valid portal session", func(t *testing.T) {
		app, _ := setupBillingApp(repo, actor)
		req := httptest.NewRequest(http.MethodPost, "/billing/portal", bytes.NewBufferString(`{"return_url":"https://intivai.com"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var out struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		require.NotEmpty(t, out.Data.URL)
	})
}
