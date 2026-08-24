package errors

import "errors"

// PermanentWorkerError reports whether err matches one of the permanent
// sentinel errors (not-found, wrong-org, invalid payload) — failures that
// can never succeed on retry. Everything else (pool exhaustion, driver
// faults, network blips) is transient and must be returned to the queue for
// retry instead of being skipped (finding D20).
func PermanentWorkerError(err error, permanentSentinels ...error) bool {
	if err == nil {
		return false
	}
	for _, sentinel := range permanentSentinels {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
