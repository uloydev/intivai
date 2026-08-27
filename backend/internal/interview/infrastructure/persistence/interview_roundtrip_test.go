package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

func seedOrg(t *testing.T, pool *gorm.DB, orgID, slug string) {
	t.Helper()
	if err := db.RunInTx(context.Background(), pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES ($1,$2,$3)`, orgID, "t", slug).Error
	}); err != nil {
		t.Fatal(err)
	}
}

// Round-trip: interview create (NULL optional fields) → get → update with
// transcript/status. Guards NULL scans + jsonb transcript.
func TestInterviewRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.NewString()
	seedOrg(t, pool, orgID, "iv"+orgID[:8])

	// application FK: create job + candidate + application chain
	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "J", "d").Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO candidates (id, org_id, name, status, created_at) VALUES ($1,$2,$3,'extracted',NOW())`, candID, orgID, "Jane").Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status, created_at) VALUES ($1,$2,$3,$4,'passed',NOW())`, appID, orgID, candID, jobID).Error
	})
	if err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresInterviewRepo(pool)
	iv, err := ivdomain.NewInterview(uuid.MustParse(orgID), appID, []ivdomain.Question{
		{Idx: 1, Content: "Q1", Category: "technical", Skill: "Go"},
	}, time.Now().UTC().Add(time.Hour), ivdomain.SystemClock())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, iv)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got *ivdomain.Interview
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatalf("get (NULL optional scans): %v", err)
	}
	if got.Status != ivdomain.StatusPending || len(got.Questions) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	_ = got.Start()
	_ = got.Answer("my answer")
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, got)
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ivdomain.StatusInProgress || got.LastQuestionIdx != 1 || len(got.Answers) != 1 {
		t.Fatalf("updated mismatch: status=%s idx=%d answers=%d", got.Status, got.LastQuestionIdx, len(got.Answers))
	}
}

func TestQuestionBankRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgID := uuid.New()
	seedOrg(t, pool, orgID.String(), "qb"+orgID.String()[:8])

	bank := NewPostgresQuestionBank(pool)
	q := ivdomain.Question{
		Content:  "How do channels coordinate goroutines?",
		Category: "technical",
		Skill:    "Go",
	}

	if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		return bank.Create(tctx, orgID, q)
	}); err != nil {
		t.Fatalf("bank create: %v", err)
	}

	var list []ivdomain.Question
	if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		var err error
		list, err = bank.ListByOrg(tctx, orgID)
		return err
	}); err != nil {
		t.Fatalf("bank list: %v", err)
	}

	if len(list) == 0 || list[0].Content != q.Content || list[0].Skill != q.Skill {
		t.Fatalf("unexpected question bank results: %+v", list)
	}
}

// seedInterviewRow — full org/job/candidate/application/interview chain for
// repo-level interview tests.
func seedInterviewRow(t *testing.T, pool *gorm.DB, orgID string) (*PostgresInterviewRepo, *ivdomain.Interview) {
	t.Helper()
	ctx := context.Background()
	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()
	err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		if err := tx.Exec(`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, jobID, orgID, "J", "d").Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO candidates (id, org_id, name, status, created_at) VALUES ($1,$2,$3,'extracted',NOW())`, candID, orgID, "Jane").Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, appID, orgID, candID, jobID).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresInterviewRepo(pool)
	iv, err := ivdomain.NewInterview(uuid.MustParse(orgID), appID, []ivdomain.Question{
		{Idx: 1, Content: "Q1", Category: "technical", Skill: "Go"},
	}, time.Now().UTC().Add(time.Hour), ivdomain.SystemClock())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Create(tctx, iv)
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// QA appends are valid ONLY while the interview is in progress (J6):
	// mirror the production Start() transition the service performs before
	// candidate input is accepted.
	if err := iv.Start(); err != nil {
		t.Fatal(err)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		return repo.Update(tctx, iv)
	}); err != nil {
		t.Fatalf("start persist: %v", err)
	}
	return repo, iv
}

// I15/I16: the append is guarded AT THE SQL LAYER by the live pair count, so
// the cap holds atomically under concurrent frames; a legacy jsonb-'null'
// column must normalize to [] instead of concatenating a null element.
func TestAppendQAPairWithinLimitGuard(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgUUID := uuid.New()
	orgID := orgUUID.String()
	seedOrg(t, pool, orgID, "qa"+orgID[:8])
	repo, iv := seedInterviewRow(t, pool, orgID)

	appendPair := func(question string, limit int) bool {
		t.Helper()
		var appended bool
		err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
			var err error
			appended, err = repo.AppendQAPairWithinLimit(tctx, iv.ID, ivdomain.QAPair{Question: question, Answer: "a: " + question, CreatedAt: time.Now().UTC()}, limit)
			return err
		})
		if err != nil {
			t.Fatalf("append %q: %v", question, err)
		}
		return appended
	}

	if !appendPair("first", 1) {
		t.Fatal("first append under limit=1 rejected")
	}
	if appendPair("second", 1) {
		t.Fatal("append over limit=1 accepted (guard missing)")
	}

	var got *ivdomain.Interview
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.QAPairs) != 1 || got.QAPairs[0].Question != "first" {
		t.Fatalf("stored pairs = %+v, want exactly [first]", got.QAPairs)
	}
	if err := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		return tx.Exec(`UPDATE interviews SET qa_pairs = 'null'::jsonb WHERE id = $1`, iv.ID).Error
	}); err != nil {
		t.Fatal(err)
	}
	if !appendPair("legacy", 5) {
		t.Fatal("append onto legacy 'null' column rejected")
	}
	err = db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		got, err = repo.GetByID(tctx, iv.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.QAPairs) != 1 || got.QAPairs[0].Question != "legacy" {
		t.Fatalf("legacy-null append produced %d pairs (%+v), want exactly [legacy]", len(got.QAPairs), got.QAPairs)
	}
}

// J6: the append UPDATE carries the ACTIVE/EXPIRY predicate itself, so a
// completed or expired interview can never accept a new pair even if a frame
// races the transition (service-side checks are advisory). Distinct outcomes:
// inactive/expired => ErrQAActive, missing => ErrNotFound, cap => (false,nil).
func TestAppendQAPairWithinLimitGatesInactiveAndExpired(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	orgUUID := uuid.New()
	orgID := orgUUID.String()
	seedOrg(t, pool, orgID, "j6"+orgID[:8])
	repo, iv := seedInterviewRow(t, pool, orgID)

	appendExpect := func(t *testing.T, statusQuery bool, wantErr error, pairQuestion string) {
		t.Helper()
		var gotErr error
		var appended bool
		errDB := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
			tx, _ := db.TxFrom(tctx)
			if statusQuery {
				if err := tx.Exec(`UPDATE interviews SET status = 'completed', qa_pairs = '[]'::jsonb WHERE id = $1`, iv.ID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Exec(`UPDATE interviews SET status = 'in_progress', expires_at = NOW() - interval '1 minute', qa_pairs = '[]'::jsonb WHERE id = $1`, iv.ID).Error; err != nil {
					return err
				}
			}
			appended, gotErr = repo.AppendQAPairWithinLimit(tctx, iv.ID,
				ivdomain.QAPair{Question: pairQuestion, Answer: "a", CreatedAt: time.Now().UTC()}, 10)
			return nil
		})
		if errDB != nil {
			t.Fatalf("tx: %v", errDB)
		}
		if appended {
			t.Fatalf("append accepted on non-active interview (question %q)", pairQuestion)
		}
		if wantErr == nil {
			if gotErr != nil {
				t.Fatalf("got err %v, want nil (cap case)", gotErr)
			}
		} else if !errors.Is(gotErr, wantErr) {
			t.Fatalf("got err %v, want %v", gotErr, wantErr)
		}
	}

	appendExpect(t, true, ivdomain.ErrQAActive, "q-completed")
	appendExpect(t, false, ivdomain.ErrQAActive, "q-expired")

	// Missing row.
	errDB := db.RunInTx(ctx, pool, orgID, func(tctx context.Context) error {
		_, e := repo.AppendQAPairWithinLimit(tctx, uuid.New(),
			ivdomain.QAPair{Question: "q-missing", Answer: "a", CreatedAt: time.Now().UTC()}, 10)
		return e
	})
	if !errors.Is(errDB, ivdomain.ErrNotFound) {
		t.Fatalf("missing row: got %v, want ErrNotFound", errDB)
	}
}
