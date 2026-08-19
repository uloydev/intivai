package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrdomain "github.com/intivai/backend/internal/screening/domain"
	"github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/pkg/db"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const TaskParseCV = "parse_cv"

// ObjectStore — MinIO storage as an interface (stubbable in tests).
type ObjectStore interface {
	Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}

// Enqueuer — asynq client as an interface (stubbable in tests).
type Enqueuer interface {
	Enqueue(ctx context.Context, task string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// CVService handles upload: persist candidate row, store file in MinIO,
// enqueue parse_cv.
type CVService struct {
	repo    domain.CandidateRepository
	appRepo scrdomain.ApplicationRepository
	store   ObjectStore
	queue   Enqueuer
	pool    *gorm.DB
}

func NewCVService(repo domain.CandidateRepository, appRepo scrdomain.ApplicationRepository, store ObjectStore, queueClient Enqueuer, pool *gorm.DB) *CVService {
	return &CVService{repo: repo, appRepo: appRepo, store: store, queue: queueClient, pool: pool}
}

func (s *CVService) Upload(ctx context.Context, actor application.AuthContext, name, email string, file []byte, contentType string) (*CVResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	candidate, err := domain.NewCandidate(actor.OrgID, strings.TrimSpace(name), strings.TrimSpace(email))
	if err != nil {
		return nil, err
	}
	candidate.CVPath = fmt.Sprintf("cvs/%s/%s.pdf", actor.OrgID, candidate.ID)
	candidate.Status = domain.StatusParsing

	if err := s.store.Upload(ctx, candidate.CVPath, strings.NewReader(string(file)), int64(len(file)), contentType); err != nil {
		return nil, errors.NewDomainError("CV_STORAGE_FAILED", "failed to store cv file")
	}
	if err := s.repo.Create(ctx, candidate); err != nil {
		_ = s.store.Delete(ctx, candidate.CVPath) // compensate orphan object
		return nil, err
	}
	if _, err := s.queue.Enqueue(ctx, TaskParseCV, ParseCVPayload{
		OrgID: actor.OrgID.String(), CandidateID: candidate.ID.String(),
	}, asynq.MaxRetry(5)); err != nil {
		// Full compensation: no row, no file, no dangling task.
		_ = s.store.Delete(ctx, candidate.CVPath)
		_ = s.repo.Delete(ctx, candidate.ID)
		return nil, errors.NewDomainError("CV_QUEUE_FAILED", "failed to enqueue cv parsing")
	}
	return &CVResult{ID: candidate.ID, Status: candidate.Status}, nil
}

// ReExtract re-triggers extract_cv for a candidate that failed extraction
// (or was parsed but never extracted). Guards against re-running on
// in-flight or already-extracted candidates.
func (s *CVService) ReExtract(ctx context.Context, actor application.AuthContext, id uuid.UUID) (*CVResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	candidate, err := s.repo.GetByID(ctx, id)
	if err == domain.ErrNotFound {
		return nil, errors.NewNotFoundError("candidate", id.String())
	}
	if err != nil {
		return nil, err
	}
	if candidate.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "candidate belongs to another org")
	}
	switch candidate.Status {
	case domain.StatusFailedOCR:
		if _, err := s.queue.Enqueue(ctx, TaskParseCV, ParseCVPayload{
			OrgID: actor.OrgID.String(), CandidateID: candidate.ID.String(),
		}, asynq.MaxRetry(5)); err != nil {
			return nil, errors.NewDomainError("CV_QUEUE_FAILED", "failed to enqueue cv parsing")
		}
		return &CVResult{ID: candidate.ID, Status: domain.StatusParsing}, nil
	case domain.StatusFailedExtract, domain.StatusParsed:
		if _, err := s.queue.Enqueue(ctx, TaskExtractCV, ExtractCVPayload{
			OrgID: actor.OrgID.String(), CandidateID: candidate.ID.String(),
		}, asynq.MaxRetry(5)); err != nil {
			return nil, errors.NewDomainError("CV_QUEUE_FAILED", "failed to enqueue extraction")
		}
		return &CVResult{ID: candidate.ID, Status: domain.StatusExtracting}, nil
	default:
		return nil, errors.NewDomainError("CANDIDATE_NOT_RETRYABLE", "candidate is not in a retryable state")
	}
}

type CVResult struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type BulkUploadResult struct {
	BatchID uuid.UUID `json:"batch_id"`
}

type BulkUploadFile struct {
	Name        string
	Data        []byte
	ContentType string
}

func (s *CVService) BulkUpload(ctx context.Context, actor application.AuthContext, files []BulkUploadFile) (*BulkUploadResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	batchID := uuid.New()

	for _, f := range files {
		candidate, err := domain.NewCandidate(actor.OrgID, strings.TrimSpace(f.Name), "")
		if err != nil {
			continue // skip invalid names
		}
		candidate.BatchID = &batchID
		candidate.CVPath = fmt.Sprintf("cvs/%s/%s.pdf", actor.OrgID, candidate.ID)
		candidate.Status = domain.StatusParsing

		if err := s.store.Upload(ctx, candidate.CVPath, strings.NewReader(string(f.Data)), int64(len(f.Data)), f.ContentType); err != nil {
			continue // skip if upload fails
		}
		if err := s.repo.Create(ctx, candidate); err != nil {
			_ = s.store.Delete(ctx, candidate.CVPath)
			continue
		}
		if _, err := s.queue.Enqueue(ctx, TaskParseCV, ParseCVPayload{
			OrgID: actor.OrgID.String(), CandidateID: candidate.ID.String(),
		}); err != nil {
			_ = s.store.Delete(ctx, candidate.CVPath)
			_ = s.repo.Delete(ctx, candidate.ID)
		}
	}
	return &BulkUploadResult{BatchID: batchID}, nil
}

