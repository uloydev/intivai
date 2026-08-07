package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity is the base for all entities: identity + timestamps.
type Entity struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewID() uuid.UUID { return uuid.New() }
