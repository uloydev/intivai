package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	jobdomain "github.com/intivai/backend/internal/job/domain"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// EvaluationService — recruiter-facing interview + candidate report queries.
type EvaluationService struct {
	pool     *gorm.DB
	ivRepo   ivdomain.InterviewRepository
	appRepo  scrdomain.ApplicationRepository
	candRepo cvdomain.CandidateRepository
	jobRepo  jobdomain.JobRepository
}

func NewEvaluationService(pool *gorm.DB, ivRepo ivdomain.InterviewRepository, appRepo scrdomain.ApplicationRepository,
	candRepo cvdomain.CandidateRepository, jobRepo jobdomain.JobRepository) *EvaluationService {
	return &EvaluationService{pool: pool, ivRepo: ivRepo, appRepo: appRepo, candRepo: candRepo, jobRepo: jobRepo}
}

// --- DTOs ---

type AnswerDTO struct {
	Idx        int       `json:"idx"`
	Content    string    `json:"content"`
	AnsweredAt time.Time `json:"answered_at"`
}

type QuestionDTO struct {
	Idx      int    `json:"idx"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Skill    string `json:"skill,omitempty"`
}

type CandidateDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type JobDTO struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type InterviewDetail struct {
	InterviewID    uuid.UUID       `json:"interview_id"`
	Status         ivdomain.Status `json:"status"`
	ContextVersion int             `json:"context_version"`
	TotalQuestions int             `json:"total_questions"`
	Questions      []QuestionDTO   `json:"questions"`
	Answers        []AnswerDTO     `json:"answers"`
	Evaluation     json.RawMessage `json:"evaluation"`
	Candidate      *CandidateDTO   `json:"candidate"`
	Job            *JobDTO         `json:"job"`
	CreatedAt      time.Time       `json:"created_at"`
	CompletedAt    *time.Time      `json:"completed_at"`
}

type InterviewSummary struct {
	InterviewID uuid.UUID       `json:"interview_id"`
	Status      ivdomain.Status `json:"status"`
	Evaluation  json.RawMessage `json:"evaluation"`
	CompletedAt *time.Time      `json:"completed_at"`
}

type CandidateReport struct {
	Candidate  CandidateDTO       `json:"candidate"`
	Interviews []InterviewSummary `json:"interviews"`
}

// InterviewListItem — recruiter list row (no full transcript, keeps the list light).
type InterviewListItem struct {
	InterviewID    uuid.UUID       `json:"interview_id"`
	Status         ivdomain.Status `json:"status"`
	CandidateID    uuid.UUID       `json:"candidate_id"`
	CandidateName  string          `json:"candidate_name"`
	JobTitle       string          `json:"job_title"`
	Evaluation     json.RawMessage `json:"evaluation"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ListInterviews returns the org's interviews, newest first, with candidate
// + job context for the recruiter list view.
func (s *EvaluationService) ListInterviews(ctx context.Context, actor application.AuthContext) ([]*InterviewListItem, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter, iamdomain.RoleInterviewer); err != nil {
		return nil, err
	}
	var out []*InterviewListItem
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		ivs, err := s.ivRepo.ListByOrg(tctx, actor.OrgID)
		if err != nil {
			return err
		}
		out = make([]*InterviewListItem, 0, len(ivs))
		for _, iv := range ivs {
			item := &InterviewListItem{
				InterviewID: iv.ID,
				Status:      iv.Status,
				Evaluation:  json.RawMessage(iv.Evaluation),
				CreatedAt:   iv.CreatedAt,
			}
			if app, err := s.appRepo.GetByID(tctx, iv.ApplicationID); err == nil {
				item.CandidateID = app.CandidateID
				if c, err := s.candRepo.GetByID(tctx, app.CandidateID); err == nil {
					item.CandidateName = c.Name
				}
				if j, err := s.jobRepo.GetByID(tctx, app.JobID); err == nil {
					item.JobTitle = j.Title
				}
			}
			out = append(out, item)
		}
		return nil
	})
	return out, err
}

