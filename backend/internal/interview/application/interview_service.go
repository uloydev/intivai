package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	gensvc "github.com/intivai/backend/internal/interview/domain/service"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
	"gorm.io/gorm"
)

const ticketTTL = 10 * time.Minute

// TaskEnqueuer — async evaluation retry seam (main wires the asynq client;
// tests pass nil).
type TaskEnqueuer interface {
	EnqueueEvaluation(ctx context.Context, orgID, interviewID string) error
}

// InterviewService — create interviews (recruiter), issue WS tickets
// (candidate, invitation token → short-lived JWT bound to session+interview).
type InterviewService struct {
	pool        *gorm.DB
	ivRepo      ivdomain.InterviewRepository
	tokenRepo   ivdomain.TokenRepository
	bank        ivdomain.QuestionBank
	appRepo     scrdomain.ApplicationRepository
	candRepo    cvdomain.CandidateRepository
	jobRepo     jobdomain.JobRepository
	contextRepo ctxdomain.ContextRepository
	store       *storage.Storage
	tokens      application.TokenProvider
	clock       ivdomain.Clock
	enqueuer    TaskEnqueuer
}

func NewInterviewService(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, tokenRepo ivdomain.TokenRepository,
	bank ivdomain.QuestionBank, appRepo scrdomain.ApplicationRepository, candRepo cvdomain.CandidateRepository,
	jobRepo jobdomain.JobRepository, contextRepo ctxdomain.ContextRepository, store *storage.Storage,
	tokens application.TokenProvider, clock ivdomain.Clock, enqueuer TaskEnqueuer) *InterviewService {
	return &InterviewService{pool: pool, ivRepo: ivRepo, tokenRepo: tokenRepo, bank: bank,
		appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo, contextRepo: contextRepo,
		store: store, tokens: tokens, clock: clock, enqueuer: enqueuer}
}

type CreateInterviewCommand struct {
	ApplicationID uuid.UUID
	QuestionCount int
}

type CreateInterviewResult struct {
	InterviewID    uuid.UUID `json:"interview_id"`
	Token          string    `json:"invitation_token"`
	ExpiresAt      time.Time `json:"expires_at"`
	ContextVersion int       `json:"context_version"`
}

// CreateInterview: load application → CV-gap questions → persist interview +
// question bank + invitation token (7-day, 32-char random).
func (s *InterviewService) CreateInterview(ctx context.Context, actor application.AuthContext, cmd CreateInterviewCommand) (*CreateInterviewResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	var result *CreateInterviewResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		app, err := s.appRepo.GetByID(tctx, cmd.ApplicationID)
		if err == scrdomain.ErrNotFound {
			return errors.NewNotFoundError("application", cmd.ApplicationID.String())
		}
		if err != nil {
			return err
		}
		if app.OrgID != actor.OrgID {
			return errors.NewDomainError("FORBIDDEN", "application belongs to another org")
		}
		if app.PassedScreening == nil || !*app.PassedScreening {
			return errors.NewDomainError("APPLICATION_NOT_PASSED", "only passed applications can be interviewed")
		}

		candidate, err := s.candRepo.GetByID(tctx, app.CandidateID)
		if err != nil {
			return err
		}
		job, err := s.jobRepo.GetByID(tctx, app.JobID)
		if err != nil {
			return err
		}
		if job.Status != jobdomain.StatusActive {
			return errors.NewDomainError("JOB_NOT_ACTIVE", "job is not active")
		}

		questions := s.generateQuestions(candidate, job, cmd.QuestionCount)
		domainQuestions := make([]ivdomain.Question, 0, len(questions))
		for i, q := range questions {
			domainQuestions = append(domainQuestions, ivdomain.Question{Idx: i + 1, Content: q.Prompt, Category: q.Category, Skill: q.Skill})
			if err := s.bank.Create(tctx, actor.OrgID, domainQuestions[i]); err != nil {
				return err
			}
		}

		now := s.clock.Now()
		iv, err := ivdomain.NewInterview(actor.OrgID, app.ID, domainQuestions, now.Add(7*24*time.Hour), s.clock)
		if err != nil {
			return err
		}
		// Pin the company-context version the interviewer will see (audit).
		contexts, err := s.contextRepo.ListContexts(tctx, actor.OrgID)
		if err != nil {
			return err
		}
		if len(contexts) > 0 {
			iv.ContextVersion = contexts[0].Version
		}
		if err := s.ivRepo.Create(tctx, iv); err != nil {
			return err
		}

		invite := &ivdomain.InvitationToken{
			ID: uuid.New(), OrgID: actor.OrgID, InterviewID: iv.ID,
			Token:     randomToken(),
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		}
		if err := s.tokenRepo.Create(tctx, invite); err != nil {
			return err
		}
		result = &CreateInterviewResult{InterviewID: iv.ID, Token: invite.Token, ExpiresAt: invite.ExpiresAt, ContextVersion: iv.ContextVersion}
		return nil
	})
	return result, err
}

