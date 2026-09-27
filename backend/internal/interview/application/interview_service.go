package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
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
	jobapp "github.com/intivai/backend/internal/job/application"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const ticketTTL = 10 * time.Minute

// TaskEnqueuer — async evaluation and notification seam (main wires the asynq client;
// tests pass nil).
type TaskEnqueuer interface {
	EnqueueEvaluation(ctx context.Context, orgID, interviewID string) error
	EnqueueInterviewInvitation(ctx context.Context, to, name, jobTitle, interviewID, inviteToken string) error
	EnqueueHumanRequest(ctx context.Context, to, candidateName, jobTitle, interviewID string) error
}

// InterviewService — create interviews (recruiter), issue WS tickets
// (candidate, invitation token → short-lived JWT bound to session+interview).
// OrgQALimitReader — driven port for the org-configurable candidate Q&A cap
// (D3/B4). Implemented in cmd/server over the IAM org repo; defaults to
// DefaultQALimit when unset or unreadable so the feature always has a bound.
type OrgQALimitReader interface {
	CandidateQALimit(ctx context.Context, orgID uuid.UUID) (int, error)
}

// DefaultQALimit — per-interview candidate questions answered when the org has
// not configured an explicit cap (D3).
const DefaultQALimit = 10

type InterviewService struct {
	pool        *gorm.DB
	ivRepo      ivdomain.InterviewRepository
	tokenRepo   ivdomain.TokenRepository
	bank        ivdomain.QuestionBank
	appRepo     scrdomain.ApplicationRepository
	candRepo    cvdomain.CandidateRepository
	jobRepo     jobdomain.JobRepository
	jobCandRepo jobdomain.CandidateContextRepository // per-job Q&A context (D2), pinned at connect
	contextRepo ctxdomain.ContextRepository
	store       *storage.Storage
	tokens      application.TokenProvider
	clock       ivdomain.Clock
	enqueuer    TaskEnqueuer
	orgQALimit  OrgQALimitReader
	log         zerolog.Logger
	// completeFn applies the terminal in_progress → completed transition.
	// Seam for tests (D25): inject a failure to prove the error is logged and
	// surfaced, not swallowed. Defaults to the domain Complete method.
	completeFn func(*ivdomain.Interview) error
}

func NewInterviewService(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, tokenRepo ivdomain.TokenRepository,
	bank ivdomain.QuestionBank, appRepo scrdomain.ApplicationRepository, candRepo cvdomain.CandidateRepository,
	jobRepo jobdomain.JobRepository, jobCandRepo jobdomain.CandidateContextRepository, contextRepo ctxdomain.ContextRepository,
	store *storage.Storage, tokens application.TokenProvider, clock ivdomain.Clock, enqueuer TaskEnqueuer,
	orgQALimit OrgQALimitReader, log zerolog.Logger) *InterviewService {
	return &InterviewService{pool: pool, ivRepo: ivRepo, tokenRepo: tokenRepo, bank: bank,
		appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo, jobCandRepo: jobCandRepo, contextRepo: contextRepo,
		store: store, tokens: tokens, clock: clock, enqueuer: enqueuer, orgQALimit: orgQALimit, log: log,
		completeFn: (*ivdomain.Interview).Complete}
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
	// EmailEnqueued — false + EmailError set when the invitation email could
	// NOT be enqueued (D9): the interview + token still exist and the email
	// can be retried, but the recruiter MUST be told instead of it failing
	// silently.
	EmailEnqueued bool   `json:"email_enqueued"`
	EmailError    string `json:"email_error,omitempty"`
}

