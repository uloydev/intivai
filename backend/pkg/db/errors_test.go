package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	sharederrors "github.com/intivai/backend/internal/shared/errors"
)

func pgErr(code, constraint, message string) *pgconn.PgError {
	return &pgconn.PgError{Code: code, ConstraintName: constraint, Message: message}
}

// WrapError must translate only well-known constraint violations into domain
// sentinels; every unexpected failure (canceled context, pool exhaustion,
// driver faults) is INTERNAL detail that must never masquerade as a 400-able
// DomainError nor leak its text to clients (finding D21).
func TestWrapErrorClassification(t *testing.T) {
	t.Run("nil passthrough", func(t *testing.T) {
		if err := WrapError(nil); err != nil {
			t.Fatalf("WrapError(nil) = %v", err)
		}
	})

	t.Run("unique violation maps to ALREADY_EXISTS", func(t *testing.T) {
		err := WrapError(pgErr("23505", "jobs_slug_key", "duplicate key"))
		var dom *sharederrors.DomainError
		if !errors.As(err, &dom) || dom.Code != "ALREADY_EXISTS" {
			t.Fatalf("want DomainError(ALREADY_EXISTS), got %v", err)
		}
	})

	t.Run("foreign key violation maps to NOT_FOUND", func(t *testing.T) {
		err := WrapError(pgErr("23503", "applications_job_id_fkey", "violates foreign key"))
		var dom *sharederrors.DomainError
		if !errors.As(err, &dom) || dom.Code != "NOT_FOUND" {
			t.Fatalf("want DomainError(NOT_FOUND), got %v", err)
		}
	})

	t.Run("context canceled becomes generic Internal, not DomainError", func(t *testing.T) {
		cause := fmt.Errorf("query: %w", context.Canceled)
		err := WrapError(cause)

		var dom *sharederrors.DomainError
		if errors.As(err, &dom) {
			t.Fatalf("canceled context wrapped as DomainError %+v — renders as 400 + leaks text", dom)
		}
		var internal *sharederrors.Internal
		if !errors.As(err, &internal) {
			t.Fatalf("want *sharederrors.Internal, got %T (%v)", err, err)
		}
		if err.Error() != "internal error" {
			t.Fatalf("message %q leaks internal detail", err.Error())
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatal("original cause unreachable through Unwrap — logging loses detail")
		}
	})

	t.Run("unexpected driver error becomes generic Internal", func(t *testing.T) {
		err := WrapError(errors.New("conn busy"))
		var internal *sharederrors.Internal
		if !errors.As(err, &internal) {
			t.Fatalf("want *sharederrors.Internal, got %v", err)
		}
		if err.Error() != "internal error" {
			t.Fatalf("message %q leaks internal detail", err.Error())
		}
	})

	t.Run("other pg errors become generic Internal without leaking message", func(t *testing.T) {
		err := WrapError(pgErr("42P01", "", `relation "ghost" does not exist`))
		var dom *sharederrors.DomainError
		if errors.As(err, &dom) {
			t.Fatalf("unmapped pg error wrapped as DomainError %+v — leaks pg text via 400", dom)
		}
		if err.Error() != "internal error" {
			t.Fatalf("message %q leaks internal detail", err.Error())
		}
	})

	t.Run("existing DomainError passes through untouched", func(t *testing.T) {
		sentinel := sharederrors.NewDomainError("CV_STORAGE_FAILED", "failed to store cv file")
		if err := WrapError(sentinel); err != sentinel {
			t.Fatalf("sentinel rewritten: %v", err)
		}
	})
}