// CVListItem — summary only. Raw text and structured data (PII) stay on
// GET /cvs/:id.
type CVListItem struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	CVOCRMethod  string    `json:"cv_ocr_method,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    string    `json:"created_at"`
}

type CVDetail struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Email        string          `json:"email"`
	Status       string          `json:"status"`
	CVPath       string          `json:"cv_path"`
	CVRawText    string          `json:"cv_raw_text,omitempty"`
	CVStructured json.RawMessage `json:"cv_structured,omitempty"`
	CVOCRMethod  string          `json:"cv_ocr_method,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    string          `json:"created_at"`
}

func (s *CVService) Get(ctx context.Context, actor application.AuthContext, id uuid.UUID) (*CVDetail, error) {
	candidate, err := s.repo.GetByID(ctx, id)
	if err == domain.ErrNotFound {
		return nil, errors.NewNotFoundError("candidate", id.String())
	}
	if err != nil {
		return nil, err
	}
	if candidate.OrgID != actor.OrgID {
		return nil, errors.NewDomainError("FORBIDDEN", "candidate belongs to another org")
	}
	return toDetail(candidate), nil
}

func (s *CVService) List(ctx context.Context, actor application.AuthContext) ([]*CVListItem, error) {
	list, err := s.repo.List(ctx, actor.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]*CVListItem, 0, len(list))
	for _, c := range list {
		out = append(out, &CVListItem{
			ID: c.ID, Name: c.Name, Email: c.Email, Status: c.Status,
			CVOCRMethod: c.CVOCRMethod, ErrorMessage: c.ErrorMessage, CreatedAt: c.CreatedAt.String(),
		})
	}
	return out, nil
}

func toDetail(c *domain.Candidate) *CVDetail {
	return &CVDetail{
		ID: c.ID, Name: c.Name, Email: c.Email, Status: c.Status, CVPath: c.CVPath,
		CVRawText: c.CVRawText, CVStructured: c.CVStructured, CVOCRMethod: c.CVOCRMethod,
		ErrorMessage: c.ErrorMessage, CreatedAt: c.CreatedAt.String(),
	}
}

func (s *CVService) DeleteCandidate(ctx context.Context, actor application.AuthContext, id uuid.UUID) error {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return err
	}
	candidate, err := s.repo.GetByID(ctx, id)
	if err == domain.ErrNotFound {
		return errors.NewNotFoundError("candidate", id.String())
	}
	if err != nil {
		return err
	}
	if candidate.OrgID != actor.OrgID {
		return errors.NewDomainError("FORBIDDEN", "candidate belongs to another org")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if candidate.CVPath != "" {
		_ = s.store.Delete(ctx, candidate.CVPath)
	}
	return nil
}

func (s *CVService) ReviewProfile(ctx context.Context, token string) (*CVDetail, error) {
	candidate, err := s.repo.GetByReviewToken(ctx, token)
	if err == domain.ErrNotFound {
		return nil, errors.NewNotFoundError("candidate_review", "invalid token")
	}
	if err != nil {
		return nil, err
	}
	if candidate.Status != domain.StatusPendingReview {
		return nil, errors.NewDomainError("CANDIDATE_NOT_READY", "candidate is not pending review")
	}
	return toDetail(candidate), nil
}

func (s *CVService) ConfirmProfile(ctx context.Context, token string, structuredData []byte) error {
	orgID, candID, err := s.repo.ConfirmReview(ctx, token, structuredData)
	if err != nil {
		return err
	}
	if orgID == uuid.Nil {
		return errors.NewNotFoundError("candidate_review", "invalid token")
	}

	// Enqueue TaskScoreCV per application — the confirm endpoint is public
	// (no tenant middleware), so the enumeration runs inside a tenant tx
	// using the org returned by the SECURITY DEFINER function.
	var appIDs []string
	err = db.RunInTx(ctx, s.pool, orgID.String(), func(tctx context.Context) error {
		apps, err := s.appRepo.ByCandidate(tctx, orgID, candID)
		if err != nil {
			return err
		}
		for _, app := range apps {
			appIDs = append(appIDs, app.ID.String())
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, appID := range appIDs {
		if _, err := s.queue.Enqueue(ctx, scrapp.TaskScoreCV, scrapp.ScoreCVPayload{
			OrgID:         orgID.String(),
			ApplicationID: appID,
		}, asynq.MaxRetry(5)); err != nil {
			log.Warn().Err(err).Str("application_id", appID).Msg("enqueue score_cv failed")
		}
	}
	return nil
}
