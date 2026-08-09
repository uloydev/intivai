package domain

import (
	"crypto/rand"
	"encoding/base64"
)

// APIKey is a value object: high-entropy random key.
type APIKey struct {
	raw string
}

func NewAPIKey() (APIKey, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return APIKey{}, err
	}
	return APIKey{raw: base64.RawURLEncoding.EncodeToString(b)}, nil
}

func (k APIKey) String() string { return k.raw }
