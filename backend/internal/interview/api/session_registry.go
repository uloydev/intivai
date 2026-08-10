package api

import "sync"

// sessionRegistry enforces single-active-connection per interview: a second
// socket for the same interview is rejected with an error frame. In-memory
// only (single instance); a distributed lock (Redis) is the multi-instance
// upgrade path.
type sessionRegistry struct {
	mu     sync.Mutex
	active map[string]struct{}
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{active: make(map[string]struct{})}
}

// TryAcquire claims the key; false when already held.
func (r *sessionRegistry) TryAcquire(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.active[key]; ok {
		return false
	}
	r.active[key] = struct{}{}
	return true
}

// Release frees the key (idempotent).
func (r *sessionRegistry) Release(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, key)
}
