package errors

// Internal wraps unexpected internal failures (driver faults, canceled
// contexts, pool exhaustion) so they never masquerade as client-facing
// DomainErrors. Error() is a static generic message safe to serialize;
// the original cause stays reachable via Unwrap for structured logging.
type Internal struct {
	Err error
}

func (e *Internal) Error() string { return "internal error" }

func (e *Internal) Unwrap() error { return e.Err }

func NewInternal(err error) *Internal { return &Internal{Err: err} }
