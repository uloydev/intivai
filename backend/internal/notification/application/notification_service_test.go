package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	notifapp "github.com/intivai/backend/internal/notification/application"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	"github.com/stretchr/testify/require"
)

type mockNotifRepo struct {
	items []*notifdomain.RecruiterNotification
}

func (m *mockNotifRepo) Create(ctx context.Context, n *notifdomain.RecruiterNotification) error {
	m.items = append(m.items, n)
	return nil
}

func (m *mockNotifRepo) List(ctx context.Context, orgID uuid.UUID, limit int) ([]*notifdomain.RecruiterNotification, error) {
	return m.items, nil
}

func (m *mockNotifRepo) MarkRead(ctx context.Context, orgID uuid.UUID, id uuid.UUID) error {
	for _, n := range m.items {
		if n.ID == id {
			n.Read = true
		}
	}
	return nil
}

func (m *mockNotifRepo) MarkAllRead(ctx context.Context, orgID uuid.UUID) error {
	for _, n := range m.items {
		n.Read = true
	}
	return nil
}

func (m *mockNotifRepo) UnreadCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	count := 0
	for _, n := range m.items {
		if !n.Read {
			count++
		}
	}
	return count, nil
}

func TestNotification_EventTypesAndReadState(t *testing.T) {
	orgID := uuid.New()
	notifID := uuid.New()

	repo := &mockNotifRepo{
		items: []*notifdomain.RecruiterNotification{
			{
				ID:        notifID,
				OrgID:     orgID,
				EventType: notifdomain.EventInterviewCompleted,
				Title:     "Interview Completed",
				Message:   "Candidate completed interview.",
				ActionURL: "/interviews/123",
				Read:      false,
				CreatedAt: time.Now(),
			},
		},
	}

	count, err := repo.UnreadCount(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = repo.MarkRead(context.Background(), orgID, notifID)
	require.NoError(t, err)

	count, err = repo.UnreadCount(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	svc := notifapp.NewNotificationService(nil, repo)
	err = svc.Create(context.Background(), orgID, notifdomain.EventHumanRequested, "Human Requested", "Candidate asked for human", "/interviews/456")
	require.NoError(t, err)

	unread, err := svc.UnreadCount(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 1, unread)

	err = svc.MarkAllRead(context.Background(), orgID)
	require.NoError(t, err)

	unread, err = svc.UnreadCount(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 0, unread)
}
