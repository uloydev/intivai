package domain

import "errors"

var (
	ErrNotFound = errors.New("interview not found")
	ErrNoClock  = errors.New("interview has no clock")
)
