package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamapp "github.com/intivai/backend/internal/iam/application"
	notifapp "github.com/intivai/backend/internal/notification/application"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/intivai/backend/pkg/queue"
	"gorm.io/gorm"
)

const (
	otpTTL            = 10 * time.Minute
	otpResendCooldown = 60 * time.Second
	maxOTPAttempts    = 5
	candidateTokenTTL = 7 * 24 * time.Hour
)

type CandidatePortalHandler struct {
	pool      *gorm.DB
	tokens    iamapp.TokenProvider
	queue     *queue.Client
	publicURL string
}

func NewCandidatePortalHandler(
	pool *gorm.DB,
	tokens iamapp.TokenProvider,
	q *queue.Client,
	publicURL string,
) *CandidatePortalHandler {
	return &CandidatePortalHandler{
		pool:      pool,
		tokens:    tokens,
		queue:     q,
		publicURL: strings.TrimSuffix(publicURL, "/"),
	}
}

type RequestOTPRequest struct {
	Email string `json:"email"`
}

func otpHash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// RequestOTP handles POST /api/v1/public/candidate/auth/otp
func (h *CandidatePortalHandler) RequestOTP(c *fiber.Ctx) error {
	var req RequestOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid email address is required"})
	}

	// Per-email resend cooldown + purge of consumed/expired rows (unbounded
	// OTP emailing is an abuse vector).
	var lastCreated *time.Time
	_ = h.pool.WithContext(c.UserContext()).Raw(
		`SELECT MAX(created_at) FROM candidate_otps WHERE LOWER(email) = ? AND used_at IS NULL`, email).Scan(&lastCreated)
	if lastCreated != nil && !lastCreated.IsZero() && time.Since(*lastCreated) < otpResendCooldown {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "please wait before requesting another code"})
	}
	_ = h.pool.WithContext(c.UserContext()).Exec(
		`DELETE FROM candidate_otps WHERE LOWER(email) = ? AND (expires_at < NOW() OR used_at IS NOT NULL)`, email)

	// Generate secure 6-digit OTP; store only its hash (plaintext codes in
	// the DB would be trivially brute-forced if the table leaks).
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate otp"})
	}
	code := fmt.Sprintf("%06d", nBig.Int64())
	magicToken := uuid.NewString()
	expiresAt := time.Now().UTC().Add(otpTTL)

	err = h.pool.WithContext(c.UserContext()).Exec(
		`INSERT INTO candidate_otps (id, email, code_hash, token, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, NOW())`,
		uuid.New(), email, otpHash(code), magicToken, expiresAt,
	).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to record otp"})
	}

	magicLink := fmt.Sprintf("%s/candidate/portal?token=%s", h.publicURL, magicToken)

	// Dispatch email asynchronously
	if h.queue != nil {
		_, _ = h.queue.Enqueue(c.UserContext(), notifapp.TaskSendEmail, notifapp.SendEmailPayload{
			Type:      notifapp.EmailTypeCandidateOTP,
			To:        email,
			OTPCode:   code,
			MagicLink: magicLink,
		})
	}

	return httpapi.OK(c, fiber.Map{
		"message":    "Verification code sent to your email.",
		"email":      email,
		"expires_in": int(otpTTL.Seconds()),
	})
}

type VerifyOTPRequest struct {
	Email string `json:"email,omitempty"`
	Code  string `json:"code,omitempty"`
	Token string `json:"token,omitempty"`
}

