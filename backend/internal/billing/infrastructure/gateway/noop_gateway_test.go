package gateway_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	"github.com/intivai/backend/internal/billing/infrastructure/gateway"
	"github.com/stretchr/testify/require"
)

func TestNoopPaymentGateway(t *testing.T) {
	gw := gateway.NewNoopPaymentGateway()
	ctx := context.Background()
	orgID := uuid.New()

	cust, err := gw.CreateCustomer(ctx, orgID, "Org", "admin@org.com")
	require.NoError(t, err)
	require.Contains(t, cust, "cus_stub_")

	sess, err := gw.CreateCheckoutSession(ctx, billingdomain.CreateCheckoutCommand{
		OrgID:      orgID,
		Plan:       billingdomain.PlanPro,
		SuccessURL: "http://localhost/success",
	})
	require.NoError(t, err)
	require.NotEmpty(t, sess.ID)
	require.Contains(t, sess.URL, "http://localhost/success")

	portal, err := gw.CreatePortalSession(ctx, cust, "http://localhost/return")
	require.NoError(t, err)
	require.Equal(t, "http://localhost/return", portal)

	_, err = gw.VerifyWebhookSignature([]byte("payload"), "sig")
	require.Error(t, err)
}
