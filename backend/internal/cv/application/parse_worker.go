package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/cv/infrastructure/ocr"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/ledongthuc/pdf"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// ParseWorker: download CV from MinIO, extract text (ledongthuc/pdf),
// OCR fallback for scanned PDFs, persist raw text, enqueue extraction.
type ParseWorker struct {
	pool  *gorm.DB
	repo  cvdomain.CandidateRepository
	store *storage.Storage
	queue *queue.Client
	log   zerolog.Logger
}

func NewParseWorker(pool *gorm.DB, repo cvdomain.CandidateRepository, store *storage.Storage, q *queue.Client, log zerolog.Logger) *ParseWorker {
	return &ParseWorker{pool: pool, repo: repo, store: store, queue: q, log: log}
}

func (w *ParseWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskParseCV, w.handle)
}

func (w *ParseWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p ParseCVPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}

	candidate, err := w.fetch(ctx, p)
	if err != nil {
		return err
	}

	data, err := w.download(ctx, candidate.CVPath)
	if err != nil {
		return w.mark(ctx, p, cvdomain.StatusFailedOCR, err)
	}

	text, err := extractPDFText(data)
	if err != nil {
		return w.mark(ctx, p, cvdomain.StatusFailedOCR, err)
	}
	method := "pdfcpu"
	if len(strings.TrimSpace(text)) < 50 {
		ocrText, oerr := ocr.Extract(data)
		if oerr != nil {
			return w.mark(ctx, p, cvdomain.StatusFailedOCR, oerr)
		}
		text = ocrText
		method = "tesseract"
	}

	err = db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		c, err := w.repo.GetByID(tctx, candidate.ID)
		if err != nil {
			return err
		}
		c.CVRawText = text
		c.CVOCRMethod = method
		c.Status = cvdomain.StatusParsed
		return w.repo.Update(tctx, c)
	})
	if err != nil {
		return err
	}
	w.queueExtract(ctx, p)
	return nil
}
func (w *ParseWorker) queueExtract(ctx context.Context, p ParseCVPayload) {
	if _, err := w.queue.Enqueue(ctx, TaskExtractCV, p); err != nil {
		w.log.Error().Err(err).Str("candidate_id", p.CandidateID).Msg("enqueue extract_cv failed")
	}
}

// fetch loads the candidate and marks it parsing, in its own transaction.
func (w *ParseWorker) fetch(ctx context.Context, p ParseCVPayload) (*cvdomain.Candidate, error) {
	var candidate *cvdomain.Candidate
	err := db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		var err error
		candidate, err = w.repo.GetByID(tctx, mustUUID(p.CandidateID))
		if errors.Is(err, cvdomain.ErrNotFound) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}
		candidate.Status = cvdomain.StatusParsing
		return w.repo.Update(tctx, candidate)
	})
	return candidate, err
}

// mark sets a terminal failure status in its own transaction — never inside
// an aborted one.
func (w *ParseWorker) mark(ctx context.Context, p ParseCVPayload, status string, cause error) error {
	if cause != nil {
		w.log.Error().Err(cause).Str("candidate_id", p.CandidateID).Msg("parse_cv failed")
	}
	return db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		c, err := w.repo.GetByID(tctx, mustUUID(p.CandidateID))
		if errors.Is(err, cvdomain.ErrNotFound) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}
		c.Status = status
		if cause != nil {
			c.ErrorMessage = cause.Error()
		}
		return w.repo.Update(tctx, c)
	})
}

func (w *ParseWorker) download(ctx context.Context, path string) ([]byte, error) {
	reader, err := w.store.Download(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractPDFText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		text, err := r.Page(i).GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return "", errors.New("no extractable text")
	}
	return sb.String(), nil
}
