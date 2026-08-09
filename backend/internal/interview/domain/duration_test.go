package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type advancingClock struct{ t time.Time }

func (c *advancingClock) Now() time.Time          { return c.t }
func (c *advancingClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func durationInterview(t *testing.T, clock Clock) *Interview {
	t.Helper()
	iv, err := NewInterview(uuid.New(), uuid.New(), []Question{
		{Idx: 1, Content: "Q1", Category: "technical"},
		{Idx: 2, Content: "Q2", Category: "behavioral"},
	}, clock.Now().Add(24*time.Hour), clock)
	if err != nil {
		t.Fatal(err)
	}
	return iv
}

// Duration cap (Research §2: InterviewMaxDuration = 30 min). The interview
// expires once it has been running for 30 minutes, even with activity.
func TestDurationCapExpires(t *testing.T) {
	clock := &advancingClock{t: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)}
	iv := durationInterview(t, clock)
	if err := iv.Start(); err != nil {
		t.Fatal(err)
	}

	clock.advance(29 * time.Minute)
	iv.ExpireIfNeeded()
	if iv.Status != StatusInProgress {
		t.Fatalf("status = %s, want in_progress before the cap", iv.Status)
	}

	clock.advance(2 * time.Minute) // 31 min total
	iv.ExpireIfNeeded()
	if iv.Status != StatusExpired {
		t.Fatalf("status = %s, want expired after 30-min cap", iv.Status)
	}
}

// Activity must NOT reset the duration clock — the cap is wall-clock from
// Start, not idle-based.
func TestDurationCapIgnoresActivity(t *testing.T) {
	clock := &advancingClock{t: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)}
	iv := durationInterview(t, clock)
	_ = iv.Start()
	clock.advance(31 * time.Minute)
	iv.Touch() // candidate still typing
	iv.ExpireIfNeeded()
	if iv.Status != StatusExpired {
		t.Fatalf("status = %s, want expired despite activity", iv.Status)
	}
}

// The 7-day invitation deadline still applies independently.
func TestDurationCapDoesNotExtendInvitationExpiry(t *testing.T) {
	clock := &advancingClock{t: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)}
	iv, err := NewInterview(uuid.New(), uuid.New(), []Question{
		{Idx: 1, Content: "Q1", Category: "technical"},
	}, clock.Now().Add(1*time.Hour), clock)
	if err != nil {
		t.Fatal(err)
	}
	_ = iv.Start()
	clock.advance(2 * time.Hour)
	iv.ExpireIfNeeded()
	if iv.Status != StatusExpired {
		t.Fatalf("status = %s, want expired via invitation deadline", iv.Status)
	}
}

func TestPerQuestionTimeoutConstant(t *testing.T) {
	if PerQuestionTimeout != 3*time.Minute {
		t.Fatalf("PerQuestionTimeout = %v, want 3m (Research §2)", PerQuestionTimeout)
	}
	if MaxInterviewDuration != 30*time.Minute {
		t.Fatalf("MaxInterviewDuration = %v, want 30m (Research §2)", MaxInterviewDuration)
	}
}
