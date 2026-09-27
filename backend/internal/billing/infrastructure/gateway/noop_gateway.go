package gateway

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
)

// NoopPaymentGateway provides an interface-ready stub when merchant keys are not yet configured.
type NoopPaymentGateway struct{}

func NewNoopPaymentGateway() *NoopPaymentGateway {
	return &NoopPaymentGateway{}
}

func (g *NoopPaymentGateway) CreateCustomer(ctx context.Context, orgID uuid.UUID, orgName, email string) (string, error) {
	return fmt.Sprintf("cus_stub_%s", orgID.String()[:8]), nil
}

func (g *NoopPaymentGateway) CreateCheckoutSession(ctx context.Context, cmd billingdomain.CreateCheckoutCommand) (*billingdomain.CheckoutSession, error) {
	return &billingdomain.CheckoutSession{
		ID:  fmt.Sprintf("cs_stub_%s", uuid.NewString()[:8]),
		URL: fmt.Sprintf("%s?session_id=stub_session", cmd.SuccessURL),
	}, nil
}

func (g *NoopPaymentGateway) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return returnURL, nil
}

func (g *NoopPaymentGateway) VerifyWebhookSignature(payload []byte, signature string) (*billingdomain.WebhookEvent, error) {
	return nil, fmt.Errorf("payment gateway unconfigured: setup API credentials to verify webhooks")
}
