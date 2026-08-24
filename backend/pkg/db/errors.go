package db

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	sharederrors "github.com/intivai/backend/internal/shared/errors"
)

// WrapError translates well-known PostgreSQL constraint violations into
// domain sentinels; every other failure is wrapped as a generic Internal
// error so driver/context text never reaches clients (finding D21).
// Repositories should wrap Exec/Scan errors with this function unless they
// need to map specific constraints to specific domain sentinels.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	var domErr *sharederrors.DomainError
	if errors.As(err, &domErr) {
		return err // already classified — do not re-wrap
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return sharederrors.NewDomainError("ALREADY_EXISTS", "resource already exists: "+pgErr.ConstraintName)
		case "23503": // foreign_key_violation
			return sharederrors.NewDomainError("NOT_FOUND", "referenced resource not found: "+pgErr.ConstraintName)
		}
		return sharederrors.NewInternal(fmt.Errorf("postgres %s: %s", pgErr.Code, pgErr.Message))
	}
	return sharederrors.NewInternal(err)
}
