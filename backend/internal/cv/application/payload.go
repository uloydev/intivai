package application

import "github.com/google/uuid"

// payloadUUID strictly parses a worker payload id — a malformed id is a
// permanent payload error (SkipRetry), never a silent zero value that would
// mask itself as "not found".
func payloadUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, errInvalidPayloadID
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, errInvalidPayloadID
	}
	return id, nil
}

var errInvalidPayloadID = &invalidPayloadIDError{}

type invalidPayloadIDError struct{}

func (*invalidPayloadIDError) Error() string { return "invalid payload id" }