// CreateInterview: load application → CV-gap questions → persist interview +
// question bank + invitation token (7-day, 32-char random).
func (s *InterviewService) CreateInterview(ctx context.Context, actor application.AuthContext, cmd CreateInterviewCommand) (*CreateInterviewResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	var result *CreateInterviewResult
	var candEmail, candName, jobTitle string
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

		candEmail = candidate.Email
		candName = candidate.Name
		jobTitle = job.Title

		// B4: load the job's stored QuestionSet (generated at publish, D5) so
		// the interview carries the SAME questions AND their descriptive
		// context/expectation framing for every candidate. Falls back to the
		// deterministic templates when no usable set exists.
		rawSet, err := jobapp.LoadQuestionSet(tctx, job.ID)
		if err != nil {
			return err
		}
		set, _ := jobdomain.SelectQuestionSet(rawSet)
		questions, err := s.generateQuestions(candidate, job, cmd.QuestionCount, set)
		if err != nil {
			return err
		}
		domainQuestions := make([]ivdomain.Question, 0, len(questions))
		for i, q := range questions {
			// Fix B4 gap: persist the stored-set framing so it isn't dropped.
			domainQuestions = append(domainQuestions, ivdomain.Question{
				Idx: i + 1, Content: q.Prompt, Category: q.Category, Skill: q.Skill,
				Context: q.Context, Expectation: q.Expectation,
			})
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
		inviteToken, err := randomToken()
		if err != nil {
			return fmt.Errorf("generate invitation token: %w", err)
		}

		invite := &ivdomain.InvitationToken{
			ID: uuid.New(), OrgID: actor.OrgID, InterviewID: iv.ID,
			Token:     inviteToken,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		}
		if err := s.tokenRepo.Create(tctx, invite); err != nil {
			return err
		}
		result = &CreateInterviewResult{InterviewID: iv.ID, Token: invite.Token, ExpiresAt: invite.ExpiresAt, ContextVersion: iv.ContextVersion}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.enqueuer != nil && candEmail != "" && result != nil {
		if err := s.enqueuer.EnqueueInterviewInvitation(ctx, candEmail, candName, jobTitle, result.InterviewID.String(), result.Token); err != nil {
			// D9: the email is the ONLY delivery path for the invite token —
			// a silent drop ghosts the candidate. Surface it on the response
			// (the interview itself succeeded; email is retryable).
			s.log.Warn().Err(err).Str("interview_id", result.InterviewID.String()).Msg("failed to enqueue interview invitation email")
			result.EmailEnqueued = false
			result.EmailError = "interview created but invitation email could not be queued; retry an invite"
		} else {
			result.EmailEnqueued = true
		}
	}
	return result, nil
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
	extra := application.TokenExtra{SessionID: sessionID.String(), InterviewID: cmd.InterviewID.String()}
	ticket, err := s.tokens.Issue(cmd.InterviewID, invite.OrgID, "candidate", application.TokenTypeWSTicket, ticketTTL, extra)
	if err != nil {
		return nil, err
	}
	return &IssueTicketResult{Ticket: ticket, SessionID: sessionID.String(), ExpiresAt: s.clock.Now().Add(ticketTTL)}, nil
}

// RequestHuman: candidate requests a human interviewer. Sets the flag and
// notifies the org's recruiters/admins by email. Auth accepts either the
// invitation token (portal flow) or the WS ticket JWT (in-session request).
func (s *InterviewService) RequestHuman(ctx context.Context, interviewID uuid.UUID, invitationToken string) error {
	invite, status := s.tokenRepo.Validate(ctx, invitationToken)
	if status != ivdomain.TokenValid && status != ivdomain.TokenUsed {
		// Fall back to the WS ticket JWT — the Chat page only holds the ticket.
		claims, err := s.tokens.Parse(invitationToken)
		if err != nil || claims == nil || claims.Type != application.TokenTypeWSTicket {
			return errors.NewDomainError("TOKEN_INVALID", "invalid invitation token")
		}
		if claims.Extra.InterviewID != interviewID.String() {
			return errors.NewDomainError("TOKEN_MISMATCH", "token does not match this interview")
		}
		invite = &ivdomain.InvitationToken{OrgID: claims.OrgID, InterviewID: interviewID}
	} else if invite == nil || invite.InterviewID != interviewID {
		return errors.NewDomainError("TOKEN_MISMATCH", "token does not match this interview")
	}

	var notify []struct {
		Email    string
		CandName string
		JobTitle string
	}
	err := db.RunInTx(ctx, s.pool, invite.OrgID.String(), func(tctx context.Context) error {
		if err := s.ivRepo.SetHumanRequested(tctx, interviewID, true); err != nil {
			return err
		}
		tx, ok := db.TxFrom(tctx)
		if !ok {
			return errors.NewDomainError("INTERNAL_ERROR", "no transaction")
		}
		return tx.Raw(
			`SELECT u.email, c.name AS cand_name, j.title AS job_title
			 FROM applications a
			 JOIN candidates c ON c.id = a.candidate_id
			 JOIN jobs j ON j.id = a.job_id
			 JOIN users u ON u.org_id = a.org_id AND u.role IN ('admin', 'recruiter')
			 WHERE a.id = (SELECT application_id FROM interviews WHERE id = $1)`,
			interviewID).Scan(&notify).Error
	})
	if err != nil {
		return err
	}

	if s.enqueuer != nil {
		for _, n := range notify {
			if n.Email == "" {
				continue
			}
			if err := s.enqueuer.EnqueueHumanRequest(ctx, n.Email, n.CandName, n.JobTitle, interviewID.String()); err != nil {
				s.log.Warn().Err(err).Str("email", n.Email).Str("interview_id", interviewID.String()).Msg("failed to enqueue human request notification")
			}
		}
	}
	return nil
}

// ConnectContexts — everything a WS connection pins at connect time.
type ConnectContexts struct {
	// Prompt is the composed interviewer system prompt (default + tenant +
	// company + job context + safety rails last).
	Prompt string
	// QAContext is the ONLY material the grounded candidate-question answer
	// path may use — no safety rails, no live DB.
	QAContext string
	// GroundingUnavailable is true when the QA grounding could not be
	// composed reliably (J11): the candidate-question answer path must fail
	// CLOSED — refuse rather than run the LLM against partial/no context.
	GroundingUnavailable bool
}

// ComposeConnectContexts builds the interviewer prompt AND the Q&A grounding
// from ONE tenant transaction plus ONE company-context download. Separate
// loads (I12) doubled connect-time reads/storage egress and let a recruiter
// save between them pin different context versions into the prompt vs the QA
// grounding. Job-context lookup failures degrade gracefully: the prompt still
// composes, grounding just carries less material.
func (s *InterviewService) ComposeConnectContexts(ctx context.Context, orgID uuid.UUID, interviewID uuid.UUID) (ConnectContexts, error) {
	var out ConnectContexts
	in := gensvc.ComposerInput{DefaultPrompt: gensvc.DefaultInterviewerPrompt}
	var qa strings.Builder
	var contextPath string
	err := db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		p, err := s.contextRepo.GetLatestPrompt(tctx, orgID)
		if err == nil {
			in.TenantPrompt = p.SystemPrompt
		} else if !stderrors.Is(err, ctxdomain.ErrNotFound) {
			return fmt.Errorf("load tenant prompt: %w", err)
		}
		contexts, err := s.contextRepo.ListContexts(tctx, orgID)
		if err != nil {
			return fmt.Errorf("list company contexts: %w", err)
		}
		if len(contexts) > 0 {
			contextPath = contexts[0].StoragePath
		}
		// D2: pin the per-job candidate context captured at connect so
		// mid-interview recruiter edits cannot change what the candidate hears.
		if s.jobCandRepo != nil {
			iv, err := s.ivRepo.GetByID(tctx, interviewID)
			if err != nil {
				// Interview row absent (test fixtures / pre-interview):
				// there is no job-context grounding to pin, but the interview
				// prompt still composes — this is NOT a grounding LOAD error.
				if !stderrors.Is(err, ivdomain.ErrNotFound) {
					return fmt.Errorf("load interview for grounding: %w", err)
				}
			} else {
				app, err := s.appRepo.GetByID(tctx, iv.ApplicationID)
				if err != nil {
					return fmt.Errorf("load application for grounding: %w", err)
				}
				cc, err := s.jobCandRepo.GetByJobID(tctx, app.JobID)
				if err != nil && !stderrors.Is(err, jobdomain.ErrNotFound) {
					// J11 identical policy: transport/DB failure in the
					// grounding backend is NOT "no context" — answering with
					// an empty grounding block would fail OPEN.
					return fmt.Errorf("load job candidate context for grounding: %w", err)
				}
				if err == nil && cc.Content != "" {
					in.JobContext = cc.Content
					qa.WriteString("Job candidate context:\n")
					qa.WriteString(cc.Content)
					qa.WriteString("\n\n")
				}
			}
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	company := ""
	if contextPath != "" {
		reader, err := s.store.Download(ctx, contextPath)
		if err != nil {
			return out, fmt.Errorf("download company context: %w", err)
		}
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, reader); err != nil {
			_ = reader.Close()
			return out, fmt.Errorf("read company context: %w", err)
		}
		if err := reader.Close(); err != nil {
			return out, fmt.Errorf("close company context: %w", err)
		}
		company = buf.String()
		in.CompanyContext = company
	}
	if company != "" {
		qa.WriteString("Company context:\n")
		qa.WriteString(company)
	}
	out.Prompt = gensvc.ComposeSystemPrompt(in)
	out.QAContext = qa.String()
	// J11 fail-closed: the flag is set ONLY when a grounding load FAILED (the
	// caller returns error in those cases). A job/company that legitimately
	// has no context yet still answers (the prompt instructs the model to say
	// "not in context" instead of inventing) — that is the existing design.
	return out, nil
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

// TopicDialogueResult holds the state result of processing a multi-turn topic dialogue exchange.
type TopicDialogueResult struct {
	NextQuestion    *ivdomain.Question
	IsTopicComplete bool
	CurrentTurn     int
	MaxTurns        int
	TotalQuestions  int
	// TransitionErr carries a non-fatal Complete-transition failure (D25):
	// the dialogue still advances (LLM-failure advance semantics), but the
	// WS layer emits it on the standard error-frame path instead of the
	// error vanishing into `_ =`.
	TransitionErr error
}

// ProcessTopicDialogue records a candidate dialogue turn (reply or advance), updates pacing telemetry,
// and determines whether the active topic remains open for further discussion or transitions to the next question.
func (s *InterviewService) ProcessTopicDialogue(ctx context.Context, orgID string, interviewID uuid.UUID, content string, action string, pacing *ivdomain.PacingMetrics) (*TopicDialogueResult, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.NewDomainError("INVALID_ORG_ID", "invalid organization id")
	}
	var res TopicDialogueResult
	err = db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		iv.ExpireIfNeeded()
		if iv.Status == ivdomain.StatusExpired {
			return errors.NewDomainError("INTERVIEW_EXPIRED", "interview expired")
		}

		answered := iv.NextQuestion()
		next, isComplete, turn, maxTurns, err := iv.ProcessTopicDialogue(content, action, pacing)
		if err != nil {
			return err
		}

		if isComplete && answered != nil && !answered.IsProbe && gensvc.ShouldProbe(gensvc.ProbeInput{Answer: content}) {
			probe := gensvc.ProbeQuestion(answered.Category, answered.Skill)
			p, insertErr := iv.InsertProbeAfter(answered.Idx, probe.Prompt, probe.Category, probe.Skill)
			if insertErr != nil {
				// D25: never swallow — log with interview id + stage; the
				// dialogue continues without the probe question.
				s.log.Error().Err(insertErr).Str("interview_id", interviewID.String()).Str("stage", "probe_insert").
					Msg("probe insertion failed; continuing without probe")
			} else if createErr := s.bank.Create(tctx, orgUUID, ivdomain.Question{Idx: p.Idx, Content: p.Content, Category: p.Category, Skill: p.Skill, IsProbe: true}); createErr != nil {
				s.log.Error().Err(createErr).Str("interview_id", interviewID.String()).Str("stage", "probe_persist").
					Msg("probe persist failed; continuing without probe")
			} else {
				next = p
			}
		}

		var transitionErr error
		if isComplete && next == nil {
			if completeErr := s.completeFn(iv); completeErr != nil {
				// D25: e.g. racing ExpireIfNeeded flipped the status. The
				// answer is already recorded and the advance stands, but the
				// failure must not vanish: log it and let the WS layer emit
				// the standard error frame.
				s.log.Error().Err(completeErr).Str("interview_id", interviewID.String()).Str("stage", "complete").
					Msg("interview complete transition failed")
				transitionErr = completeErr
			}
		}

		res = TopicDialogueResult{
			NextQuestion:    next,
			IsTopicComplete: isComplete,
			CurrentTurn:     turn,
			MaxTurns:        maxTurns,
			TotalQuestions:  len(iv.Questions),
			TransitionErr:   transitionErr,
		}

		return s.ivRepo.Update(tctx, iv)
	})
	return &res, err
}