// VerifyOTP handles POST /api/v1/public/candidate/auth/verify
func (h *CandidatePortalHandler) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	code := strings.TrimSpace(req.Code)
	token := strings.TrimSpace(req.Token)

	var (
		otpID     uuid.UUID
		userEmail string
		attempts  int
	)
	if token != "" {
		row := h.pool.WithContext(c.UserContext()).Raw(
			`SELECT id, email, attempts FROM candidate_otps
			 WHERE token = ? AND used_at IS NULL AND expires_at > NOW()
			 ORDER BY created_at DESC LIMIT 1`, token).Row()
		if err := row.Scan(&otpID, &userEmail, &attempts); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired verification code"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "verification lookup failed"})
		}
	} else if email != "" && code != "" {
		row := h.pool.WithContext(c.UserContext()).Raw(
			`SELECT id, email, attempts FROM candidate_otps
			 WHERE LOWER(email) = ? AND code_hash = ? AND used_at IS NULL AND expires_at > NOW()
			 ORDER BY created_at DESC LIMIT 1`, email, otpHash(code)).Row()
		if err := row.Scan(&otpID, &userEmail, &attempts); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Record the failed attempt against the latest live row so
				// brute-force is bounded per email even with IP rotation.
				_ = h.pool.WithContext(c.UserContext()).Exec(
					`UPDATE candidate_otps SET attempts = attempts + 1
					 WHERE LOWER(email) = ? AND used_at IS NULL AND expires_at > NOW()`, email)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired verification code"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "verification lookup failed"})
		}
		if attempts >= maxOTPAttempts {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many attempts, request a new code"})
		}
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "provide either (email + code) or magic token"})
	}

	// Single-statement consume — two concurrent verifies cannot both pass
	// (TOCTOU-safe; the update error is checked, not ignored).
	res := h.pool.WithContext(c.UserContext()).Exec(
		`UPDATE candidate_otps SET used_at = NOW() WHERE id = ? AND used_at IS NULL`, otpID)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to consume verification code"})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired verification code"})
	}

	// Issue candidate token (valid for 7 days). Subject is a deterministic
	// uuid derived from the email so the token stays tied to one identity
	// (uuid.Nil/random subjects could not be linked or revoked per candidate).
	candidateID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(userEmail))
	extra := map[string]any{
		"email": userEmail,
	}
	jwtToken, err := h.tokens.Issue(candidateID, uuid.Nil, "candidate", iamapp.TokenTypeCandidate, candidateTokenTTL, extra)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to issue candidate token"})
	}

	return httpapi.OK(c, fiber.Map{
		"token":      jwtToken,
		"email":      userEmail,
		"expires_at": time.Now().UTC().Add(candidateTokenTTL).Format(time.RFC3339),
	})
}

// RequireCandidateAuth middleware validates candidate JWTs
func (h *CandidatePortalHandler) RequireCandidateAuth(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing candidate authorization header"})
	}
	token := strings.TrimPrefix(header, "Bearer ")
	claims, err := h.tokens.Parse(token)
	if err != nil || claims.Type != iamapp.TokenTypeCandidate {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid candidate authorization token"})
	}

	email, ok := claims.Extra["email"].(string)
	if !ok || email == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "candidate email missing from claims"})
	}

	c.Locals("candidate_email", email)
	return c.Next()
}

type CandidateApplicationDTO struct {
	ApplicationID     uuid.UUID  `json:"application_id"`
	OrgID             uuid.UUID  `json:"org_id"`
	OrgName           string     `json:"org_name"`
	OrgSlug           string     `json:"org_slug"`
	JobID             uuid.UUID  `json:"job_id"`
	JobTitle          string     `json:"job_title"`
	JobLocation       string     `json:"job_location"`
	JobEmploymentType string     `json:"job_employment_type"`
	CandidateID       uuid.UUID  `json:"candidate_id"`
	CandidateName     string     `json:"candidate_name"`
	CandidateEmail    string     `json:"candidate_email"`
	CVScore           *float64   `json:"cv_score"`
	PassedScreening   *bool      `json:"passed_screening"`
	ApplicationStatus string     `json:"application_status"`
	AppliedAt         string     `json:"applied_at"`
	InterviewID       *uuid.UUID `json:"interview_id"`
	InterviewStatus   *string    `json:"interview_status"`
	InterviewType     *string    `json:"interview_type"`
	InvitationToken   *string    `json:"invitation_token"`
	OverallScore      *float64   `json:"overall_score"`
	Recommendation    *string    `json:"recommendation"`
}

// ListApplications handles GET /api/v1/candidate/portal/applications
func (h *CandidatePortalHandler) ListApplications(c *fiber.Ctx) error {
	email, ok := c.Locals("candidate_email").(string)
	if !ok || email == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	rows, err := h.pool.WithContext(c.UserContext()).Raw(
		`SELECT application_id, org_id, org_name, org_slug, job_id, job_title, job_location, job_employment_type,
		        candidate_id, candidate_name, candidate_email, cv_score, passed_screening, application_status,
		        applied_at, interview_id, interview_status, interview_type, invitation_token, overall_score, recommendation
		 FROM candidate_applications_lookup(?)`, email).Rows()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	defer func() { _ = rows.Close() }()

	out := []*CandidateApplicationDTO{}
	for rows.Next() {
		var a CandidateApplicationDTO
		var appliedAt time.Time
		if err := rows.Scan(
			&a.ApplicationID, &a.OrgID, &a.OrgName, &a.OrgSlug, &a.JobID, &a.JobTitle, &a.JobLocation, &a.JobEmploymentType,
			&a.CandidateID, &a.CandidateName, &a.CandidateEmail, &a.CVScore, &a.PassedScreening, &a.ApplicationStatus,
			&appliedAt, &a.InterviewID, &a.InterviewStatus, &a.InterviewType, &a.InvitationToken, &a.OverallScore, &a.Recommendation,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}
		a.AppliedAt = appliedAt.Format(time.RFC3339)
		out = append(out, &a)
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return httpapi.OK(c, out)
}
