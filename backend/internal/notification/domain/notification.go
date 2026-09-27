package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventInterviewCompleted EventType = "interview.completed"
	EventHumanRequested     EventType = "interview.human_requested"
	EventScreeningPassed    EventType = "candidate.screening_passed"
)

type RecruiterNotification struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	EventType EventType  `json:"event_type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	ActionURL string     `json:"action_url"`
	Read      bool       `json:"read"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationRepository interface {
	Create(ctx context.Context, n *RecruiterNotification) error
	List(ctx context.Context, orgID uuid.UUID, limit int) ([]*RecruiterNotification, error)
	MarkRead(ctx context.Context, orgID uuid.UUID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, orgID uuid.UUID) error
	UnreadCount(ctx context.Context, orgID uuid.UUID) (int, error)
}
