package application

import (
	"context"

	"github.com/google/uuid"
)

// OrgSettingsReader — driven port for org-level screening settings
// (weight overrides + passing threshold). Implemented by an adapter over the
// IAM org repo in cmd/server; keeps screening decoupled from persistence.
type OrgSettingsReader interface {
	ReadOrgSettings(ctx context.Context, orgID uuid.UUID) (weights map[string]float64, minScore float64, err error)
}
