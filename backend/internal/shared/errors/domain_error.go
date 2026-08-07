package errors

import "fmt"

// DomainError is the base for all domain errors.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }

func NewDomainError(code, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

func (e *DomainError) WithContext(format string, args ...any) *DomainError {
	e.Message = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), e.Message)
	return e
}
