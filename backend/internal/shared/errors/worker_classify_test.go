package errors

import (
	"errors"
	"testing"
)

var (
	errSentinelNotFound  = NewNotFoundError("webhook_delivery", "")
	errSentinelForbidden = NewDomainError("FORBIDDEN", "wrong org")
	errTransientPool     = errors.New("pgx: pool exhausted")
	errTransientNet      = errors.New("connection reset by peer")
)

// Queue workers must retry every non-sentinel failure (finding D20): pool
// exhaustion, driver faults and network blips are recoverable, while
// not-found / wrong-org lookups never succeed on retry.
func TestPermanentWorkerError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		sentinels []error
		want      bool
	}{
		{"nil is transient-by-convention", nil, []error{errSentinelNotFound}, false},
		{"direct sentinel match", errSentinelNotFound, []error{errSentinelNotFound}, true},
		{"wrapped sentinel match", errWrap("load delivery", errSentinelNotFound), []error{errSentinelNotFound}, true},
		{"second sentinel matches", errSentinelForbidden, []error{errSentinelNotFound, errSentinelForbidden}, true},
		{"transient pool failure", errTransientPool, []error{errSentinelNotFound}, false},
		{"transient network failure", errTransientNet, []error{errSentinelNotFound, errSentinelForbidden}, false},
		{"unclassified generic error", errors.New("something odd"), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PermanentWorkerError(tt.err, tt.sentinels...); got != tt.want {
				t.Fatalf("PermanentWorkerError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type wrappedErr struct {
	op  string
	err error
}

func (w *wrappedErr) Error() string { return w.op + ": " + w.err.Error() }
func (w *wrappedErr) Unwrap() error { return w.err }

func errWrap(op string, err error) error { return &wrappedErr{op: op, err: err} }
