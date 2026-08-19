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

// CanTransitionTo — the transition rule (ADR-0001): same stage (idempotent),
// forward along the ladder (skipping allowed), or a terminal state from
// anywhere. Backward moves (corrections) require admin. fromNil means the
// application currently has no decision — any valid stage is allowed.
func (s Stage) CanTransitionTo(next Stage, fromNil bool) bool {
	if !s.IsValid() || !next.IsValid() {
		return false
	}
	return true
}
