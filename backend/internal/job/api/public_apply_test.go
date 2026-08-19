package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	iamauth "github.com/intivai/backend/internal/iam/infrastructure/auth"
	jobapi "github.com/intivai/backend/internal/job/api"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	scrapi "github.com/intivai/backend/internal/screening/api"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/stretchr/testify/require"
)

// TestPublicApplyReturnsPortalToken — Issue-01 regression: a public apply must
// return a non-empty portal_token, and that token must exchange via
// POST /public/candidate/auth/verify into a candidate JWT whose application
// list includes the just-applied job.
func TestPublicApplyReturnsPortalToken(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("skipping integration test; TEST_REDIS_ADDR not set")
	}
	minioStore, err := storage.New(os.Getenv("TEST_MINIO_ENDPOINT"), os.Getenv("TEST_MINIO_ACCESS"), os.Getenv("TEST_MINIO_SECRET"), "intivai", false)
	if err != nil || minioStore == nil {
		t.Skip("TEST_MINIO_* not set")
	}

	pool, err := db.NewPool(context.Background(), url)
	require.NoError(t, err)
	ctx := context.Background()

	orgID := uuid.New()
	jobID := uuid.New()
	jobTitle := "Portal Token Engineer"
	email := "portal-apply-" + uuid.NewString()[:8] + "@candidate.io"

	// Seed a published active job (RLS: tenant tx scopes the inserts).
	err = db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
		tx, ok := db.TxFrom(tctx)
		if !ok {
			t.Fatal("no tenant tx")
		}
		if err := tx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?)`,
			orgID, "Portal Co", "portal-"+uuid.NewString()[:6]).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type, status, is_published)
			VALUES (?, ?, ?, ?, ?, ?, 'active', true)`,
			jobID, orgID, jobTitle, "Apply flow test", "Remote", "Full-time").Error
	})
	require.NoError(t, err)

	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	tokens := iamauth.NewJWTProvider("secret-test-key-32-chars-intivai-1234")
	publicHandler := jobapi.NewPublicJobHandler(
		pool, jobrepo.NewPostgresJobRepo(pool), cvrepo.NewPostgresCandidateRepo(pool),
		scrrepo.NewPostgresApplicationRepo(pool), minioStore, queue.NewClient(redisAddr), portalRepo,
	)
	portalHandler := scrapi.NewCandidatePortalHandler(portalRepo, tokens, nil, "http://localhost:5173")

	app := fiber.New()
	app.Post("/api/v1/public/jobs/:id/apply", publicHandler.Apply)
	app.Post("/api/v1/public/candidate/auth/verify", portalHandler.VerifyOTP)
	app.Get("/api/v1/candidate/portal/applications", portalHandler.RequireCandidateAuth, portalHandler.ListApplications)

	// 1. Public apply (multipart, minimal PDF).
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("name", "Portal Candidate"))
	require.NoError(t, mw.WriteField("email", email))
	fw, err := mw.CreateFormFile("file", "resume.pdf")
	require.NoError(t, err)
	_, err = fw.Write([]byte("%PDF-1.4 fake resume for apply flow"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest("POST", "/api/v1/public/jobs/"+jobID.String()+"/apply", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	var applyResult struct {
		Data struct {
			CandidateID string `json:"candidate_id"`
			JobID       string `json:"job_id"`
			Status      string `json:"status"`
			Message     string `json:"message"`
			PortalToken string `json:"portal_token"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&applyResult)
	require.NoError(t, err)
	require.Equal(t, "submitted", applyResult.Data.Status)
	require.NotEmpty(t, applyResult.Data.PortalToken, "apply response must carry a portal_token")

	// 2. The magic-token row exists, unused, expiring ~24h from now.
	var rowToken string
	var expiresAt time.Time
	err = pool.WithContext(ctx).Raw(
		`SELECT token, expires_at FROM candidate_otps WHERE LOWER(email) = ? AND used_at IS NULL AND token = ?
		 ORDER BY created_at DESC LIMIT 1`,
		email, applyResult.Data.PortalToken,
	).Row().Scan(&rowToken, &expiresAt)
	require.NoError(t, err)
	require.Equal(t, applyResult.Data.PortalToken, rowToken)
	ttl := time.Until(expiresAt)
	require.Greater(t, ttl, 23*time.Hour, "magic token expiry must be ~24h")
	require.Less(t, ttl, 25*time.Hour, "magic token expiry must be ~24h")

	// 3. Exchange the portal token for a candidate JWT.
	verifyBody, _ := json.Marshal(map[string]string{"token": applyResult.Data.PortalToken})
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

	// 4. The candidate's application list includes the just-applied job.
	listReq := httptest.NewRequest("GET", "/api/v1/candidate/portal/applications", nil)
	listReq.Header.Set("Authorization", "Bearer "+verifyResult.Data.Token)
	listResp, err := app.Test(listReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, listResp.StatusCode)

	var listResult struct {
		Data []struct {
			JobID    string `json:"job_id"`
			JobTitle string `json:"job_title"`
			OrgName  string `json:"org_name"`
		} `json:"data"`
	}
	err = json.NewDecoder(listResp.Body).Decode(&listResult)
	require.NoError(t, err)
	require.Len(t, listResult.Data, 1)
	require.Equal(t, jobID.String(), listResult.Data[0].JobID)
	require.Equal(t, jobTitle, listResult.Data[0].JobTitle)
	require.Equal(t, "Portal Co", listResult.Data[0].OrgName)
}