// InterviewDetail returns the full interview (questions, answers, evaluation)
// with candidate + job context. Org-scoped via RLS + explicit org checks.
func (s *EvaluationService) InterviewDetail(ctx context.Context, actor application.AuthContext, interviewID uuid.UUID) (*InterviewDetail, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter, iamdomain.RoleInterviewer); err != nil {
		return nil, err
	}
	var detail *InterviewDetail
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		iv, err := s.ivRepo.GetByID(tctx, interviewID)
		if err != nil {
			if err == ivdomain.ErrNotFound {
				return errors.NewNotFoundError("interview", interviewID.String())
			}
			return err
		}
		app, err := s.appRepo.GetByID(tctx, iv.ApplicationID)
		if err == scrdomain.ErrNotFound {
			// Application row gone (deleted) — the interview itself still
			// stands; surface it without candidate/job context.
			detail = &InterviewDetail{
				InterviewID:    iv.ID,
				Status:         iv.Status,
				ContextVersion: iv.ContextVersion,
				TotalQuestions: len(iv.Questions),
				Evaluation:     json.RawMessage(iv.Evaluation),
				CreatedAt:      iv.CreatedAt,
				CompletedAt:    iv.CompletedAt,
			}
			return nil
		}
		if err != nil {
			return err
		}
		if app.OrgID != actor.OrgID {
			return errors.NewDomainError("FORBIDDEN", "interview belongs to another org")
		}
		candidate, err := s.candRepo.GetByID(tctx, app.CandidateID)
		if err != nil {
			return err
		}
		job, err := s.jobRepo.GetByID(tctx, app.JobID)
		if err != nil {
			return err
		}

		d := &InterviewDetail{
			InterviewID:    iv.ID,
			Status:         iv.Status,
			ContextVersion: iv.ContextVersion,
			TotalQuestions: len(iv.Questions),
			Evaluation:     json.RawMessage(iv.Evaluation),
			Candidate:      &CandidateDTO{ID: candidate.ID, Name: candidate.Name, Email: candidate.Email},
			Job:            &JobDTO{ID: job.ID, Title: job.Title},
			CreatedAt:      iv.CreatedAt,
			CompletedAt:    iv.CompletedAt,
		}
		for _, q := range iv.Questions {
			d.Questions = append(d.Questions, QuestionDTO{Idx: q.Idx, Content: q.Content, Category: q.Category, Skill: q.Skill})
		}
		for _, a := range iv.Answers {
			d.Answers = append(d.Answers, AnswerDTO{Idx: a.Idx, Content: a.Content, AnsweredAt: a.AnsweredAt})
		}
		detail = d
		return nil
	})
	return detail, err
}

// CandidateReport returns the candidate + every interview with its evaluation.
func (s *EvaluationService) CandidateReport(ctx context.Context, actor application.AuthContext, candidateID uuid.UUID) (*CandidateReport, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter, iamdomain.RoleInterviewer); err != nil {
		return nil, err
	}
	var report *CandidateReport
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		candidate, err := s.candRepo.GetByID(tctx, candidateID)
		if err != nil {
			if err == cvdomain.ErrNotFound {
				return errors.NewNotFoundError("candidate", candidateID.String())
			}
			return err
		}
		if candidate.OrgID != actor.OrgID {
			return errors.NewDomainError("FORBIDDEN", "candidate belongs to another org")
		}
		apps, err := s.appRepo.ByCandidate(tctx, actor.OrgID, candidateID)
		if err != nil {
			return err
		}
		r := &CandidateReport{Candidate: CandidateDTO{ID: candidate.ID, Name: candidate.Name, Email: candidate.Email}}
		for _, app := range apps {
			ivs, err := s.ivRepo.ByApplication(tctx, app.ID)
			if err != nil {
				return err
			}
			for _, iv := range ivs {
				r.Interviews = append(r.Interviews, InterviewSummary{
					InterviewID: iv.ID,
					Status:      iv.Status,
					Evaluation:  json.RawMessage(iv.Evaluation),
					CompletedAt: iv.CompletedAt,
				})
			}
		}
		report = r
		return nil
	})
	return report, err
}
