package application

import (
	"context"

	"github.com/google/uuid"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type NotificationService struct {
	pool *gorm.DB
	repo notifdomain.NotificationRepository
}

func NewNotificationService(pool *gorm.DB, repo notifdomain.NotificationRepository) *NotificationService {
	return &NotificationService{pool: pool, repo: repo}
}

func (s *NotificationService) runInTx(ctx context.Context, orgID string, fn func(tctx context.Context) error) error {
	if s.pool == nil {
		return fn(ctx)
	}
	return db.RunInTx(ctx, s.pool, orgID, fn)
}

func (s *NotificationService) Create(ctx context.Context, orgID uuid.UUID, eventType notifdomain.EventType, title, message, actionURL string) error {
	return s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		return s.repo.Create(tctx, &notifdomain.RecruiterNotification{
			OrgID:     orgID,
			EventType: eventType,
			Title:     title,
			Message:   message,
			ActionURL: actionURL,
			Read:      false,
		})
	})
}

func (s *NotificationService) List(ctx context.Context, orgID uuid.UUID, limit int) ([]*notifdomain.RecruiterNotification, error) {
	var out []*notifdomain.RecruiterNotification
	err := s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		var err error
		out, err = s.repo.List(tctx, orgID, limit)
		return err
	})
	return out, err
}

func (s *NotificationService) MarkRead(ctx context.Context, orgID uuid.UUID, id uuid.UUID) error {
	return s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		return s.repo.MarkRead(tctx, orgID, id)
	})
}

func (s *NotificationService) MarkAllRead(ctx context.Context, orgID uuid.UUID) error {
	return s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		return s.repo.MarkAllRead(tctx, orgID)
	})
}

func (s *NotificationService) UnreadCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.runInTx(ctx, orgID.String(), func(tctx context.Context) error {
		var err error
		count, err = s.repo.UnreadCount(tctx, orgID)
		return err
	})
	return count, err
}
