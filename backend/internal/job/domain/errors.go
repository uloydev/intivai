package domain

import "errors"

var (
	ErrNotFound      = errors.New("job not found")
	ErrDuplicateSlug = errors.New("job slug already taken")
)
