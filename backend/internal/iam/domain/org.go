package domain

import (
	"encoding/json"
	"time"

	"github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

type Org struct {
	domain.AggregateRoot
	Name              string
	Slug              string
	Plan              string
	ScoringWeights    map[string]float64 // per-tenant partial override; falls back to global defaults
	MinScoreToProceed *float64           // per-tenant threshold override (default 50 via code)
}

func NewOrg(name, slug string) (*Org, error) {
	if name == "" {
		return nil, errors.NewDomainError("ORG_NAME_REQUIRED", "org name is required")
	}
	if slug == "" {
		return nil, errors.NewDomainError("ORG_SLUG_REQUIRED", "org slug is required")
	}
	return &Org{
		AggregateRoot: domain.AggregateRoot{Entity: domain.Entity{ID: domain.NewID(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}},
		Name:          name,
		Slug:          slug,
		Plan:          "free",
	}, nil
}

func (o *Org) SetScoringWeights(raw map[string]float64) error {
	if raw == nil {
		return nil
	}
	for k, v := range raw {
		if !validWeight(v) {
			return errors.NewDomainError("INVALID_WEIGHT", "invalid scoring weight for "+k+": must be 0..1")
		}
	}
	o.ScoringWeights = raw
	return nil
}

func validWeight(v float64) bool {
	return v >= 0 && v <= 1
}

func (o *Org) MarshalScoringWeights() json.RawMessage {
	if o.ScoringWeights == nil {
		return nil
	}
	b, _ := json.Marshal(o.ScoringWeights)
	return b
}
