package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreateCheckoutCommand struct {
	OrgID      uuid.UUID `json:"org_id"`
	Plan       Plan      `json:"plan"`
	Credits    int       `json:"credits,omitempty"`
	SuccessURL string    `json:"success_url"`
	CancelURL  string    `json:"cancel_url"`
}

type CheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type WebhookEvent struct {
	Type           string     `json:"type"`
	CustomerID     string     `json:"customer_id"`
	SubscriptionID string     `json:"subscription_id"`
	Plan           Plan       `json:"plan"`
	Status         string     `json:"status"`
	PeriodStart    *time.Time `json:"period_start,omitempty"`
	PeriodEnd      *time.Time `json:"period_end,omitempty"`
}

// PaymentGateway abstracts the merchant provider (Stripe, Paddle, etc.).
type PaymentGateway interface {
	CreateCustomer(ctx context.Context, orgID uuid.UUID, orgName, email string) (string, error)
	CreateCheckoutSession(ctx context.Context, cmd CreateCheckoutCommand) (*CheckoutSession, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
	VerifyWebhookSignature(payload []byte, signature string) (*WebhookEvent, error)
}
