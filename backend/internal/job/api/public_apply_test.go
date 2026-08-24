package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	iamauth "github.com/intivai/backend/internal/iam/infrastructure/auth"
	jobapi "github.com/intivai/backend/internal/job/api"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrapi "github.com/intivai/backend/internal/screening/api"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeEnqueuer — queue seam capture (no Redis): records every enqueued task
// so tests can assert on job type + payload.
type fakeEnqueuer struct {
	mu    sync.Mutex
	tasks []capturedTask
}

type capturedTask struct {
	jobType string
	payload any
}

func (f *fakeEnqueuer) Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks = append(f.tasks, capturedTask{jobType: jobType, payload: payload})
	return &asynq.TaskInfo{}, nil
}

func (f *fakeEnqueuer) ofType(jobType string) []capturedTask {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []capturedTask{}
	for _, t := range f.tasks {
		if t.jobType == jobType {
			out = append(out, t)
		}
	}
	return out
}

// newApplyTestStore — MinIO-backed object store for apply tests (skips when
// TEST_MINIO_* is unset).
func newApplyTestStore(t *testing.T) cvapp.ObjectStore {
	t.Helper()
	store, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || store == nil {
		t.Skip("TEST_MINIO_* not set")
	}
	return store
}

// seedActiveJob — published active job under a fresh org (RLS: tenant tx).
func seedActiveJob(t *testing.T, pool *gorm.DB, orgID, jobID uuid.UUID, title string) {
	t.Helper()
	err := db.RunInTx(context.Background(), pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			t.Fatal("no tenant tx")
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT (id) DO NOTHING`,
			orgID, "Portal Co", "portal-"+orgID.String()[:8]).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type, status, is_published)
			VALUES (?, ?, ?, ?, ?, ?, 'active', true)`,
			jobID, orgID, title, "Apply flow test", "Remote", "Full-time").Error
	})
	require.NoError(t, err)
}

