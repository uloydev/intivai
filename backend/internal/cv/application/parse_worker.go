package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/cv/infrastructure/ocr"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/ledongthuc/pdf"
	docxlib "github.com/nguyenthenguyen/docx"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// ParseWorker: download CV from MinIO, extract text (ledongthuc/pdf),
// OCR fallback for scanned PDFs, persist raw text, enqueue extraction.
type ParseWorker struct {
	pool  *gorm.DB
	repo  cvdomain.CandidateRepository
	store storage.FileStorage
	queue parseEnqueuer
	log   zerolog.Logger
}

type parseEnqueuer interface {
	Enqueue(ctx context.Context, task string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

func NewParseWorker(pool *gorm.DB, repo cvdomain.CandidateRepository, store *storage.Storage, queueClient *queue.Client, log zerolog.Logger) *ParseWorker {
	return &ParseWorker{pool: pool, repo: repo, store: store, queue: queueClient, log: log}
}

func (w *ParseWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskParseCV, w.handle)
}

func (w *ParseWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p ParseCVPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	if _, err := payloadUUID(p.CandidateID); err != nil {
		return asynq.SkipRetry
	}

	candidate, err := w.fetch(ctx, p)
	if err != nil {
		return err
	}
	if candidate.Status == cvdomain.StatusParsed && strings.TrimSpace(candidate.CVRawText) != "" {
		return w.queueExtract(ctx, p)
	}

	data, err := w.download(ctx, candidate.CVPath)
	if err != nil {
		// Transient object-store/network fault — the task must stay
		// retryable; mislabeling it terminal failed_ocr burns candidates on
		// MinIO blips (finding D26 / plan 5.7).
		w.log.Warn().Err(err).Str("candidate_id", p.CandidateID).Msg("cv download failed — task will retry")
		return fmt.Errorf("download cv %s: %w", p.CandidateID, err)
	}

	// Detect file type and extract text
	var text, method, format string
	if isDOCX(data) {
		text, err = extractDOCXText(data)
		method = "docx"
		format = "docx"
	} else {
		text, err = extractPDFText(data)
		method = "pdfcpu"
		format = "pdf"
	}
	if err != nil {
		if markErr := w.mark(ctx, p, cvdomain.StatusFailedOCR, fmt.Errorf("%w: %v", cvdomain.ErrUnreadable, err)); markErr != nil {
			return fmt.Errorf("mark failed OCR after text extraction failure: %w", markErr)
		}
		return asynq.SkipRetry
	}
	if method == "docx" && len(strings.TrimSpace(text)) < 50 {
		// Scanned/photo DOCX carry no embedded text — OCR needs a renderer
		// (libreoffice/poppler) that the worker image does not ship. Mark the
		// outcome honestly and surface it instead of silently parsing empty text.
		w.log.Warn().Str("candidate_id", p.CandidateID).Msg("docx yielded <50 chars of text — likely scanned; re-upload as PDF to OCR")
	}
	if method == "pdfcpu" && len(strings.TrimSpace(text)) < 50 {
		ocrText, oerr := ocr.ExtractContext(ctx, data)
		if oerr != nil {
			// True OCR exhaustion — terminal, with a safe category message.
			if markErr := w.mark(ctx, p, cvdomain.StatusFailedOCR, fmt.Errorf("%w: %v", cvdomain.ErrUnreadable, oerr)); markErr != nil {
				return fmt.Errorf("mark failed OCR after OCR failure: %w", markErr)
			}
			return asynq.SkipRetry
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
		c.CVFormat = format
		c.Status = cvdomain.StatusParsed
		return w.repo.Update(tctx, c)
	})
	if err != nil {
		return err
	}
	return w.queueExtract(ctx, p)
}
func (w *ParseWorker) queueExtract(ctx context.Context, p ParseCVPayload) error {
	if _, err := w.queue.Enqueue(ctx, TaskExtractCV, p, asynq.MaxRetry(5)); err != nil {
		w.log.Error().Err(err).Str("candidate_id", p.CandidateID).Msg("enqueue extract_cv failed")
		return fmt.Errorf("enqueue extract_cv: %w", err)
	}
	return nil
}

// fetch loads the candidate and marks it parsing, in its own transaction.
func (w *ParseWorker) fetch(ctx context.Context, p ParseCVPayload) (*cvdomain.Candidate, error) {
	var candidate *cvdomain.Candidate
	err := db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		var err error
		candidate, err = w.repo.GetByID(tctx, uuid.MustParse(p.CandidateID))
		if errors.Is(err, cvdomain.ErrNotFound) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}
		if candidate.Status == cvdomain.StatusParsed && strings.TrimSpace(candidate.CVRawText) != "" {
			return nil
		}
		candidate.Status = cvdomain.StatusParsing
		return w.repo.Update(tctx, candidate)
	})
	return candidate, err
}

// mark sets a terminal failure status in its own transaction — never inside
// an aborted one. error_message gets a candidate-safe category; the raw
// cause stays in logs (finding D26).
func (w *ParseWorker) mark(ctx context.Context, p ParseCVPayload, status string, cause error) error {
	if cause != nil {
		w.log.Error().Err(cause).Str("candidate_id", p.CandidateID).Msg("parse_cv failed")
	}
	safeMessage := SafeFailureMessage(cause)
	return db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		c, err := w.repo.GetByID(tctx, uuid.MustParse(p.CandidateID))
		if errors.Is(err, cvdomain.ErrNotFound) {
			return asynq.SkipRetry
		}
		if err != nil {
			return err
		}
		c.Status = status
		c.ErrorMessage = safeMessage
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

// isDOCX detects DOCX files by checking for ZIP header + word/document.xml
func isDOCX(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// ZIP signature: PK\x03\x04
	if data[0] != 0x50 || data[1] != 0x4B || data[2] != 0x03 || data[3] != 0x04 {
		return false
	}
	// Check for word/document.xml in ZIP contents
	return bytes.Contains(data, []byte("word/document.xml"))
}

// extractDOCXText extracts plain text from a DOCX file
func extractDOCXText(data []byte) (string, error) {
	r, err := docxlib.ReadDocxFromMemory(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	doc := r.Editable()
	text := doc.GetContent()
	if len(strings.TrimSpace(text)) == 0 {
		return "", errors.New("no extractable text in DOCX")
	}
	return text, nil
}
