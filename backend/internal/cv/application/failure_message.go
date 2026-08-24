package application

import (
	"errors"

	cvdomain "github.com/intivai/backend/internal/cv/domain"
	"github.com/intivai/backend/internal/llm"
)

// Safe categories persisted into candidate-visible error_message. Raw causes
// (provider endpoints, timeouts, parser internals) stay in structured logs
// only — error_message is exposed via GET /cvs (finding D26).
const (
	msgTemporaryError = "temporary_error — retry with POST /cvs/{id}/extract"
	msgUnreadable     = "cv_unreadable — file could not be parsed as PDF or DOCX"
	msgBadModelOutput = "extraction_failed — the model returned unusable output"
	msgExtraction     = "extraction_failed"
)

// SafeFailureMessage maps an internal failure cause to a candidate-safe
// category message.
func SafeFailureMessage(cause error) string {
	switch {
	case cause == nil:
		return ""
	case errors.Is(cause, ErrExtractTransient):
		return msgTemporaryError
	case errors.Is(cause, cvdomain.ErrUnreadable):
		return msgUnreadable
	case errors.Is(cause, llm.ErrStructuredParse):
		return msgBadModelOutput
	default:
		return msgExtraction
	}
}
