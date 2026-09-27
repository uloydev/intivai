package session

import (
	"sync"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

// TurnState tracks the in-flight state of a single interview turn.
type TurnState struct {
	Mu              sync.Mutex
	QuestionSent    bool
	Next            *ivdomain.Question
	Total           int
	RemainingSec    int
	IsTopicComplete bool
	TopicTurn       int
	MaxTopicTurns   int
	OnSent          func(*ivdomain.Question)
}

// SendQuestionOnce emits the next question frame exactly once.
func (t *TurnState) SendQuestionOnce(send func(any)) {
	t.Mu.Lock()
	defer t.Mu.Unlock()
	if t.QuestionSent || !t.IsTopicComplete || t.Next == nil {
		return
	}
	t.QuestionSent = true
	arch, limit := ivdomain.DetermineQuestionArchetype(*t.Next)
	send(ivdomain.QuestionMessage{
		Type:                ivdomain.MsgQuestion,
		Content:             t.Next.Content,
		Idx:                 t.Next.Idx,
		TotalQuestions:      t.Total,
		IsProbe:             t.Next.IsProbe,
		Archetype:           arch,
		TimeLimitSec:        limit,
		SessionRemainingSec: t.RemainingSec,
		TopicTurn:           1,
		MaxTopicTurns:       t.MaxTopicTurns,
	})
	if t.OnSent != nil {
		t.OnSent(t.Next)
	}
}

// WasQuestionSent returns true if the question has already been dispatched.
func (t *TurnState) WasQuestionSent() bool {
	t.Mu.Lock()
	defer t.Mu.Unlock()
	return t.QuestionSent
}