// applyMultipart — builds the multipart apply request.
func applyMultipart(t *testing.T, jobID uuid.UUID, name, email string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("name", name))
	require.NoError(t, mw.WriteField("email", email))
	fw, err := mw.CreateFormFile("file", "resume.pdf")
	require.NoError(t, err)
	_, err = fw.Write([]byte("%PDF-1.4 fake resume for apply flow"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest("POST", "/api/v1/public/jobs/"+jobID.String()+"/apply", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// TestPublicApplyDoesNotReturnPortalToken — C9 regression: the public apply
// response must NEVER carry the portal magic token (attacker-supplied email +
// returned credential = victim account takeover). The token row is still
// minted; delivery happens by email only.
func TestPublicApplyDoesNotReturnPortalToken(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	minioStore := newApplyTestStore(t)

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)
	ctx := context.Background()

	orgID := uuid.New()
	jobID := uuid.New()
	jobTitle := "Portal Token Engineer"
	email := "portal-apply-" + uuid.NewString()[:8] + "@candidate.io"
	publicURL := "http://localhost:5173"

	seedActiveJob(t, pool, orgID, jobID, jobTitle)

	fake := &fakeEnqueuer{}
	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	tokens := iamauth.NewJWTProvider("secret-test-key-32-chars-intivai-1234")
	publicHandler := jobapi.NewPublicJobHandler(
		pool, jobrepo.NewPostgresJobRepo(pool), cvrepo.NewPostgresCandidateRepo(pool),
		scrrepo.NewPostgresApplicationRepo(pool), minioStore, fake, portalRepo, publicURL,
	)
	portalHandler := scrapi.NewCandidatePortalHandler(portalRepo, tokens, nil, publicURL)

	app := fiber.New()
	app.Post("/api/v1/public/jobs/:id/apply", publicHandler.Apply)
	app.Post("/api/v1/public/candidate/auth/verify", portalHandler.VerifyOTP)
	app.Get("/api/v1/candidate/portal/applications", portalHandler.RequireCandidateAuth, portalHandler.ListApplications)

	req := applyMultipart(t, jobID, "Portal Candidate", email)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContainsf(t, string(bodyBytes), "portal_token",
		"apply response must not leak the portal credential: %s", string(bodyBytes))

	var applyResult struct {
		Data struct {
			CandidateID string `json:"candidate_id"`
			JobID       string `json:"job_id"`
			Status      string `json:"status"`
			Message     string `json:"message"`
		} `json:"data"`
	}
	err = json.Unmarshal(bodyBytes, &applyResult)
	require.NoError(t, err)
	require.Equal(t, "submitted", applyResult.Data.Status)

	// The magic-token row still exists (delivery moved to email), unused,
	// expiring ~24h from now.
	var rowToken string
	var expiresAt time.Time
	err = pool.WithContext(ctx).Raw(
		`SELECT token, expires_at FROM candidate_otps WHERE LOWER(email) = ? AND used_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`,
		email,
	).Row().Scan(&rowToken, &expiresAt)
	require.NoError(t, err)
	require.NotEmpty(t, rowToken)
	ttl := time.Until(expiresAt)
	require.Greater(t, ttl, 23*time.Hour, "magic token expiry must be ~24h")
	require.Less(t, ttl, 25*time.Hour, "magic token expiry must be ~24h")

	// The minted token exchanges via POST /public/candidate/auth/verify into a
	// candidate JWT whose application list includes the just-applied job.
	verifyBody, _ := json.Marshal(map[string]string{"token": rowToken})
	vReq := httptest.NewRequest("POST", "/api/v1/public/candidate/auth/verify", bytes.NewReader(verifyBody))
	vReq.Header.Set("Content-Type", "application/json")
	vResp, err := app.Test(vReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, vResp.StatusCode)

	var verifyResult struct {
		Data struct {
			Token string `json:"token"`
			Email string `json:"email"`
		} `json:"data"`
	}
	err = json.NewDecoder(vResp.Body).Decode(&verifyResult)
	require.NoError(t, err)
	require.NotEmpty(t, verifyResult.Data.Token)
	require.Equal(t, email, verifyResult.Data.Email)

	// 4. The candidate's application list includes the just-applied job —
	// and carries NO interview invitation_token (L1 strip).
	listReq := httptest.NewRequest("GET", "/api/v1/candidate/portal/applications", nil)
	listReq.Header.Set("Authorization", "Bearer "+verifyResult.Data.Token)
	listResp, err := app.Test(listReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, listResp.StatusCode)

	var listResult struct {
		Data []struct {
			JobID           string  `json:"job_id"`
			JobTitle        string  `json:"job_title"`
			OrgName         string  `json:"org_name"`
			InvitationToken *string `json:"invitation_token"`
		} `json:"data"`
	}
	err = json.NewDecoder(listResp.Body).Decode(&listResult)
	require.NoError(t, err)
	require.Len(t, listResult.Data, 1)
	require.Equal(t, jobID.String(), listResult.Data[0].JobID)
	require.Equal(t, jobTitle, listResult.Data[0].JobTitle)
	require.Equal(t, "Portal Co", listResult.Data[0].OrgName)
}

// TestApplyDailyCapBoundary — pure unit: the per-email daily cap decision.
func TestApplyDailyCapBoundary(t *testing.T) {
	require.False(t, jobapi.ApplyCapExceeded(0))
	require.False(t, jobapi.ApplyCapExceeded(jobapi.MaxApplyPerEmailPerDay-1))
	require.True(t, jobapi.ApplyCapExceeded(jobapi.MaxApplyPerEmailPerDay))
	require.True(t, jobapi.ApplyCapExceeded(jobapi.MaxApplyPerEmailPerDay+5))
}

// TestPublicApplyEnqueuesPortalAccessEmail — 1b: after apply, exactly one
// portal-access email is enqueued to the supplied email containing a magic
// link; the HTTP response carries no credential (C9).
func TestPublicApplyEnqueuesPortalAccessEmail(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	minioStore := newApplyTestStore(t)

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)

	orgID := uuid.New()
	jobID := uuid.New()
	email := "portal-mail-" + uuid.NewString()[:8] + "@candidate.io"
	publicURL := "https://portal.example.com"

	seedActiveJob(t, pool, orgID, jobID, "Portal Mail Engineer")

	fake := &fakeEnqueuer{}
	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	publicHandler := jobapi.NewPublicJobHandler(
		pool, jobrepo.NewPostgresJobRepo(pool), cvrepo.NewPostgresCandidateRepo(pool),
		scrrepo.NewPostgresApplicationRepo(pool), minioStore, fake, portalRepo, publicURL,
	)

	app := fiber.New()
	app.Post("/api/v1/public/jobs/:id/apply", publicHandler.Apply)

	resp, err := app.Test(applyMultipart(t, jobID, "Mail Candidate", email), -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(bodyBytes), "portal_token")

	emails := fake.ofType(notifapp.TaskSendEmail)
	require.Len(t, emails, 2, "confirmation + portal access email must be enqueued")
	var portalMails int
	for _, e := range emails {
		payload, ok := e.payload.(notifapp.SendEmailPayload)
		require.True(t, ok)
		if payload.Type != notifapp.EmailTypePortalAccess {
			continue
		}
		portalMails++
		require.Equal(t, email, payload.To)
		tokenIdx := strings.LastIndex(payload.MagicLink, "?token=")
		require.Greater(t, tokenIdx, 0, "magic link must carry ?token=")
		magicToken := strings.TrimPrefix(payload.MagicLink[tokenIdx:], "?token=")
		require.NotEmpty(t, magicToken)
		require.True(t, strings.HasPrefix(payload.MagicLink, publicURL+"/candidate/portal"),
			"link must use the configured PublicURL: %s", payload.MagicLink)

		// The emailed token resolves to an unused candidate_otps row.
		var rowToken string
		err = pool.Raw(
			`SELECT token FROM candidate_otps WHERE LOWER(email) = ? AND used_at IS NULL AND token = ?`,
			email, magicToken).Row().Scan(&rowToken)
		require.NoError(t, err)
		require.Equal(t, magicToken, rowToken)
	}
	require.Equal(t, 1, portalMails, "exactly one portal access email")

	// The CV parse task is still enqueued alongside the email.
	require.Len(t, fake.ofType("parse_cv"), 1)
}

// TestPublicApplyPerEmailDailyCap429 — C9 abuse cap: more than
// MaxApplyPerEmailPerDay applications from the same (org, lower(email)) in
// 24h must get 429; a different email stays unaffected.
func TestPublicApplyPerEmailDailyCap429(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	minioStore := newApplyTestStore(t)

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)

	orgID := uuid.New()
	email := "cap-apply-" + uuid.NewString()[:8] + "@candidate.io"
	freshEmail := "cap-fresh-" + uuid.NewString()[:8] + "@candidate.io"
	publicURL := "http://localhost:5173"

	jobIDs := make([]uuid.UUID, jobapi.MaxApplyPerEmailPerDay+1)
	for i := range jobIDs {
		jobIDs[i] = uuid.New()
		seedActiveJob(t, pool, orgID, jobIDs[i], fmt.Sprintf("Cap Engineer %d", i))
	}

	fake := &fakeEnqueuer{}
	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	publicHandler := jobapi.NewPublicJobHandler(
		pool, jobrepo.NewPostgresJobRepo(pool), cvrepo.NewPostgresCandidateRepo(pool),
		scrrepo.NewPostgresApplicationRepo(pool), minioStore, fake, portalRepo, publicURL,
	)

	app := fiber.New()
	app.Post("/api/v1/public/jobs/:id/apply", publicHandler.Apply)

	for i := 0; i < jobapi.MaxApplyPerEmailPerDay; i++ {
		resp, err := app.Test(applyMultipart(t, jobIDs[i], "Cap Candidate", email), -1)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		require.Equalf(t, 201, resp.StatusCode, "apply #%d must succeed", i+1)
	}

	capped, err := app.Test(applyMultipart(t, jobIDs[jobapi.MaxApplyPerEmailPerDay], "Cap Candidate", email), -1)
	require.NoError(t, err)
	cappedBody, _ := io.ReadAll(capped.Body)
	require.Equalf(t, 429, capped.StatusCode, "body: %s", string(cappedBody))

	// A different email on the same org/job is unaffected (cap is per-email).
	fresh, err := app.Test(applyMultipart(t, jobIDs[jobapi.MaxApplyPerEmailPerDay], "Fresh Candidate", freshEmail), -1)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, fresh.Body)
	require.Equal(t, 201, fresh.StatusCode)
}
