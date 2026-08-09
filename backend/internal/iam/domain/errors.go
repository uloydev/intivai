package domain

import "errors"

// Sentinel errors for repo/use-case boundaries. Persistence adapters wrap
// pgx.ErrNoRows and pg 23505 into these.
var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateSlug  = errors.New("org slug already taken")
	ErrDuplicateEmail = errors.New("email already exists in org")
)
