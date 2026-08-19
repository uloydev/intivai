// Package uuidx — tiny UUID helpers shared across contexts (single source
// for the former duplicated mustUUID definitions).
package uuidx

import "github.com/google/uuid"

// MustParse parses s or returns uuid.Nil. Only for ids that were validated
// at an API boundary; never use it to swallow real parse errors.
func MustParse(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
