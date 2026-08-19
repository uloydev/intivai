package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CandidateOTP — a passwordless login code + magic-link token for the
// candidate portal. Only the code hash is stored (plaintext codes would be
// trivially brute-forced if the table leaked).
type CandidateOTP struct {
	ID        uuid.UUID
	Email     string
	CodeHash  string
	Token     string
	Attempts  int
	ExpiresAt time.Time
}

// CandidateApplicationView — one row of the candidate portal application
// list (joined across orgs via the candidate_applications_lookup function).
type CandidateApplicationView struct {
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

// CandidatePortalRepository — the candidate self-service surface (OTP auth +
// application lookups). Handlers never run OTP SQL directly and never
// duplicate the lookup logic (AGENTS.md convention).
type CandidatePortalRepository interface {
	CreateOTP(ctx context.Context, email, codeHash, token string, expiresAt time.Time) error
	// LastRequestAt — most recent un-consumed request for the email (cooldown).
	LastRequestAt(ctx context.Context, email string) (*time.Time, error)
	// OTPCountSince — requests issued for the email since `since` (daily cap).
	OTPCountSince(ctx context.Context, email string, since time.Time) (int, error)
	FindValidByToken(ctx context.Context, token string) (*CandidateOTP, error)
	FindValidByCodeHash(ctx context.Context, email, codeHash string) (*CandidateOTP, error)
	// IncrementAttempts — records a failed verification (per-email brute-force bound).
	IncrementAttempts(ctx context.Context, email string) error
	// Consume — single-statement consume; returns false when already used.
	Consume(ctx context.Context, id uuid.UUID) (bool, error)
	PurgeExpired(ctx context.Context, email string) error
	// EraseCandidate — GDPR right-to-erasure: removes all portal rows,
	// applications, interviews and candidates for an email across orgs.
	EraseCandidate(ctx context.Context, email string) error
	// ListApplications — the candidate's application rows across orgs
	// (SECURITY DEFINER lookup function; no tenant context exists yet).
	ListApplications(ctx context.Context, email string) ([]*CandidateApplicationView, error)
}
