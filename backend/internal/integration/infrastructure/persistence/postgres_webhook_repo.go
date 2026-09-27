package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/integration/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresWebhookRepo struct {
	pool *gorm.DB
}

func NewPostgresWebhookRepo(pool *gorm.DB) *PostgresWebhookRepo {
	return &PostgresWebhookRepo{pool: pool}
}

func (r *PostgresWebhookRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresWebhookRepo) CreateConfig(ctx context.Context, cfg *domain.WebhookConfig) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	eventsJSON, err := json.Marshal(cfg.Events)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO webhook_configs (id, org_id, url, events, secret, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cfg.ID, cfg.OrgID, cfg.URL, eventsJSON, cfg.Secret, cfg.Active, cfg.CreatedAt, cfg.UpdatedAt,
	).Error
}

func (r *PostgresWebhookRepo) GetConfigByID(ctx context.Context, id uuid.UUID) (*domain.WebhookConfig, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var cfg domain.WebhookConfig
	var eventsJSON []byte
	row := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, url, events, secret, active, created_at, updated_at
		 FROM webhook_configs WHERE id = ?`, id,
	).Row()
	if err := row.Scan(&cfg.ID, &cfg.OrgID, &cfg.URL, &eventsJSON, &cfg.Secret, &cfg.Active, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(eventsJSON, &cfg.Events); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *PostgresWebhookRepo) ListConfigsByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.WebhookConfig, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, url, events, secret, active, created_at, updated_at
		 FROM webhook_configs WHERE org_id = ? ORDER BY created_at DESC`, orgID,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var configs []*domain.WebhookConfig
	for rows.Next() {
		var cfg domain.WebhookConfig
		var eventsJSON []byte
		if err := rows.Scan(&cfg.ID, &cfg.OrgID, &cfg.URL, &eventsJSON, &cfg.Secret, &cfg.Active, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(eventsJSON, &cfg.Events); err != nil {
			return nil, err
		}
		configs = append(configs, &cfg)
	}
	return configs, rows.Err()
}

func (r *PostgresWebhookRepo) UpdateConfig(ctx context.Context, cfg *domain.WebhookConfig) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	eventsJSON, err := json.Marshal(cfg.Events)
	if err != nil {
		return err
	}
	cfg.UpdatedAt = time.Now()
	return tx.WithContext(ctx).Exec(
		`UPDATE webhook_configs SET url = ?, events = ?, secret = ?, active = ?, updated_at = ?
		 WHERE id = ? AND org_id = ?`,
		cfg.URL, eventsJSON, cfg.Secret, cfg.Active, cfg.UpdatedAt, cfg.ID, cfg.OrgID,
	).Error
}

func (r *PostgresWebhookRepo) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(`DELETE FROM webhook_configs WHERE id = ?`, id).Error
}

func (r *PostgresWebhookRepo) CreateDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO webhook_deliveries (id, org_id, webhook_id, event, payload, status_code, response_body, attempts, next_retry_at, final_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.OrgID, d.WebhookID, d.Event, d.Payload, d.StatusCode, d.ResponseBody, d.Attempts, d.NextRetryAt, d.FinalStatus, d.CreatedAt, d.UpdatedAt,
	).Error
}

func (r *PostgresWebhookRepo) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*domain.WebhookDelivery, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var d domain.WebhookDelivery
	row := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, webhook_id, event, payload, status_code, response_body, attempts, next_retry_at, final_status, created_at, updated_at
		 FROM webhook_deliveries WHERE id = ?`, id,
	).Row()
	if err := row.Scan(&d.ID, &d.OrgID, &d.WebhookID, &d.Event, &d.Payload, &d.StatusCode, &d.ResponseBody, &d.Attempts, &d.NextRetryAt, &d.FinalStatus, &d.CreatedAt, &d.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *PostgresWebhookRepo) IncrementDeliveryAttempt(ctx context.Context, id uuid.UUID) (int, bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, false, err
	}
	var attempts int
	row := tx.WithContext(ctx).Raw(
		`UPDATE webhook_deliveries
		 SET attempts = attempts + 1, updated_at = ?
		 WHERE id = ? AND final_status = 'pending' AND attempts < ?
		 RETURNING attempts`, time.Now(), id, 3,
	).Row()
	if err := row.Scan(&attempts); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return attempts, true, nil
}

func (r *PostgresWebhookRepo) ListDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*domain.WebhookDelivery, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, webhook_id, event, payload, status_code, response_body, attempts, next_retry_at, final_status, created_at, updated_at
		 FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT ?`, webhookID, limit,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var deliveries []*domain.WebhookDelivery
	for rows.Next() {
		var d domain.WebhookDelivery
		if err := rows.Scan(&d.ID, &d.OrgID, &d.WebhookID, &d.Event, &d.Payload, &d.StatusCode, &d.ResponseBody, &d.Attempts, &d.NextRetryAt, &d.FinalStatus, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, rows.Err()
}

func (r *PostgresWebhookRepo) UpdateDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	d.UpdatedAt = time.Now()
	return tx.WithContext(ctx).Exec(
		`UPDATE webhook_deliveries SET status_code = ?, response_body = ?, attempts = ?, next_retry_at = ?, final_status = ?, updated_at = ?
		 WHERE id = ?`,
		d.StatusCode, d.ResponseBody, d.Attempts, d.NextRetryAt, d.FinalStatus, d.UpdatedAt, d.ID,
	).Error
}

func (r *PostgresWebhookRepo) UpdateDeliveryIfNotDelivered(ctx context.Context, d *domain.WebhookDelivery) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	d.UpdatedAt = time.Now()
	result := tx.WithContext(ctx).Exec(
		`UPDATE webhook_deliveries SET status_code = ?, response_body = ?, attempts = ?, next_retry_at = ?, final_status = ?, updated_at = ?
		 WHERE id = ? AND final_status <> 'delivered'`,
		d.StatusCode, d.ResponseBody, d.Attempts, d.NextRetryAt, d.FinalStatus, d.UpdatedAt, d.ID,
	)
	return result.RowsAffected == 1, result.Error
}

func (r *PostgresWebhookRepo) ListConfigsByEvent(ctx context.Context, orgID uuid.UUID, event domain.WebhookEvent) ([]*domain.WebhookConfig, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, url, events, secret, active, created_at, updated_at
		 FROM webhook_configs WHERE org_id = ? AND active = true AND events @> ?::jsonb`, orgID, `["`+string(event)+`"]`,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var configs []*domain.WebhookConfig
	for rows.Next() {
		var cfg domain.WebhookConfig
		var eventsJSON []byte
		if err := rows.Scan(&cfg.ID, &cfg.OrgID, &cfg.URL, &eventsJSON, &cfg.Secret, &cfg.Active, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(eventsJSON, &cfg.Events); err != nil {
			return nil, err
		}
		configs = append(configs, &cfg)
	}
	return configs, rows.Err()
}
