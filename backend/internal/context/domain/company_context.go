package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

const (
	TypeFile = "file"
	TypeText = "text"
)

// CompanyContext — versioned tenant context (file or pasted text).
// content_hash enables dedup; version pins the context at interview time.
type CompanyContext struct {
	domain.Entity
	OrgID       uuid.UUID
	Type        string
	ContentHash string
	Version     int
	StoragePath string
}

func NewCompanyContext(orgID uuid.UUID, contentType, contentHash, storagePath string) (*CompanyContext, error) {
	if contentType != TypeFile && contentType != TypeText {
		return nil, errors.NewDomainError("CTX_TYPE_INVALID", "context type must be file or text")
	}
	if contentHash == "" {
		return nil, errors.NewDomainError("CTX_HASH_REQUIRED", "content hash is required")
	}
	return &CompanyContext{
		Entity:      domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		OrgID:       orgID,
		Type:        contentType,
		ContentHash: contentHash,
		StoragePath: storagePath,
		Version:     1,
	}, nil
}

type TenantPrompt struct {
	OrgID        uuid.UUID
	SystemPrompt string
	Version      int
	CreatedAt    time.Time
}