// AnswerAndAdvance: record answer, persist, return the next question.
func (s *InterviewService) AnswerAndAdvance(ctx context.Context, orgID string, interviewID uuid.UUID, content string) (*ivdomain.Question, error) {
	next, _, err := s.AnswerAndAdvanceWithPacing(ctx, orgID, interviewID, content, nil)
	return next, err
}

// AnswerAndAdvanceWithPacing: record candidate answer with pacing metrics and advance to next question.
func (s *InterviewService) AnswerAndAdvanceWithPacing(ctx context.Context, orgID string, interviewID uuid.UUID, content string, pacing *ivdomain.PacingMetrics) (*ivdomain.Question, int, error) {
	res, err := s.ProcessTopicDialogue(ctx, orgID, interviewID, content, "advance", pacing)
	if err != nil {
		return nil, 0, err
	}
	return res.NextQuestion, res.TotalQuestions, nil
}

// SessionRemaining calculates remaining seconds before the 30-minute global budget expires.
func (s *InterviewService) SessionRemaining(ctx context.Context, orgID string, interviewID uuid.UUID) int {
	var remaining = int(ivdomain.MaxInterviewDuration.Seconds())
	if err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err == nil && iv != nil {
			iv.SetClock(s.clock)
			remaining = iv.SessionRemaining()
		}
		return nil
	}); err != nil {
		s.log.Warn().Err(err).Str("interview_id", interviewID.String()).Msg("session remaining lookup failed; using full budget")
	}
	return remaining
}

