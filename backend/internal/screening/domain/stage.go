package domain

// Stage — the recruiter's authoritative hiring decision for an application
// (ADR-0001). Null until a recruiter decides; NEVER derived from Status
// (pipeline mechanics), never written by workers.
type Stage string

const (
	StageApplied            Stage = "applied"
	StageScreeningPassed    Stage = "screening_passed"
	StageScreeningFailed    Stage = "screening_failed"
	StageInterviewInvited   Stage = "interview_invited"
	StageInterviewCompleted Stage = "interview_completed"
	StageOfferExtended      Stage = "offer_extended"
	StageHired              Stage = "hired"
	StageRejected           Stage = "rejected"
)

// StageLadder — the forward progression order. screening_failed and
// rejected are terminal states outside the ladder.
var StageLadder = []Stage{
	StageApplied,
	StageScreeningPassed,
	StageInterviewInvited,
	StageInterviewCompleted,
	StageOfferExtended,
	StageHired,
}

var stageOrder = func() map[Stage]int {
	m := make(map[Stage]int, len(StageLadder))
	for i, s := range StageLadder {
		m[s] = i
	}
	return m
}()

// IsValid reports whether s is a known stage.
func (s Stage) IsValid() bool {
	if _, ok := stageOrder[s]; ok {
		return true
	}
	switch s {
	case StageScreeningFailed, StageRejected:
		return true
	}
	return false
}

// terminal reports whether s ends the ladder (no forward moves from it).
func (s Stage) terminal() bool {
	return s == StageScreeningFailed || s == StageRejected
}

// RequiresAdmin reports whether the transition is a backward/correction move
// (ADR-0001): leaving a terminal state, or moving down the ladder.
func (s Stage) RequiresAdmin(next Stage) bool {
	if !s.IsValid() || !next.IsValid() || s == next {
		return false
	}
	if s.terminal() {
		return true // leaving a terminal state is a correction
	}
	if next.terminal() {
		return false
	}
	return stageOrder[next] < stageOrder[s]
}

// CanTransitionTo — the transition rule (ADR-0001): same stage (idempotent),
// forward along the ladder (skipping allowed), or a terminal state from
// anywhere. Backward moves require an admin override (see RequiresAdmin).
// fromNil means the application currently has no decision — any valid stage
// is allowed.
func (s Stage) CanTransitionTo(next Stage, fromNil bool) bool {
	if !s.IsValid() || !next.IsValid() {
		return false
	}
	if fromNil {
		return true
	}
	if s == next {
		return true
	}
	if next.terminal() {
		return true
	}
	if s.terminal() {
		return false // only corrections (admin) leave a terminal state
	}
	return stageOrder[next] > stageOrder[s]
}
