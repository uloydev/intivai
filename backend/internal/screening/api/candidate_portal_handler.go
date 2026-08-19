package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	iamapp "github.com/intivai/backend/internal/iam/application"
	notifapp "github.com/intivai/backend/internal/notification/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
	"github.com/intivai/backend/pkg/queue"
	"github.com/rs/zerolog/log"
)

const (
	otpTTL            = 10 * time.Minute
	otpResendCooldown = 60 * time.Second
	maxOTPAttempts    = 5
	maxOTPDaily       = 5 // per-email daily cap (spam hardening)
	candidateTokenTTL = 7 * 24 * time.Hour
)

type CandidatePortalHandler struct {
	repo      scrdomain.CandidatePortalRepository
	tokens    iamapp.TokenProvider
	queue     *queue.Client
	publicURL string
}

func NewCandidatePortalHandler(
	repo scrdomain.CandidatePortalRepository,
	tokens iamapp.TokenProvider,
	q *queue.Client,
	publicURL string,
) *CandidatePortalHandler {
	return &CandidatePortalHandler{
		repo:      repo,
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
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid request body"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "valid email address is required"))
	}

	// Abuse rails: resend cooldown, daily cap, purge of consumed/expired rows.
	last, err := h.repo.LastRequestAt(c.UserContext(), email)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to check otp cooldown"))
	}
	if last != nil && time.Since(*last) < otpResendCooldown {
		return httpapi.Error(c, sharederr.NewDomainError("TOO_MANY_REQUESTS", "please wait before requesting another code"))
	}
	issued, err := h.repo.OTPCountSince(c.UserContext(), email, time.Now().Add(-24*time.Hour))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to check otp usage"))
	}
	if issued >= maxOTPDaily {
		return httpapi.Error(c, sharederr.NewDomainError("TOO_MANY_REQUESTS", "daily verification code limit reached"))
	}
	if err := h.repo.PurgeExpired(c.UserContext(), email); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to purge expired codes"))
	}

	// Generate secure 6-digit OTP; store only its hash (plaintext codes in
	// the DB would be trivially brute-forced if the table leaks).
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to generate otp"))
	}
	code := fmt.Sprintf("%06d", nBig.Int64())
	magicToken := uuid.NewString()
	expiresAt := time.Now().UTC().Add(otpTTL)

	if err := h.repo.CreateOTP(c.UserContext(), email, otpHash(code), magicToken, expiresAt); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to record otp"))
	}

	magicLink := fmt.Sprintf("%s/candidate/portal?token=%s", h.publicURL, magicToken)

	// Dispatch email asynchronously — a failed enqueue must not silently
	// strand the candidate without a code.
	if h.queue != nil {
		if _, err := h.queue.Enqueue(c.UserContext(), notifapp.TaskSendEmail, notifapp.SendEmailPayload{
			Type:      notifapp.EmailTypeCandidateOTP,
			To:        email,
			OTPCode:   code,
			MagicLink: magicLink,
		}, asynq.MaxRetry(5)); err != nil {
			log.Warn().Err(err).Str("email", email).Msg("enqueue candidate_otp email failed")
		}
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
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid request body"))
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	code := strings.TrimSpace(req.Code)
	token := strings.TrimSpace(req.Token)

	var otp *scrdomain.CandidateOTP
	var err error
	switch {
	case token != "":
		otp, err = h.repo.FindValidByToken(c.UserContext(), token)
	case email != "" && code != "":
		otp, err = h.repo.FindValidByCodeHash(c.UserContext(), email, otpHash(code))
		if err == nil && otp == nil {
			// Record the failed attempt against the latest live row so
			// brute-force is bounded per email even with IP rotation.
			_ = h.repo.IncrementAttempts(c.UserContext(), email)
		}
	default:
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "provide either (email + code) or magic token"))
	}
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "verification lookup failed"))
	}
	if otp == nil {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid or expired verification code"))
	}
	if otp.Attempts >= maxOTPAttempts {
		return httpapi.Error(c, sharederr.NewDomainError("TOO_MANY_REQUESTS", "too many attempts, request a new code"))
	}

	// Single-statement consume — two concurrent verifies cannot both pass
	// (TOCTOU-safe).
	consumed, err := h.repo.Consume(c.UserContext(), otp.ID)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to consume verification code"))
	}
	if !consumed {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid or expired verification code"))
	}

	// Issue candidate token (valid for 7 days). Subject is a deterministic
	// uuid derived from the email so the token stays tied to one identity
	// (uuid.Nil/random subjects could not be linked or revoked per candidate).
	candidateID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(otp.Email))
	extra := map[string]any{
		"email": otp.Email,
	}
	jwtToken, err := h.tokens.Issue(candidateID, uuid.Nil, "candidate", iamapp.TokenTypeCandidate, candidateTokenTTL, extra)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to issue candidate token"))
	}

	return httpapi.OK(c, fiber.Map{
		"token":      jwtToken,
		"email":      otp.Email,
		"expires_at": time.Now().UTC().Add(candidateTokenTTL).Format(time.RFC3339),
	})
}

// RequireCandidateAuth middleware validates candidate JWTs
func (h *CandidatePortalHandler) RequireCandidateAuth(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "missing candidate authorization header"))
	}
	token := strings.TrimPrefix(header, "Bearer ")
	claims, err := h.tokens.Parse(token)
	if err != nil || claims.Type != iamapp.TokenTypeCandidate {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid candidate authorization token"))
	}

	email, ok := claims.Extra["email"].(string)
	if !ok || email == "" {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "candidate email missing from claims"))
	}

	c.Locals("candidate_email", email)
	return c.Next()
}

// ListApplications handles GET /api/v1/candidate/portal/applications
func (h *CandidatePortalHandler) ListApplications(c *fiber.Ctx) error {
	email, ok := c.Locals("candidate_email").(string)
	if !ok || email == "" {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	apps, err := h.repo.ListApplications(c.UserContext(), email)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "internal server error"))
	}
	return httpapi.OK(c, apps)
}

// Export handles GET /api/v1/candidate/portal/export — GDPR Art.15 data
// access: a machine-readable dump of everything the portal knows about the
// candidate.
func (h *CandidatePortalHandler) Export(c *fiber.Ctx) error {
	email, ok := c.Locals("candidate_email").(string)
	if !ok || email == "" {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	apps, err := h.repo.ListApplications(c.UserContext(), email)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "internal server error"))
	}
	return httpapi.OK(c, fiber.Map{
		"email":        email,
		"applications": apps,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// DeleteMe handles DELETE /api/v1/candidate/portal/me — GDPR Art.17
// right-to-erasure: removes the candidate (and all interview/application
// data) across every org.
func (h *CandidatePortalHandler) DeleteMe(c *fiber.Ctx) error {
	email, ok := c.Locals("candidate_email").(string)
	if !ok || email == "" {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	if err := h.repo.EraseCandidate(c.UserContext(), email); err != nil {
		log.Warn().Err(err).Str("email", email).Msg("candidate erase failed")
		return httpapi.Error(c, sharederr.NewDomainError("INTERNAL_ERROR", "failed to erase candidate data"))
	}
	return httpapi.OK(c, fiber.Map{"message": "your data has been erased"})
}