// GetInvitePreview returns candidate-safe metadata for the invite page pre-flight check.
func (s *InterviewService) GetInvitePreview(ctx context.Context, token string) (*ivdomain.InvitePreview, error) {
	return s.tokenRepo.GetPreview(ctx, token)
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

	// Otherwise check if token is a ticket JWT — verify org_id matches interview's org
	claims, err := s.tokens.Parse(token)
	if err == nil && claims != nil && claims.Type == application.TokenTypeWSTicket {
		if claims.Extra.InterviewID == interviewID.String() {
			err = db.RunInTx(ctx, s.pool, claims.OrgID.String(), func(tctx context.Context) error {
				iv, e := s.ivRepo.GetByID(tctx, interviewID)
				if e != nil {
					return e
				}
				if iv.OrgID != claims.OrgID {
					return errors.NewDomainError("AUTH_FORBIDDEN", "ticket org does not match this interview")
				}
				return s.ivRepo.RecordProctoringEvent(tctx, interviewID, event)
			})
			if err != nil {
				var de *errors.DomainError
				if stderrors.As(err, &de) {
					return err
				}
				return errors.NewDomainError("AUTH_UNAUTHORIZED", "valid ticket or invitation token required for telemetry")
			}
			return nil
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
func (s *InterviewService) TouchInterview(ctx context.Context, orgID string, interviewID uuid.UUID) error {
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
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

// ErrQALimitExceeded — the per-interview candidate Q&A cap (D3) was reached.
// The handler turns this into a polite refusal qa_answer frame, not an error.
var ErrQALimitExceeded = errors.NewDomainError("QA_LIMIT_EXCEEDED", "the candidate question limit for this interview has been reached")

// ErrInterviewNotActive — candidate input arrived on an expired or completed
// interview. Also surfaced as a refusal qa_answer frame.
var ErrInterviewNotActive = errors.NewDomainError("INTERVIEW_NOT_ACTIVE", "this interview is no longer active")

const maxStoredQAAnswerRunes = 4000

// clampRunes bounds persisted text length; the parse layer rejects overlength
// candidate frames, this defends the service against direct callers and caps
// LLM-authored answers (I13).
func clampRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// persistExpireIfDue commits clock-driven expiry INDEPENDENTLY of any later
// refusal: a gate that returns an error from inside RunInTx rolls the whole
// transaction back, which would silently erase the honest 'expired' state.
func (s *InterviewService) persistExpireIfDue(ctx context.Context, orgID string, interviewID uuid.UUID) error {
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		iv.SetClock(s.clock)
		iv.ExpireIfNeeded()
		if iv.Status != ivdomain.StatusExpired {
			return nil
		}
		return s.ivRepo.ExpireIfDue(tctx, interviewID)
	})
}

// activeOrError refuses inactive interviews (I7) after committing any due
// expiry so storage reflects reality even when the caller is refused.
func (s *InterviewService) activeOrError(ctx context.Context, orgID string, interviewID uuid.UUID) error {
	if err := s.persistExpireIfDue(ctx, orgID, interviewID); err != nil {
		return err
	}
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		if iv.Status != ivdomain.StatusInProgress {
			return ErrInterviewNotActive
		}
		return nil
	})
}

