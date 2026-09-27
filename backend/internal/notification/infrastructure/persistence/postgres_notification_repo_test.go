package persistence_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	notifrepo "github.com/intivai/backend/internal/notification/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestPostgresNotificationRepo(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)

	repo := notifrepo.NewPostgresNotificationRepo(pool)
	orgID := uuid.New()

	ctx := context.Background()
	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		require.True(t, ok)
		_ = tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, orgID, "Notif Org", "notif-"+uuid.NewString()[:6])

		// 1. Create notification
		notif := &notifdomain.RecruiterNotification{
			OrgID:     orgID,
			EventType: notifdomain.EventInterviewCompleted,
			Title:     "Interview Completed",
			Message:   "Candidate finished interview",
			ActionURL: "/interviews/123",
			Read:      false,
		}
		err := repo.Create(tctx, notif)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, notif.ID)

		// 2. UnreadCount
		count, err := repo.UnreadCount(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// 3. List
		list, err := repo.List(tctx, orgID, 10)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, notif.Title, list[0].Title)

		// 4. MarkRead
		err = repo.MarkRead(tctx, orgID, notif.ID)
		require.NoError(t, err)

		count, err = repo.UnreadCount(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		// 5. Create another and MarkAllRead
		notif2 := &notifdomain.RecruiterNotification{
			OrgID:     orgID,
			EventType: notifdomain.EventScreeningPassed,
			Title:     "Screening Passed",
			Message:   "Candidate qualified",
			ActionURL: "/candidates/123",
			Read:      false,
		}
		err = repo.Create(tctx, notif2)
		require.NoError(t, err)

		err = repo.MarkAllRead(tctx, orgID)
		require.NoError(t, err)

		count, err = repo.UnreadCount(tctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		return nil
	})
	require.NoError(t, err)
}
