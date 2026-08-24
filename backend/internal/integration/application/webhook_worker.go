package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/integration/domain"
	sharederrors "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const (
	TaskDeliverWebhook  = "deliver_webhook"
	maxDeliveryAttempts = 3
)

type DeliverWebhookPayload struct {
	DeliveryID string `json:"delivery_id"`
	OrgID      string `json:"org_id"`
}

type WebhookWorker struct {
	repo   domain.WebhookRepository
	pool   *gorm.DB
	client *http.Client
	logger zerolog.Logger
}

func NewWebhookWorker(repo domain.WebhookRepository, pool *gorm.DB, logger zerolog.Logger) *WebhookWorker {
	return &WebhookWorker{
		repo: repo,
		pool: pool,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				if err := domain.ValidateWebhookURL(req.URL.String()); err != nil {
					return fmt.Errorf("webhook redirect rejected: %w", err)
				}
				return nil
			},
		},
		logger: logger,
	}
}

func (w *WebhookWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskDeliverWebhook, w.handle)
}

func (w *WebhookWorker) handle(ctx context.Context, task *asynq.Task) error {
	var p DeliverWebhookPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	orgID, err := uuid.Parse(p.OrgID)
	if err != nil {
		return asynq.SkipRetry
	}
	deliveryID, err := uuid.Parse(p.DeliveryID)
	if err != nil {
		return asynq.SkipRetry
	}

	// Read delivery + config inside a tenant tx (FORCED RLS, migration 025).
	var delivery *domain.WebhookDelivery
	var cfg *domain.WebhookConfig
	var claimed bool
	err = db.RunInTx(ctx, w.pool, orgID.String(), func(tctx context.Context) error {
		var err error
		delivery, err = w.repo.GetDeliveryByID(tctx, deliveryID)
		if err != nil {
			return err
		}
		if delivery.OrgID != orgID {
			return domain.ErrNotFound
		}
		cfg, err = w.repo.GetConfigByID(tctx, delivery.WebhookID)
		if err != nil {
			return err
		}
		var attempts int
		attempts, claimed, err = w.repo.IncrementDeliveryAttempt(tctx, delivery.ID)
		if err == nil && claimed {
			delivery.Attempts = attempts
		}
		return err
	})
	if err != nil {
		if sharederrors.PermanentWorkerError(err, domain.ErrNotFound) {
			// Not-found / wrong-org never recovers by retrying. When the
			// delivery row exists but its config is gone, park it failed so
			// it does not sit pending forever; an absent or cross-org row
			// must not (and cannot) be updated.
			if delivery != nil && delivery.OrgID == orgID {
				delivery.FinalStatus = "failed"
				delivery.ResponseBody = "webhook config removed"
				if _, perr := w.persist(ctx, orgID, delivery); perr != nil {
					w.logger.Error().Err(perr).Str("delivery_id", p.DeliveryID).Msg("park delivery after config loss")
				}
			}
			return asynq.SkipRetry
		}
		// Transient claim failure — asynq must retry, skipping would strand
		// the delivery pending forever (finding D20).
		w.logger.Warn().Err(err).Str("delivery_id", p.DeliveryID).Str("org_id", orgID.String()).Msg("webhook claim failed — task will retry")
		return fmt.Errorf("claim webhook delivery %s: %w", p.DeliveryID, err)
	}
	if !claimed {
		return asynq.SkipRetry
	}
	if err := w.deliver(ctx, orgID, delivery, cfg); err != nil {
		return err
	}
	return nil
}

// deliver performs the HTTP POST and persists the outcome. Transient failures
// (network error, 5xx) return an error so asynq retries with its own backoff;
// the attempts counter is persisted BEFORE returning so it survives retries.
// Permanent outcomes (2xx, 4xx, attempts exhausted) return nil.
func (w *WebhookWorker) deliver(ctx context.Context, orgID uuid.UUID, delivery *domain.WebhookDelivery, cfg *domain.WebhookConfig) error {
	signature := computeHMAC(delivery.Payload, cfg.Secret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(delivery.Payload))
	if err != nil {
		return asynq.SkipRetry
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Intivai-Signature", "sha256="+signature)
	req.Header.Set("X-Intivai-Event", string(delivery.Event))

	resp, err := w.client.Do(req)
	if err != nil {
		delivery.FinalStatus = "pending"
		if delivery.Attempts >= maxDeliveryAttempts {
			delivery.FinalStatus = "failed"
		}
		updated, uerr := w.persist(ctx, orgID, delivery)
		if uerr != nil {
			return uerr
		}
		if !updated {
			return asynq.SkipRetry
		}
		if delivery.Attempts >= maxDeliveryAttempts {
			w.logger.Warn().Str("delivery_id", delivery.ID.String()).Int("attempts", delivery.Attempts).Err(err).Msg("webhook delivery permanently failed")
			return asynq.SkipRetry
		}
		return fmt.Errorf("webhook delivery attempt %d failed: %w", delivery.Attempts, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return fmt.Errorf("read webhook response body: %w", err)
	}
	delivery.StatusCode = resp.StatusCode
	delivery.ResponseBody = string(body)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		delivery.FinalStatus = "delivered"
		updated, err := w.persist(ctx, orgID, delivery)
		if err != nil {
			return err
		}
		if !updated {
			return asynq.SkipRetry
		}
		return nil
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Client error — retrying won't help.
		delivery.FinalStatus = "failed"
		updated, err := w.persist(ctx, orgID, delivery)
		if err != nil {
			return err
		}
		if !updated {
			return asynq.SkipRetry
		}
		w.logger.Warn().Str("delivery_id", delivery.ID.String()).Int("status", resp.StatusCode).Msg("webhook delivery rejected by receiver")
		return asynq.SkipRetry
	default:
		// 3xx/5xx — transient, retry.
		delivery.FinalStatus = "pending"
		if delivery.Attempts >= maxDeliveryAttempts {
			delivery.FinalStatus = "failed"
		}
		updated, uerr := w.persist(ctx, orgID, delivery)
		if uerr != nil {
			return uerr
		}
		if !updated {
			return asynq.SkipRetry
		}
		if delivery.Attempts >= maxDeliveryAttempts {
			w.logger.Warn().Str("delivery_id", delivery.ID.String()).Int("attempts", delivery.Attempts).Int("status", resp.StatusCode).Msg("webhook delivery permanently failed")
			return asynq.SkipRetry
		}
		return fmt.Errorf("webhook delivery attempt %d got status %d", delivery.Attempts, resp.StatusCode)
	}
}

// persist writes the delivery state in its own tenant tx (the HTTP call must
// not hold a DB transaction open).
func (w *WebhookWorker) persist(ctx context.Context, orgID uuid.UUID, delivery *domain.WebhookDelivery) (bool, error) {
	var updated bool
	err := db.RunInTx(ctx, w.pool, orgID.String(), func(tctx context.Context) error {
		var err error
		updated, err = w.repo.UpdateDeliveryIfNotDelivered(tctx, delivery)
		return err
	})
	return updated, err
}

func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
