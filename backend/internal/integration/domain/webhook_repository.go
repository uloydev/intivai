package domain

import (
	"context"

	"github.com/google/uuid"
)

type WebhookRepository interface {
	CreateConfig(ctx context.Context, cfg *WebhookConfig) error
	GetConfigByID(ctx context.Context, id uuid.UUID) (*WebhookConfig, error)
	ListConfigsByOrg(ctx context.Context, orgID uuid.UUID) ([]*WebhookConfig, error)
	UpdateConfig(ctx context.Context, cfg *WebhookConfig) error
	DeleteConfig(ctx context.Context, id uuid.UUID) error

	CreateDelivery(ctx context.Context, d *WebhookDelivery) error
	GetDeliveryByID(ctx context.Context, id uuid.UUID) (*WebhookDelivery, error)
	IncrementDeliveryAttempt(ctx context.Context, id uuid.UUID) (attempts int, claimed bool, err error)
	ListDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*WebhookDelivery, error)
	UpdateDelivery(ctx context.Context, d *WebhookDelivery) error
	UpdateDeliveryIfNotDelivered(ctx context.Context, d *WebhookDelivery) (bool, error)
	ListConfigsByEvent(ctx context.Context, orgID uuid.UUID, event WebhookEvent) ([]*WebhookConfig, error)
}
