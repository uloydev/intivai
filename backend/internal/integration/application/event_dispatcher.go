package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	intdomain "github.com/intivai/backend/internal/integration/domain"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	shareddomain "github.com/intivai/backend/internal/shared/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type NotificationCreator interface {
	Create(ctx context.Context, orgID uuid.UUID, eventType notifdomain.EventType, title, message, actionURL string) error
}

type Enqueuer interface {
	Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type DomainEventDispatcher interface {
	Dispatch(ctx context.Context, orgID uuid.UUID, event intdomain.WebhookEvent, payload []byte) error
}

type EventDispatcher struct {
	pool        *gorm.DB
	webhookRepo intdomain.WebhookRepository
	notifs      NotificationCreator
	queue       Enqueuer
	log         zerolog.Logger
}

func NewEventDispatcher(
	pool *gorm.DB,
	webhookRepo intdomain.WebhookRepository,
	notifs NotificationCreator,
	queue Enqueuer,
	log zerolog.Logger,
) *EventDispatcher {
	return &EventDispatcher{
		pool:        pool,
		webhookRepo: webhookRepo,
		notifs:      notifs,
		queue:       queue,
		log:         log,
	}
}

func (d *EventDispatcher) Dispatch(ctx context.Context, orgID uuid.UUID, event intdomain.WebhookEvent, payload []byte) error {
	if d.notifs != nil && event == intdomain.EventInterviewCompleted {
		if err := d.notifs.Create(ctx, orgID, notifdomain.EventInterviewCompleted,
			"Interview Completed",
			"An interview assessment finished and is ready for review.",
			"/interviews"); err != nil {
			d.log.Warn().Err(err).Str("org_id", orgID.String()).Msg("failed to create interview completed notification")
		}
	}

	var pendingDeliveries []DeliverWebhookPayload
	run := func(tctx context.Context) error {
		if d.webhookRepo == nil {
			return nil
		}

		configs, err := d.webhookRepo.ListConfigsByEvent(tctx, orgID, event)
		if err != nil {
			return fmt.Errorf("list configs by event: %w", err)
		}

		for _, whCfg := range configs {
			deliveryID := uuid.New()
			delivery := &intdomain.WebhookDelivery{
				Entity: shareddomain.Entity{
					ID:        deliveryID,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				},
				OrgID:       orgID,
				WebhookID:   whCfg.ID,
				Event:       event,
				Payload:     payload,
				FinalStatus: "pending",
				Attempts:    0,
			}
			if err := d.webhookRepo.CreateDelivery(tctx, delivery); err != nil {
				return fmt.Errorf("create webhook delivery: %w", err)
			}

			if d.queue != nil {
				pendingDeliveries = append(pendingDeliveries, DeliverWebhookPayload{
					DeliveryID: deliveryID.String(),
					OrgID:      orgID.String(),
				})
			}
		}
		return nil
	}

	var txErr error
	if d.pool != nil {
		txErr = db.RunInTx(ctx, d.pool, orgID.String(), run)
	} else {
		txErr = run(ctx)
	}
	if txErr != nil {
		return txErr
	}

	// Enqueue to Redis ONLY after Postgres transaction has committed successfully
	if d.queue != nil {
		var errs []error
		for _, pl := range pendingDeliveries {
			if _, err := d.queue.Enqueue(ctx, TaskDeliverWebhook, pl, asynq.MaxRetry(3)); err != nil {
				d.log.Error().Err(err).Str("org_id", orgID.String()).Msg("enqueue webhook delivery post-commit failed")
				errs = append(errs, fmt.Errorf("enqueue webhook delivery: %w", err))
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	return nil
}
