package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	sharederrors "github.com/intivai/backend/internal/shared/errors"
)

// WrapError translates common PostgreSQL driver errors (like unique constraint violations)
// into generic DomainErrors, preventing raw DB errors from leaking up the stack.
// Repositories should wrap Exec/Scan errors with this function unless they need
// to map specific constraints to specific domain sentinels.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return sharederrors.NewDomainError("ALREADY_EXISTS", "resource already exists: "+pgErr.ConstraintName)
		}
		if pgErr.Code == "23503" { // foreign_key_violation
			return sharederrors.NewDomainError("NOT_FOUND", "referenced resource not found: "+pgErr.ConstraintName)
		}
		// Generic fallback for other PG errors
		return sharederrors.NewDomainError("DATABASE_ERROR", "database error: "+pgErr.Message)
	}
	// Do not wrap context errors or already-domain errors
	var domErr *sharederrors.DomainError
	if errors.As(err, &domErr) {
		return err
	}
	if errors.Is(err, sharederrors.NewDomainError("", "")) {
		return err // Just a check conceptually, errors.As covers it
	}
	return sharederrors.NewDomainError("INTERNAL_ERROR", err.Error())
}
