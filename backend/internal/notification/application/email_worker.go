package application

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/intivai/backend/pkg/mailer"
	"github.com/rs/zerolog"
)

const TaskSendEmail = "send_email"

// Email types — typed constants, never string literals (asynq convention).
const (
	EmailTypeConfirmation    = "confirmation"
	EmailTypeInvitation      = "invitation"
	EmailTypeScorecard       = "scorecard"
	EmailTypeCandidateOTP    = "candidate_otp"
	EmailTypeCandidateReview = "candidate_review"
)

type SendEmailPayload struct {
	Type           string  `json:"type"`
	To             string  `json:"to"`
	CandidateName  string  `json:"candidate_name,omitempty"`
	JobTitle       string  `json:"job_title,omitempty"`
	InviteURL      string  `json:"invite_url,omitempty"`
	Score          float64 `json:"score,omitempty"`
	Recommendation string  `json:"recommendation,omitempty"`
	ReportURL      string  `json:"report_url,omitempty"`
	OTPCode        string  `json:"otp_code,omitempty"`
	MagicLink      string  `json:"magic_link,omitempty"`
}

type EmailWorker struct {
	mailer mailer.Mailer
	logger zerolog.Logger
}

func NewEmailWorker(m mailer.Mailer, logger zerolog.Logger) *EmailWorker {
	return &EmailWorker{mailer: m, logger: logger}
}

func (w *EmailWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskSendEmail, w.handle)
}

func (w *EmailWorker) handle(ctx context.Context, task *asynq.Task) error {
	var p SendEmailPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		// Permanently-bad payload — retrying cannot fix it.
		return asynq.SkipRetry
	}

	w.logger.Info().Str("type", p.Type).Str("to", p.To).Msg("processing email delivery")

	switch p.Type {
	case EmailTypeConfirmation:
		return w.mailer.SendApplicationConfirmation(ctx, p.To, p.CandidateName, p.JobTitle)
	case EmailTypeInvitation:
		return w.mailer.SendInterviewInvitation(ctx, p.To, p.CandidateName, p.JobTitle, p.InviteURL)
	case EmailTypeScorecard:
		return w.mailer.SendScorecardReady(ctx, p.To, p.CandidateName, p.JobTitle, p.Score, p.Recommendation, p.ReportURL)
	case EmailTypeCandidateOTP:
		return w.mailer.SendCandidateLoginOTP(ctx, p.To, p.OTPCode, p.MagicLink)
	case EmailTypeCandidateReview:
		return w.mailer.SendCandidateReview(ctx, p.To, p.CandidateName, p.InviteURL)
	default:
		// Unknown type is permanent — log and drop, never retry.
		w.logger.Warn().Str("type", p.Type).Msg("unknown email task type, skipping")
		return asynq.SkipRetry
	}
}