type IssueTicketCommand struct {
	InterviewID     uuid.UUID
	InvitationToken string
}

type IssueTicketResult struct {
	Ticket    string    `json:"ticket"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IssueTicket: pre-auth validate invitation token → first start marks used →
// issue 10-min WS ticket bound to session_id + interview_id.
func (s *InterviewService) IssueTicket(ctx context.Context, cmd IssueTicketCommand) (*IssueTicketResult, error) {
	invite, status := s.tokenRepo.Validate(ctx, cmd.InvitationToken)
	switch status {
	case ivdomain.TokenValid:
		// ok
	case ivdomain.TokenUsed:
		// reconnect path — the same token stays valid for resume
		if invite == nil || invite.InterviewID != cmd.InterviewID {
			return nil, errors.NewDomainError("TOKEN_MISMATCH", "token does not match this interview")
		}
	case ivdomain.TokenExpired:
		return nil, errors.NewDomainError("TOKEN_EXPIRED", "invitation expired")
	case ivdomain.TokenRevoked:
		return nil, errors.NewDomainError("TOKEN_REVOKED", "invitation revoked")
	default:
		return nil, errors.NewDomainError("TOKEN_INVALID", "invalid invitation token")
	}
	if invite == nil || invite.InterviewID != cmd.InterviewID {
		return nil, errors.NewDomainError("TOKEN_MISMATCH", "token does not match this interview")
	}

	err := db.RunInTx(ctx, s.pool, invite.OrgID.String(), func(tctx context.Context) error {
		if err := s.tokenRepo.MarkUsed(tctx, cmd.InvitationToken); err != nil {
			return err
		}
		iv, err := s.ivRepo.GetByID(tctx, cmd.InterviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		return s.ivRepo.Update(tctx, iv) // touch: candidate entered
	})
	if err != nil {
		return nil, err
	}

	sessionID := uuid.New()
	extra := map[string]any{
		"session_id":   sessionID.String(),
		"interview_id": cmd.InterviewID.String(),
	}
	ticket, err := s.tokens.Issue(cmd.InterviewID, invite.OrgID, "candidate", application.TokenTypeWSTicket, ticketTTL, extra)
	if err != nil {
		return nil, err
	}
	return &IssueTicketResult{Ticket: ticket, SessionID: sessionID.String(), ExpiresAt: s.clock.Now().Add(ticketTTL)}, nil
}

// ComposePrompt builds the interview system prompt: default + tenant prompt +
// company context (latest versions) + safety rails. Repo reads run inside a
// tenant tx (RLS).
func (s *InterviewService) ComposePrompt(ctx context.Context, orgID uuid.UUID) (string, error) {
	in := gensvc.ComposerInput{DefaultPrompt: gensvc.DefaultInterviewerPrompt}
	err := db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		if p, err := s.contextRepo.GetLatestPrompt(tctx, orgID); err == nil {
			in.TenantPrompt = p.SystemPrompt
		}
		contexts, err := s.contextRepo.ListContexts(tctx, orgID)
		if err != nil {
			return err
		}
		if len(contexts) > 0 {
			latest := contexts[0]
			if reader, err := s.store.Download(ctx, latest.StoragePath); err == nil {
				buf := new(strings.Builder)
				_, _ = io.Copy(buf, reader)
				_ = reader.Close()
				in.CompanyContext = buf.String()
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return gensvc.ComposeSystemPrompt(in), nil
}

// VerifyInterviewOrg — nil when the interview belongs to orgID (used to gate
// recruiter auth tokens on voice rooms: the token's org must own the room).
// RLS on applications makes a foreign interview resolve as not-found.
func (s *InterviewService) VerifyInterviewOrg(ctx context.Context, orgID uuid.UUID, interviewID uuid.UUID) error {
	return db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			if err == ivdomain.ErrNotFound {
				return errors.NewDomainError("FORBIDDEN", "interview not found in this org")
			}
			return err
		}
		app, err := s.appRepo.GetByID(tctx, iv.ApplicationID)
		if err != nil {
			return err
		}
		if app.OrgID != orgID {
			return errors.NewDomainError("FORBIDDEN", "interview belongs to another org")
		}
		return nil
	})
}

// AnswerAndAdvance: record answer, persist, return the next question. Shallow
// answers (weakness, Research §2) produce a deterministic probe follow-up on
// the same topic; detailed answers move to the next planned question.
func (s *InterviewService) AnswerAndAdvance(ctx context.Context, orgID string, interviewID uuid.UUID, content string) (*ivdomain.Question, error) {
	return s.AnswerAndAdvanceWithPacing(ctx, orgID, interviewID, content, nil)
}

// AnswerAndAdvanceWithPacing: record candidate answer with pacing metrics and advance to next question.
func (s *InterviewService) AnswerAndAdvanceWithPacing(ctx context.Context, orgID string, interviewID uuid.UUID, content string, pacing *ivdomain.PacingMetrics) (*ivdomain.Question, error) {
	var next *ivdomain.Question
	var answered *ivdomain.Question
	err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		iv.ExpireIfNeeded()
		if iv.Status == ivdomain.StatusExpired {
			return errors.NewDomainError("INTERVIEW_EXPIRED", "interview expired")
		}
		answered = iv.NextQuestion()
		if err := iv.AnswerWithPacing(content, pacing); err != nil {
			return err
		}
		next = iv.NextQuestion()
		// Weakness probe: follow up on the CURRENT topic (the question just
		// answered) when the answer was shallow.
		if answered != nil && gensvc.ShouldProbe(gensvc.ProbeInput{Answer: content}) {
			probe := gensvc.ProbeQuestion(answered.Category, answered.Skill)
			if p, err := iv.InsertProbeAfter(answered.Idx, probe.Prompt, probe.Category, probe.Skill); err == nil {
				// NOTE: iv.OrgID is NOT hydrated by GetByID (interviews have no
				// org_id column; RLS resolves via applications) — use the arg.
				if err := s.bank.Create(tctx, uuid.MustParse(orgID), ivdomain.Question{Idx: p.Idx, Content: p.Content, Category: p.Category, Skill: p.Skill}); err == nil {
					next = p
				}
			}
		}
		if next == nil {
			_ = iv.Complete()
		}
		return s.ivRepo.Update(tctx, iv)
	})
	return next, err
}

// SessionRemaining calculates remaining seconds before the 30-minute global budget expires.
func (s *InterviewService) SessionRemaining(ctx context.Context, orgID string, interviewID uuid.UUID) int {
	var remaining = int(ivdomain.MaxInterviewDuration.Seconds())
	_ = db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err == nil && iv != nil {
			iv.SetClock(s.clock)
			remaining = iv.SessionRemaining()
		}
		return nil
	})
	return remaining
}

// GiveConsent records GDPR consent for the interview (invitation token
// auth, same validation as ticket issuance). Idempotent.
func (s *InterviewService) GiveConsent(ctx context.Context, interviewID uuid.UUID, invitationToken string) error {
	invite, status := s.tokenRepo.Validate(ctx, invitationToken)
	switch status {
	case ivdomain.TokenValid, ivdomain.TokenUsed:
		// ok — used tokens stay valid for the reconnect/consent path
	default:
		return errors.NewDomainError("TOKEN_INVALID", "invalid invitation token")
	}
	if invite == nil || invite.InterviewID != interviewID {
		return errors.NewDomainError("TOKEN_MISMATCH", "token does not match this interview")
	}
	return db.RunInTx(ctx, s.pool, invite.OrgID.String(), func(tctx context.Context) error {
		return s.ivRepo.SetConsent(tctx, interviewID)
	})
}

// StartInterview marks in_progress (first connect / resume). GDPR consent
// must be recorded first (CONSENT_REQUIRED) — candidates cannot be asked
// questions before agreeing.
func (s *InterviewService) StartInterview(ctx context.Context, orgID string, interviewID uuid.UUID) error {
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		iv.ExpireIfNeeded()
		if iv.Status == ivdomain.StatusExpired {
			return errors.NewDomainError("INTERVIEW_EXPIRED", "interview expired")
		}
		if !iv.ConsentGiven {
			return errors.NewDomainError("CONSENT_REQUIRED", "candidate consent must be recorded before the interview")
		}
		if err := iv.Start(); err != nil {
			return err
		}
		return s.ivRepo.Update(tctx, iv)
	})
}

// CurrentState — resume support: current (next unanswered) question + total
// count + interview status.
func (s *InterviewService) CurrentState(ctx context.Context, orgID string, interviewID uuid.UUID) (next *ivdomain.Question, total int, status ivdomain.Status, err error) {
	err = db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		iv.ExpireIfNeeded()
		status = iv.Status
		total = len(iv.Questions)
		if status == ivdomain.StatusInProgress {
			next = iv.NextQuestion()
		}
		return nil
	})
	return next, total, status, err
}

// RecentContext rebuilds the conversation history (assistant question + user
// answer pairs) from the persisted transcript, windowed to the last 10 Q&A.
// Used at WS connect/resume to seed the LLM context window.
func (s *InterviewService) RecentContext(ctx context.Context, orgID string, interviewID uuid.UUID) ([]gensvc.ContextMessage, error) {
	var history []gensvc.ContextMessage
	err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		for _, a := range iv.Answers {
			if a.Idx < 1 || a.Idx > len(iv.Questions) {
				continue
			}
			history = append(history,
				gensvc.ContextMessage{Role: gensvc.RoleAssistant, Content: iv.Questions[a.Idx-1].Content},
				gensvc.ContextMessage{Role: gensvc.RoleUser, Content: a.Content},
			)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gensvc.TrimContext(history, gensvc.DefaultContextWindow), nil
}

// Transcript returns question/answer pairs for the evaluator, in order.
func (s *InterviewService) Transcript(ctx context.Context, orgID string, interviewID uuid.UUID) ([]ivdomain.TranscriptPair, error) {
	var pairs []ivdomain.TranscriptPair
	err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		pairs = iv.TranscriptPairs()
		return nil
	})
	return pairs, err
}

// EvaluateAndPersist stores the report (evaluation JSONB). Idempotent: an
// existing evaluation wins (atomic WHERE guard in the repo) — retries never
// double-run or overwrite.
func (s *InterviewService) EvaluateAndPersist(ctx context.Context, orgID string, interviewID uuid.UUID, report []byte) error {
	err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		if len(iv.Evaluation) > 0 {
			return nil
		}
		return s.ivRepo.SaveEvaluation(tctx, interviewID, report)
	})
	if err == ivdomain.ErrEvaluationExists {
		return nil // lost the race; the other writer's report stands
	}
	return err
}

// RecordTelemetry records an anti-cheating telemetry event for an interview session.
func (s *InterviewService) RecordTelemetry(ctx context.Context, orgID string, interviewID uuid.UUID, event ivdomain.ProctoringEvent) error {
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		return s.ivRepo.RecordProctoringEvent(tctx, interviewID, event)
	})
}

// RecordCandidateTelemetry records a proctoring telemetry event for candidate using ticket or invitation token.
func (s *InterviewService) RecordCandidateTelemetry(ctx context.Context, interviewID uuid.UUID, token string, event ivdomain.ProctoringEvent) error {
	// First check if token is an invitation token
	invite, status := s.tokenRepo.Validate(ctx, token)
	if (status == ivdomain.TokenValid || status == ivdomain.TokenUsed) && invite != nil && invite.InterviewID == interviewID {
		return s.RecordTelemetry(ctx, invite.OrgID.String(), interviewID, event)
	}

	// Otherwise check if token is a ticket JWT
	claims, err := s.tokens.Parse(token)
	if err == nil && claims != nil && claims.Type == application.TokenTypeWSTicket {
		if iid, ok := claims.Extra["interview_id"].(string); ok && iid == interviewID.String() {
			return s.RecordTelemetry(ctx, claims.OrgID.String(), interviewID, event)
		}
	}

	return errors.NewDomainError("AUTH_UNAUTHORIZED", "valid ticket or invitation token required for telemetry")
}

// RecordCodingSession records a code sandbox snapshot on the interview.
// Column-scoped append — never rewrites the transcript (answer commits race
// keystroke/run traffic).
func (s *InterviewService) RecordCodingSession(ctx context.Context, orgID string, interviewID uuid.UUID, session ivdomain.CodingSession) error {
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		return s.ivRepo.RecordCodingSession(tctx, interviewID, session)
	})
}

// TouchInterview refreshes the interview activity marker (column-scoped
// update; the old full-row Update() could overwrite a concurrently persisted
// answer with a stale aggregate).
func (s *InterviewService) TouchInterview(ctx context.Context, orgID string, interviewID uuid.UUID) {
	_ = db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		return s.ivRepo.Touch(tctx, interviewID)
	})
}

// EnqueueEvaluation schedules the async retry worker (no-op without enqueuer).
func (s *InterviewService) EnqueueEvaluation(ctx context.Context, orgID string, interviewID uuid.UUID) error {
	if s.enqueuer == nil {
		return nil
	}
	return s.enqueuer.EnqueueEvaluation(ctx, orgID, interviewID.String())
}

func (s *InterviewService) generateQuestions(candidate *cvdomain.Candidate, job *jobdomain.Job, count int) []gensvc.Question {
	profile := gensvc.CandidateProfile{Skills: candidateSkills(candidate), Summary: candidateSummary(candidate)}
	reqs := gensvc.JobRequirements{Title: job.Title, Description: job.Description, RequiredSkills: job.RequiredSkills}
	return gensvc.GenerateQuestions(profile, reqs, count)
}

func candidateSkills(c *cvdomain.Candidate) []string {
	if len(c.CVStructured) == 0 {
		return nil
	}
	var rd struct {
		Skills []string `json:"skills"`
	}
	_ = json.Unmarshal(c.CVStructured, &rd)
	return rd.Skills
}

func candidateSummary(c *cvdomain.Candidate) string {
	if len(c.CVStructured) == 0 {
		return c.CVRawText
	}
	var rd struct {
		Summary string `json:"summary"`
	}
	_ = json.Unmarshal(c.CVStructured, &rd)
	return rd.Summary
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
