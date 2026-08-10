package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func mustInterview(t *testing.T, at time.Time) *Interview {
	t.Helper()
	iv, err := NewInterview(uuid.New(), uuid.New(), []Question{
		{Idx: 1, Content: "Q1", Category: "technical"},
		{Idx: 2, Content: "Q2", Category: "behavioral"},
	}, at.Add(time.Hour), FrozenClock(at))
	if err != nil {
		t.Fatal(err)
	}
	return iv
}

func TestNewInterviewValidates(t *testing.T) {
	if _, err := NewInterview(uuid.New(), uuid.New(), nil, time.Now().Add(time.Hour), SystemClock()); err == nil {
		t.Fatal("empty questions accepted")
	}
}

func TestStartTransition(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	if err := iv.Start(); err != nil {
		t.Fatal(err)
	}
	if iv.Status != StatusInProgress || iv.StartedAt == nil {
		t.Fatalf("status = %s", iv.Status)
	}
	// Resume: second Start is a no-op, not an error.
	if err := iv.Start(); err != nil {
		t.Fatalf("resume start: %v", err)
	}
}

func TestStartRejectedFromTerminalStates(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	_ = iv.Start()
	_ = iv.Complete()
	if err := iv.Start(); err == nil {
		t.Fatal("completed interview restarted")
	}
}

func TestAnswerFlowAndCursor(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	_ = iv.Start()

	next := iv.NextQuestion()
	if next == nil || next.Content != "Q1" {
		t.Fatalf("first next = %+v", next)
	}
	if err := iv.Answer("answer one"); err != nil {
		t.Fatal(err)
	}
	if iv.LastQuestionIdx != 1 || len(iv.Answers) != 1 {
		t.Fatalf("cursor = %d answers = %d", iv.LastQuestionIdx, len(iv.Answers))
	}
	next = iv.NextQuestion()
	if next == nil || next.Content != "Q2" {
		t.Fatalf("second next = %+v", next)
	}
	if err := iv.Answer("answer two"); err != nil {
		t.Fatal(err)
	}
	if iv.NextQuestion() != nil {
		t.Fatal("question after last answered")
	}
}

func TestAnswerRejectedOutsideProgress(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	if err := iv.Answer("x"); err == nil {
		t.Fatal("answer before start accepted")
	}
	_ = iv.Start()
	_ = iv.Complete()
	if err := iv.Answer("x"); err == nil {
		t.Fatal("answer after complete accepted")
	}
}

func TestIdleTimeoutWithFrozenClock(t *testing.T) {
	base := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, base)
	_ = iv.Start()
	if iv.IsIdle(5 * time.Minute) {
		t.Fatal("idle immediately after activity")
	}
	// Advance clock past the 5-min window without touching.
	iv.clock = FrozenClock(base.Add(6 * time.Minute))
	if !iv.IsIdle(5 * time.Minute) {
		t.Fatal("not idle after 6 minutes")
	}
	// Touch resets.
	iv.Touch()
	if iv.IsIdle(5 * time.Minute) {
		t.Fatal("idle right after touch")
	}
}

func TestExpiry(t *testing.T) {
	base := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, base) // expires base+1h
	_ = iv.Start()
	iv.ExpireIfNeeded()
	if iv.Status != StatusInProgress {
		t.Fatal("expired before deadline")
	}
	iv.clock = FrozenClock(base.Add(2 * time.Hour))
	iv.ExpireIfNeeded()
	if iv.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", iv.Status)
	}
	// Completed interviews never expire retroactively.
	iv2 := mustInterview(t, base)
	_ = iv2.Start()
	_ = iv2.Complete()
	iv2.clock = FrozenClock(base.Add(2 * time.Hour))
	iv2.ExpireIfNeeded()
	if iv2.Status != StatusCompleted {
		t.Fatalf("completed interview expired: %s", iv2.Status)
	}
}

func TestResumeIdx(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	_ = iv.Start()
	_ = iv.Answer("a1")
	if iv.ResumeIdx() != 2 {
		t.Fatalf("resume idx = %d, want 2", iv.ResumeIdx())
	}
}