// CandidateQARemaining returns how many more candidate questions may be answered
// this interview (cap minus recorded pairs). The limit resolves INSIDE the
// tenant tx (I4) — the org reader needs it, and resolving outside silently
// degraded every pre-check to the default cap. Expired interviews refuse here
// so the handler never reaches the LLM (I7).
func (s *InterviewService) CandidateQARemaining(ctx context.Context, orgID string, interviewID uuid.UUID) (int, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return 0, errors.NewDomainError("INVALID_ORG_ID", "invalid organization id")
	}
	if err := s.activeOrError(ctx, orgID, interviewID); err != nil {
		return 0, err
	}
	var remaining int
	err = db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		limit := s.ResolveQALimit(tctx, orgUUID)
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		remaining = limit - iv.CandidateQACount()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return remaining, nil
}

// RecordCandidateQA persists one candidate Q&A pair (B4). Enforces the cap
// server-side — the SQL guard in AppendQAPairWithinLimit is authoritative, so
// concurrent frames cannot race past it — never touches the scored transcript
// or turn state, and refuses inactive interviews before any spend (I7).
// Text is clamped before persistence (I13).
func (s *InterviewService) RecordCandidateQA(ctx context.Context, orgID string, interviewID uuid.UUID, question, answer string) error {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return errors.NewDomainError("INVALID_ORG_ID", "invalid organization id")
	}
	if err := s.activeOrError(ctx, orgID, interviewID); err != nil {
		return err
	}
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		limit := s.ResolveQALimit(tctx, orgUUID)
		pair := ivdomain.QAPair{
			Question:  clampRunes(question, ivdomain.MaxCandidateQuestionRunes),
			Answer:    clampRunes(answer, maxStoredQAAnswerRunes),
			CreatedAt: s.clock.Now(),
		}
		appended, err := s.ivRepo.AppendQAPairWithinLimit(tctx, interviewID, pair, limit)
		if err != nil {
			if stderrors.Is(err, ivdomain.ErrQAActive) {
				return ErrInterviewNotActive
			}
			return err
		}
		if !appended {
			return ErrQALimitExceeded
		}
		return nil
	})
}

