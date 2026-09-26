package persistence

import (
	"context"

	"github.com/google/uuid"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresNotificationRepo struct {
	pool *gorm.DB
}

func NewPostgresNotificationRepo(pool *gorm.DB) *PostgresNotificationRepo {
	return &PostgresNotificationRepo{pool: pool}
}

func (r *PostgresNotificationRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresNotificationRepo) Create(ctx context.Context, n *notifdomain.RecruiterNotification) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return q.WithContext(ctx).Exec(
		`INSERT INTO recruiter_notifications (id, org_id, user_id, event_type, title, message, action_url, read, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		n.ID, n.OrgID, n.UserID, string(n.EventType), n.Title, n.Message, n.ActionURL, n.Read).Error
}

func (r *PostgresNotificationRepo) List(ctx context.Context, orgID uuid.UUID, limit int) ([]*notifdomain.RecruiterNotification, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := q.WithContext(ctx).Raw(
		`SELECT id, org_id, user_id, event_type, title, message, action_url, read, created_at
		 FROM recruiter_notifications
		 WHERE org_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, orgID, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*notifdomain.RecruiterNotification, 0)
	for rows.Next() {
		var (
			n     notifdomain.RecruiterNotification
			evStr string
		)
		if err := rows.Scan(&n.ID, &n.OrgID, &n.UserID, &evStr, &n.Title, &n.Message, &n.ActionURL, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.EventType = notifdomain.EventType(evStr)
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *PostgresNotificationRepo) MarkRead(ctx context.Context, orgID uuid.UUID, id uuid.UUID) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE recruiter_notifications SET read = true WHERE org_id = $1 AND id = $2`, orgID, id).Error
}

func (r *PostgresNotificationRepo) MarkAllRead(ctx context.Context, orgID uuid.UUID) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE recruiter_notifications SET read = true WHERE org_id = $1 AND read = false`, orgID).Error
}

func (r *PostgresNotificationRepo) UnreadCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	q, err := r.q(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = q.WithContext(ctx).Raw(
		`SELECT COUNT(id) FROM recruiter_notifications WHERE org_id = $1 AND read = false`, orgID).Scan(&count).Error
	return int(count), err
}
