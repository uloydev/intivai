package domain

import "github.com/intivai/backend/internal/shared/errors"

var (
	ErrNotFound       = errors.NewNotFoundError("webhook_config", "")
	ErrInvalidURL     = errors.NewDomainError("INVALID_URL", "webhook URL must be public http/https")
	ErrNoEvents       = errors.NewDomainError("NO_EVENTS", "at least one event must be specified")
	ErrInvalidEvent   = errors.NewDomainError("INVALID_EVENT", "unknown webhook event type")
	ErrDeliveryFailed = errors.NewDomainError("DELIVERY_FAILED", "webhook delivery failed after retries")
)
