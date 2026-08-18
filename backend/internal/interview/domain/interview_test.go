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

func TestInsertProbeAfter(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	_ = iv.Start()
	_ = iv.Answer("a1")

	probe, err := iv.InsertProbeAfter(1, "Can you explain that in more detail?", "technical", "go")
	if err != nil {
		t.Fatalf("insert probe failed: %v", err)
	}
	if probe.Idx != 2 || len(iv.Questions) != 3 {
		t.Fatalf("unexpected probe idx %d or len %d", probe.Idx, len(iv.Questions))
	}
	if iv.Questions[2].Idx != 3 {
		t.Fatalf("subsequent question not renumbered: %d", iv.Questions[2].Idx)
	}

	// Probe on invalid position returns error
	if _, err := iv.InsertProbeAfter(-1, "invalid", "tech", ""); err == nil {
		t.Fatal("expected error on negative index")
	}
}

func TestTranscriptPairs(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)
	_ = iv.Start()
	_ = iv.Answer("Goroutines are lightweight")
	_ = iv.Answer("Team collaboration is key")

	pairs := iv.TranscriptPairs()
	if len(pairs) != 2 {
		t.Fatalf("expected 2 transcript pairs, got %d", len(pairs))
	}
	if pairs[0].Question != "Q1" || pairs[0].Answer != "Goroutines are lightweight" {
		t.Fatalf("mismatched pair 0: %+v", pairs[0])
	}
}

func TestDetermineQuestionArchetype(t *testing.T) {
	tests := []struct {
		q            Question
		wantArch     string
		wantDuration int
	}{
		{
			q:            Question{Idx: 1, Content: "Write a function to invert a binary tree", Category: "technical"},
			wantArch:     ArchetypeCoding,
			wantDuration: TimeLimitCoding,
		},
		{
			q:            Question{Idx: 2, Content: "Design a distributed rate limiter for high throughput", Category: "system_design"},
			wantArch:     ArchetypeSystemDesign,
			wantDuration: TimeLimitSystemDesign,
		},
		{
			q:            Question{Idx: 3, Content: "Tell me about a time you handled a difficult stakeholder", Category: "behavioral"},
			wantArch:     ArchetypeConversational,
			wantDuration: TimeLimitConversational,
		},
	}

	for _, tt := range tests {
		arch, dur := DetermineQuestionArchetype(tt.q)
		if arch != tt.wantArch || dur != tt.wantDuration {
			t.Errorf("DetermineQuestionArchetype(%+v) = (%s, %d), want (%s, %d)", tt.q, arch, dur, tt.wantArch, tt.wantDuration)
		}
	}
}

func TestAnswerWithPacingAndSessionRemaining(t *testing.T) {
	at := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, at)

	// Session remaining before start
	if rem := iv.SessionRemaining(); rem != 1800 {
		t.Fatalf("expected 1800s before start, got %d", rem)
	}

	_ = iv.Start()

	// Advance 5 mins
	iv.SetClock(FrozenClock(at.Add(5 * time.Minute)))
	if rem := iv.SessionRemaining(); rem != 1500 {
		t.Fatalf("expected 1500s remaining after 5m, got %d", rem)
	}

	// Answer with pacing metrics
	pacing := &PacingMetrics{
		TimeToFirstKeystrokeMs: 1200,
		DurationMs:             15000,
		TypedChars:             180,
		PastedChars:            20,
		PastedRatio:            0.1,
	}
	if err := iv.AnswerWithPacing("Detailed implementation answer", pacing); err != nil {
		t.Fatalf("answer with pacing failed: %v", err)
	}
	if len(iv.Answers) != 1 || iv.Answers[0].PacingTelemetry == nil || iv.Answers[0].PacingTelemetry.TimeToFirstKeystrokeMs != 1200 {
		t.Fatalf("unexpected answers telemetry: %+v", iv.Answers)
	}

	// Coding session recording
	iv.RecordCodingSession(CodingSession{
		QuestionIdx: 1,
		Language:    "go",
		Code:        "package main\nfunc main() {}",
	})
	if len(iv.CodingSessions) != 1 {
		t.Fatalf("expected 1 coding session, got %d", len(iv.CodingSessions))
	}
}

func TestSetClockAndMaxDurationExpiry(t *testing.T) {
	base := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	iv := mustInterview(t, base)
	_ = iv.Start()

	newClock := FrozenClock(base.Add(35 * time.Minute))
	iv.SetClock(newClock)
	iv.ExpireIfNeeded()
	if iv.Status != StatusExpired {
		t.Fatalf("expected status expired after 35m max duration, got %s", iv.Status)
	}
	if rem := iv.SessionRemaining(); rem != 0 {
		t.Fatalf("expected 0s remaining after expiry, got %d", rem)
	}
}
