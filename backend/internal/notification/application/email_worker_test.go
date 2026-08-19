package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/notification/application"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type mockMailer struct{}

func (m *mockMailer) SendApplicationConfirmation(ctx context.Context, to, candName, jobTitle string) error {
	return nil
}

func (m *mockMailer) SendInterviewInvitation(ctx context.Context, to, candName, jobTitle, inviteURL string) error {
	return nil
}

func (m *mockMailer) SendScorecardReady(ctx context.Context, to, candName, jobTitle string, score float64, recommendation, reportURL string) error {
	return nil
}

func (m *mockMailer) SendCandidateLoginOTP(ctx context.Context, to, otp, magicLink string) error {
	return nil
}

func (m *mockMailer) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	return nil
}

func (m *mockMailer) SendCandidateReview(ctx context.Context, to, candidateName, inviteURL string) error {
	return nil
}

func (m *mockMailer) SendCandidateDecision(ctx context.Context, to, name, jobTitle, decision, portalURL string) error {
	return nil
}

func TestEmailWorker_SkipRetry(t *testing.T) {
	worker := application.NewEmailWorker(&mockMailer{}, zerolog.Nop())
	mux := asynq.NewServeMux()
	worker.Register(mux)

	// Test 1: Invalid JSON payload -> SkipRetry
	task1 := asynq.NewTask(application.TaskSendEmail, []byte(`{invalid-json`), asynq.MaxRetry(5))
	err := mux.ProcessTask(context.Background(), task1)
	require.Error(t, err)
	require.True(t, errors.Is(err, asynq.SkipRetry), "expected SkipRetry for invalid json payload")

	// Test 2: Unknown email type -> SkipRetry
	payload := application.SendEmailPayload{
		Type: "unknown_type_for_test",
		To:   "test@example.com",
	}
	b, _ := json.Marshal(payload)
	task2 := asynq.NewTask(application.TaskSendEmail, b, asynq.MaxRetry(5))
	err = mux.ProcessTask(context.Background(), task2)
	require.Error(t, err)
	require.True(t, errors.Is(err, asynq.SkipRetry), "expected SkipRetry for unknown email type")
}
