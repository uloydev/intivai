package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamauth "github.com/intivai/backend/internal/iam/infrastructure/auth"
	scrapi "github.com/intivai/backend/internal/screening/api"
	"github.com/intivai/backend/pkg/db"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("skipping integration test; TEST_DATABASE_URL not set")
	}
	pool, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	require.NoError(t, err)
	return pool
}

func TestCandidatePortal_OTPAndApplicationLookupFlow(t *testing.T) {
	pool := setupTestDB(t)
	tokens := iamauth.NewJWTProvider("secret-test-key-32-chars-intivai-1234")
	handler := scrapi.NewCandidatePortalHandler(pool, tokens, nil, "http://localhost:5173")

	app := fiber.New()
	app.Post("/api/v1/public/candidate/auth/otp", handler.RequestOTP)
	app.Post("/api/v1/public/candidate/auth/verify", handler.VerifyOTP)
	app.Get("/api/v1/candidate/portal/applications", handler.RequireCandidateAuth, handler.ListApplications)

	candidateEmail := "portal-test-" + uuid.NewString()[:8] + "@candidate.io"

	// 1. Request OTP
	body, _ := json.Marshal(map[string]string{"email": candidateEmail})
	req := httptest.NewRequest("POST", "/api/v1/public/candidate/auth/otp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	// 2. Fetch the magic token from DB to verify (codes are stored hashed)
	var magicToken string
	err = pool.WithContext(context.Background()).Raw(
		`SELECT token FROM candidate_otps WHERE email = ? AND used_at IS NULL ORDER BY created_at DESC LIMIT 1`,
		candidateEmail,
	).Scan(&magicToken).Error
	require.NoError(t, err)
	require.NotEmpty(t, magicToken)

	// 3. Verify via the magic-link token and obtain Candidate JWT
	verifyBody, _ := json.Marshal(map[string]string{"token": magicToken})
	vReq := httptest.NewRequest("POST", "/api/v1/public/candidate/auth/verify", bytes.NewReader(verifyBody))
	vReq.Header.Set("Content-Type", "application/json")
	vResp, err := app.Test(vReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, vResp.StatusCode)

	var vResult struct {
		Data struct {
			Token string `json:"token"`
			Email string `json:"email"`
		} `json:"data"`
	}
	err = json.NewDecoder(vResp.Body).Decode(&vResult)
	require.NoError(t, err)
	require.NotEmpty(t, vResult.Data.Token)
	require.Equal(t, candidateEmail, vResult.Data.Email)

	// 4. Create an Org, Job, Candidate and Application to verify ListApplications
	orgID := uuid.New()
	jobID := uuid.New()
	candID := uuid.New()
	appID := uuid.New()

	err = db.RunInTx(context.Background(), pool, orgID.String(), func(tx context.Context) error {
		gormTx, ok := db.TxFrom(tx)
		require.True(t, ok)
		_ = gormTx.Exec(`INSERT INTO orgs (id, name, slug) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, orgID, "Acme Tech", "acme-"+uuid.NewString()[:6])
		_ = gormTx.Exec(`INSERT INTO jobs (id, org_id, title, description, location, employment_type) VALUES (?, ?, ?, ?, ?, ?)`,
			jobID, orgID, "Senior Go Architect", "Lead backend", "Remote", "Full-time")
		_ = gormTx.Exec(`INSERT INTO candidates (id, org_id, name, email) VALUES (?, ?, ?, ?)`,
			candID, orgID, "Jane Candidate", candidateEmail)
		_ = gormTx.Exec(`INSERT INTO applications (id, org_id, candidate_id, job_id, cv_score, passed_screening, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			appID, orgID, candID, jobID, 88.5, true, "screening")
		return nil
	})
	require.NoError(t, err)

	// 5. Query candidate applications with Bearer JWT
	listReq := httptest.NewRequest("GET", "/api/v1/candidate/portal/applications", nil)
	listReq.Header.Set("Authorization", "Bearer "+vResult.Data.Token)
	listResp, err := app.Test(listReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, listResp.StatusCode)

	var listResult struct {
		Data []struct {
			ApplicationID string  `json:"application_id"`
			JobTitle      string  `json:"job_title"`
			OrgName       string  `json:"org_name"`
			CVScore       float64 `json:"cv_score"`
		} `json:"data"`
	}
	err = json.NewDecoder(listResp.Body).Decode(&listResult)
	require.NoError(t, err)
	require.Len(t, listResult.Data, 1)
	require.Equal(t, "Senior Go Architect", listResult.Data[0].JobTitle)
	require.Equal(t, "Acme Tech", listResult.Data[0].OrgName)
	require.Equal(t, 88.5, listResult.Data[0].CVScore)
}
