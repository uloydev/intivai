package domain

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent struct {
	ID         uuid.UUID
	Type       string
	OccurredAt time.Time
}

func NewDomainEvent(eventType string) DomainEvent {
	return DomainEvent{
		ID:         uuid.New(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
	}
}
