package api

import (
	"testing"

	"github.com/intivai/backend/internal/interview/application/session"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

func TestTurnStateDoesNotDispatchQuestionTwiceAfterRecovery(t *testing.T) {
	turn := &session.TurnState{
		Next:            &ivdomain.Question{Idx: 2, Content: "next"},
		IsTopicComplete: true,
		QuestionSent:    true,
	}
	frames := 0
	turn.SendQuestionOnce(func(any) { frames++ })
	if frames != 0 {
		t.Fatalf("recovered turn dispatched %d duplicate question frames", frames)
	}
	if !turn.WasQuestionSent() {
		t.Fatal("turn lost sent state")
	}
}
