package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	"github.com/intivai/backend/internal/integration/domain"
	"github.com/intivai/backend/internal/shared/errors"
)

type WebhookService struct {
	repo domain.WebhookRepository
}

func NewWebhookService(repo domain.WebhookRepository) *WebhookService {
	return &WebhookService{repo: repo}
}

type CreateWebhookCommand struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

type WebhookConfigResult struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

func (s *WebhookService) Create(ctx context.Context, actor application.AuthContext, cmd CreateWebhookCommand) (*WebhookConfigResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	events := make([]domain.WebhookEvent, len(cmd.Events))
	for i, e := range cmd.Events {
		events[i] = domain.WebhookEvent(e)
	}
	cfg, err := domain.NewWebhookConfig(actor.OrgID, cmd.URL, events, cmd.Secret)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return toConfigResult(cfg), nil
}

func (s *WebhookService) Get(ctx context.Context, actor application.AuthContext, id uuid.UUID) (*WebhookConfigResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	cfg, err := s.repo.GetConfigByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cfg.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "webhook belongs to another org")
	}
	return toConfigResult(cfg), nil
}

func (s *WebhookService) List(ctx context.Context, actor application.AuthContext) ([]*WebhookConfigResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	configs, err := s.repo.ListConfigsByOrg(ctx, actor.OrgID)
	if err != nil {
		return nil, err
	}
	results := make([]*WebhookConfigResult, len(configs))
	for i, cfg := range configs {
		results[i] = toConfigResult(cfg)
	}
	return results, nil
}

type UpdateWebhookCommand struct {
	URL    *string  `json:"url"`
	Events []string `json:"events"`
	Secret *string  `json:"secret"`
	Active *bool    `json:"active"`
}

func (s *WebhookService) Update(ctx context.Context, actor application.AuthContext, id uuid.UUID, cmd UpdateWebhookCommand) (*WebhookConfigResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	cfg, err := s.repo.GetConfigByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cfg.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "webhook belongs to another org")
	}
	if cmd.URL != nil {
		if err := domain.ValidateWebhookURL(*cmd.URL); err != nil {
			return nil, err
		}
		cfg.URL = *cmd.URL
	}
	if cmd.Events != nil {
		events := make([]domain.WebhookEvent, len(cmd.Events))
		for i, e := range cmd.Events {
			events[i] = domain.WebhookEvent(e)
			if !domain.ValidEvents[events[i]] {
				return nil, domain.ErrInvalidEvent
			}
		}
		cfg.Events = events
	}
	if cmd.Secret != nil {
		cfg.Secret = *cmd.Secret
	}
	if cmd.Active != nil {
		cfg.Active = *cmd.Active
	}
	if err := s.repo.UpdateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return toConfigResult(cfg), nil
}

func (s *WebhookService) Delete(ctx context.Context, actor application.AuthContext, id uuid.UUID) error {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return err
	}
	cfg, err := s.repo.GetConfigByID(ctx, id)
	if err != nil {
		return err
	}
	if cfg.OrgID != actor.OrgID {
		return errors.NewDomainError("FORBIDDEN", "webhook belongs to another org")
	}
	return s.repo.DeleteConfig(ctx, id)
}

func (s *WebhookService) ListDeliveries(ctx context.Context, actor application.AuthContext, webhookID uuid.UUID) ([]*WebhookDeliveryResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	cfg, err := s.repo.GetConfigByID(ctx, webhookID)
	if err != nil {
		return nil, err
	}
	if cfg.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "webhook belongs to another org")
	}
	deliveries, err := s.repo.ListDeliveriesByWebhook(ctx, webhookID, 10)
	if err != nil {
		return nil, err
	}
	results := make([]*WebhookDeliveryResult, len(deliveries))
	for i, d := range deliveries {
		results[i] = toDeliveryResult(d)
	}
	return results, nil
}

type WebhookDeliveryResult struct {
	ID           string `json:"id"`
	Event        string `json:"event"`
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body"`
	Attempts     int    `json:"attempts"`
	FinalStatus  string `json:"final_status"`
	CreatedAt    string `json:"created_at"`
}

func toConfigResult(cfg *domain.WebhookConfig) *WebhookConfigResult {
	events := make([]string, len(cfg.Events))
	for i, e := range cfg.Events {
		events[i] = string(e)
	}
	return &WebhookConfigResult{
		ID:        cfg.ID.String(),
		URL:       cfg.URL,
		Events:    events,
		Active:    cfg.Active,
		CreatedAt: cfg.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toDeliveryResult(d *domain.WebhookDelivery) *WebhookDeliveryResult {
	return &WebhookDeliveryResult{
		ID:           d.ID.String(),
		Event:        string(d.Event),
		StatusCode:   d.StatusCode,
		ResponseBody: d.ResponseBody,
		Attempts:     d.Attempts,
		FinalStatus:  d.FinalStatus,
		CreatedAt:    d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
