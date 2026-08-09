package application

import (
	"context"
	"testing"
)

// Consent gate: StartInterview refuses without consent; GiveConsent is
// idempotent and token-bound; the chat then starts normally.
func TestConsentGate(t *testing.T) {
	s := seedInterviewApp(t, "active")
	// Build the interview WITHOUT the seed helper's consent step.
	created, err := s.svc.CreateInterview(context.Background(), iamActor(s.orgID.String(), "admin"), CreateInterviewCommand{ApplicationID: s.appID, QuestionCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// No consent yet → start rejected.
	err = s.svc.StartInterview(ctx, s.orgID.String(), created.InterviewID)
	if err == nil || err.Error() != "candidate consent must be recorded before the interview" {
		t.Fatalf("start without consent err = %v, want CONSENT_REQUIRED", err)
	}

	// Wrong token → rejected.
	if err := s.svc.GiveConsent(ctx, created.InterviewID, "not-a-token"); err == nil {
		t.Fatal("invalid token accepted")
	}

	// Consent with the invitation token → start works; second consent idempotent.
	if err := s.svc.GiveConsent(ctx, created.InterviewID, created.Token); err != nil {
		t.Fatalf("consent: %v", err)
	}
	if err := s.svc.GiveConsent(ctx, created.InterviewID, created.Token); err != nil {
		t.Fatalf("second consent: %v", err)
	}
	if err := s.svc.StartInterview(ctx, s.orgID.String(), created.InterviewID); err != nil {
		t.Fatalf("start after consent: %v", err)
	}
}
