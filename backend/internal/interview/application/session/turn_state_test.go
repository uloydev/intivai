package session_test

import (
	"testing"

	"github.com/intivai/backend/internal/interview/application/session"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

func TestTurnState_SendQuestionOnce(t *testing.T) {
	q := &ivdomain.Question{
		Idx:     1,
		Content: "What is Go concurrency?",
		IsProbe: false,
	}
	ts := &session.TurnState{
		Next:            q,
		Total:           5,
		RemainingSec:    1800,
		IsTopicComplete: true,
		TopicTurn:       1,
		MaxTopicTurns:   3,
	}

	if ts.WasQuestionSent() {
		t.Fatal("expected WasQuestionSent false initially")
	}

	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}

	// First call sends the frame
	ts.SendQuestionOnce(send)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame sent, got %d", len(frames))
	}
	if !ts.WasQuestionSent() {
		t.Fatal("expected WasQuestionSent true after send")
	}

	qm, ok := frames[0].(ivdomain.QuestionMessage)
	if !ok {
		t.Fatalf("expected ivdomain.QuestionMessage, got %T", frames[0])
	}
	if qm.Content != q.Content || qm.Idx != q.Idx {
		t.Errorf("unexpected content or idx: %+v", qm)
	}

	// Second call must NOT send anything
	ts.SendQuestionOnce(send)
	if len(frames) != 1 {
		t.Fatalf("expected still 1 frame sent, got %d", len(frames))
	}
}

func TestTurnState_NilNextQuestion_DoesNotSend(t *testing.T) {
	ts := &session.TurnState{
		Next: nil,
	}
	frames := 0
	ts.SendQuestionOnce(func(any) { frames++ })
	if frames != 0 {
		t.Fatalf("expected 0 frames sent for nil next question, got %d", frames)
	}
	if ts.WasQuestionSent() {
		t.Fatal("expected WasQuestionSent false for nil next question")
	}
}

func TestTurnState_SendQuestionOnce_TopicIncomplete_DoesNotSend(t *testing.T) {
	q := &ivdomain.Question{Idx: 2, Content: "Next question"}
	ts := &session.TurnState{
		Next:            q,
		Total:           5,
		IsTopicComplete: false,
	}
	frames := []any{}
	send := func(f any) {
		frames = append(frames, f)
	}
	ts.SendQuestionOnce(send)
	if len(frames) != 0 {
		t.Fatalf("expected 0 frames when topic incomplete, got %d", len(frames))
	}
	if ts.WasQuestionSent() {
		t.Fatal("expected WasQuestionSent false when topic incomplete")
	}
}
