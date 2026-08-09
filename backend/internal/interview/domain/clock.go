package domain

import "time"

// Clock — injectable time source. Idle timeouts, interview expiry and token
// lifetimes use this; tests inject FrozenClock instead of waiting real time.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// SystemClock returns the real clock.
func SystemClock() Clock { return systemClock{} }

// FrozenClock returns a clock pinned to a fixed instant (tests, deterministic
// expiry checks).
func FrozenClock(at time.Time) Clock { return frozenClock{at: at.UTC()} }

type frozenClock struct{ at time.Time }

func (f frozenClock) Now() time.Time { return f.at }