// ResolveQALimit returns the org-configured candidate Q&A cap, or DefaultQALimit
// when unset/unreadable (D3).
func (s *InterviewService) ResolveQALimit(ctx context.Context, orgID uuid.UUID) int {
	if s.orgQALimit != nil {
		if n, err := s.orgQALimit.CandidateQALimit(ctx, orgID); err == nil && n > 0 {
			return n
		}
	}
	return DefaultQALimit
}

// GetCandidateQA returns the recorded candidate Q&A pairs (recruiter-visible log).
func (s *InterviewService) GetCandidateQA(ctx context.Context, orgID string, interviewID uuid.UUID) ([]ivdomain.QAPair, error) {
	var pairs []ivdomain.QAPair
	err := db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			return err
		}
		pairs = iv.QAPairs
		return nil
	})
	return pairs, err
}

func (s *InterviewService) generateQuestions(candidate *cvdomain.Candidate, job *jobdomain.Job, count int, set *jobdomain.QuestionSet) ([]gensvc.Question, error) {
	skills, summary, err := candidateProfile(candidate)
	if err != nil {
		return nil, err
	}
	profile := gensvc.CandidateProfile{Skills: skills, Summary: summary}
	reqs := gensvc.JobRequirements{Title: job.Title, Description: job.Description, RequiredSkills: job.RequiredSkills}
	return gensvc.GenerateQuestions(profile, reqs, count, set), nil
}

func candidateProfile(c *cvdomain.Candidate) ([]string, string, error) {
	if len(c.CVStructured) == 0 {
		return nil, c.CVRawText, nil
	}
	var rd struct {
		Skills  []string `json:"skills"`
		Summary string   `json:"summary"`
	}
	if err := json.Unmarshal(c.CVStructured, &rd); err != nil {
		return nil, "", fmt.Errorf("decode candidate profile: %w", err)
	}
	return rd.Skills, rd.Summary, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
