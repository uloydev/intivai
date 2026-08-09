package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusExpired    Status = "expired"
)

// Realtime config (Research §2, "Heartbeat, Timeout & Reconnection").
const (
	// MaxInterviewDuration — wall-clock cap from Start; activity does not
	// extend it.
	MaxInterviewDuration = 30 * time.Minute
	// PerQuestionTimeout — max wait for a candidate frame (question answer).
	PerQuestionTimeout = 3 * time.Minute
)

// Question VO — one step of the interview.
type Question struct {
	Idx      int    `json:"idx"` // 1-based, stable across resume
	Content  string `json:"content"`
	Category string `json:"category"`
	Skill    string `json:"skill,omitempty"`
}

// Answer VO — candidate response, stored for evaluation.
type Answer struct {
	Idx        int       `json:"idx"`
	Content    string    `json:"content"`
	AnsweredAt time.Time `json:"answered_at"`
}

// Interview aggregate — state machine driven by an injectable clock
// (idle timeout + expiry are time-based; tests use FrozenClock).
type Interview struct {
	domain.Entity
	OrgID           uuid.UUID
	ApplicationID   uuid.UUID
	Status          Status
	Questions       []Question
	Answers         []Answer
	LastQuestionIdx int
	ContextVersion  int // company-context version pinned at creation (audit)
	Evaluation      []byte // post-interview report (evaluation JSONB), hydrated by GetByID
	ConsentGiven    bool   // GDPR consent captured before interview start
	StartedAt       *time.Time
	CompletedAt     *time.Time
	ExpiresAt       *time.Time
	clock           Clock
	lastActivity    time.Time
}

func NewInterview(orgID, applicationID uuid.UUID, questions []Question, expiresAt time.Time, clock Clock) (*Interview, error) {
	if len(questions) == 0 {
		return nil, errors.NewDomainError("INTERVIEW_NO_QUESTIONS", "interview requires at least one question")
	}
	if clock == nil {
		clock = SystemClock()
	}
	exp := expiresAt.UTC()
	return &Interview{
		Entity:        domain.Entity{ID: domain.NewID(), CreatedAt: clock.Now()},
		OrgID:         orgID,
		ApplicationID: applicationID,
		Status:        StatusPending,
		Questions:     questions,
		ExpiresAt:     &exp,
		clock:         clock,
		lastActivity:  clock.Now(),
	}, nil
}

// Start moves pending → in_progress. No-op for already-started interviews
// (reconnect path).
func (iv *Interview) Start() error {
	switch iv.Status {
	case StatusPending:
		now := iv.clock.Now()
		iv.Status = StatusInProgress
		iv.StartedAt = &now
		iv.lastActivity = now
		return nil
	case StatusInProgress:
		return nil // resume
	default:
		return errors.NewDomainError("INTERVIEW_NOT_STARTABLE", "interview cannot start from "+string(iv.Status))
	}
}

// Answer records a candidate answer, advances the cursor, touches activity.
func (iv *Interview) Answer(content string) error {
	if iv.Status != StatusInProgress {
		return errors.NewDomainError("INTERVIEW_NOT_IN_PROGRESS", "interview is not in progress")
	}
	if content == "" {
		return errors.NewDomainError("ANSWER_EMPTY", "answer is empty")
	}
	idx := iv.LastQuestionIdx + 1
	if idx > len(iv.Questions) {
		idx = len(iv.Questions)
	}
	iv.Answers = append(iv.Answers, Answer{Idx: idx, Content: content, AnsweredAt: iv.clock.Now()})
	iv.LastQuestionIdx = idx
	iv.lastActivity = iv.clock.Now()
	return nil
}

// NextQuestion returns the next unanswered question, or nil when done.
func (iv *Interview) NextQuestion() *Question {
	if iv.Status != StatusInProgress {
		return nil
	}
	if iv.LastQuestionIdx >= len(iv.Questions) {
		return nil
	}
	q := iv.Questions[iv.LastQuestionIdx]
	return &q
}

// InsertProbeAfter appends a dynamic follow-up (weakness probe) right after
// the current question, renumbering the rest so the cursor (LastQuestionIdx)
// stays sequential. Persisted with the normal transcript, so resume and
// evaluation see it like any other question.
func (iv *Interview) InsertProbeAfter(currentIdx int, content, category, skill string) (*Question, error) {
	if iv.Status != StatusInProgress {
		return nil, errors.NewDomainError("INTERVIEW_NOT_IN_PROGRESS", "interview is not in progress")
	}
	insertAt := currentIdx // 0-based position after the answered question
	if insertAt < 0 || insertAt > len(iv.Questions) {
		return nil, errors.NewDomainError("INTERVIEW_PROBE_INDEX", "invalid probe position")
	}
	q := Question{
		Idx:      insertAt + 1,
		Content:  content,
		Category: category,
		Skill:    skill,
	}
	iv.Questions = append(iv.Questions, Question{})
	copy(iv.Questions[insertAt+1:], iv.Questions[insertAt:len(iv.Questions)-1])
	iv.Questions[insertAt] = q
	for i := insertAt + 1; i < len(iv.Questions); i++ {
		iv.Questions[i].Idx = i + 1
	}
	return &q, nil
}

// Complete marks the interview finished (all questions answered or recruiter
// ended it).
func (iv *Interview) Complete() error {
	if iv.Status != StatusInProgress {
		return errors.NewDomainError("INTERVIEW_NOT_IN_PROGRESS", "interview is not in progress")
	}
	now := iv.clock.Now()
	iv.Status = StatusCompleted
	iv.CompletedAt = &now
	return nil
}

// ExpireIfNeeded transitions to expired when the deadline passed: either the
// invitation deadline (7 days) or the 30-minute duration cap from Start.
func (iv *Interview) ExpireIfNeeded() {
	now := iv.clock.Now()
	if iv.Status != StatusCompleted && iv.Status != StatusExpired {
		if iv.ExpiresAt != nil && now.After(*iv.ExpiresAt) {
			iv.Status = StatusExpired
			return
		}
		if iv.StartedAt != nil && now.Sub(*iv.StartedAt) >= MaxInterviewDuration {
			iv.Status = StatusExpired
		}
	}
}

// IsIdle reports whether the candidate has been silent past the timeout.
// The idle timeout is the ONLY time-based disconnect rule (docs: 5 min).
func (iv *Interview) IsIdle(timeout time.Duration) bool {
	return iv.Status == StatusInProgress && iv.clock.Now().Sub(iv.lastActivity) >= timeout
}

// Touch refreshes the activity marker (any client frame counts).
func (iv *Interview) Touch() {
	iv.lastActivity = iv.clock.Now()
}

// SetClock attaches the clock after persistence hydration.
func (iv *Interview) SetClock(c Clock) {
	if c != nil {
		iv.clock = c
	}
}

// ResumeIdx — reconnection resumes from the last unanswered question.
func (iv *Interview) ResumeIdx() int {
	return iv.LastQuestionIdx + 1
}
