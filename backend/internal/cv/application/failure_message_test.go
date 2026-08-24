package application

import (
	"errors"
	"fmt"
	"testing"

	"github.com/intivai/backend/internal/llm"

	cvdomain "github.com/intivai/backend/internal/cv/domain"
)

// SafeFailureMessage maps internal failure causes to candidate-safe category
// messages. error_message is exposed via GET /cvs, so it must never carry
// provider/network/internal text (finding D26) — details stay in logs.
func TestSafeFailureMessage(t *testing.T) {
	transientCause := fmt.Errorf("%w: dial tcp 10.0.0.1:443 i/o timeout", ErrExtractTransient)
	unreadableCause := fmt.Errorf("%w: no extractable text", cvdomain.ErrUnreadable)
	parseCause := fmt.Errorf("extract llm returned malformed output: %w", llm.ErrStructuredParse)

	tests := []struct {
		name  string
		cause error
		want  string
	}{
		{"nil stays empty", nil, ""},
		{
			"transient provider/network/quota",
			transientCause,
			"temporary_error — retry with POST /cvs/{id}/extract",
		},
		{
			"bare transient sentinel",
			ErrExtractTransient,
			"temporary_error — retry with POST /cvs/{id}/extract",
		},
		{
			"unreadable document / OCR exhaustion",
			unreadableCause,
			"cv_unreadable — file could not be parsed as PDF or DOCX",
		},
		{
			"malformed model output",
			parseCause,
			"extraction_failed — the model returned unusable output",
		},
		{"unknown internal fault stays generic", errors.New("pq: lo_apply_overflow"), "extraction_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeFailureMessage(tt.cause)
			if got != tt.want {
				t.Fatalf("SafeFailureMessage = %q, want %q", got, tt.want)
			}
		})
	}

	for _, cause := range []error{transientCause, unreadableCause, parseCause, errors.New("boom")} {
		if msg := SafeFailureMessage(cause); msg == "" || containsInternalDetail(msg, cause) {
			t.Fatalf("message %q leaks detail of %v", msg, cause)
		}
	}
}

// containsInternalDetail fails when the raw cause text bleeds into the
// candidate-visible message.
func containsInternalDetail(msg string, cause error) bool {
	if cause == nil {
		return false
	}
	return msg == cause.Error()
}
