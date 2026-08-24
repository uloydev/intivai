package api

import (
	"testing"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

func TestTurnStateDoesNotDispatchQuestionTwiceAfterRecovery(t *testing.T) {
	turn := &turnState{
		next:            &ivdomain.Question{Idx: 2, Content: "next"},
		isTopicComplete: true,
		questionSent:    true,
	}
	frames := 0
	turn.sendQuestionOnce(func(any) { frames++ })
	if frames != 0 {
		t.Fatalf("recovered turn dispatched %d duplicate question frames", frames)
	}
	if !turn.wasQuestionSent() {
		t.Fatal("turn lost sent state")
	}
}
