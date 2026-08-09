package domain

import "errors"

var (
	ErrNotFound = errors.New("application not found")
	ErrExists   = errors.New("application already exists")
)

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func IsExists(err error) bool   { return errors.Is(err, ErrExists) }
