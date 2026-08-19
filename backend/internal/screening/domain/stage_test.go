package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageValidity(t *testing.T) {
	for _, s := range []Stage{
		StageApplied, StageScreeningPassed, StageScreeningFailed,
		StageInterviewInvited, StageInterviewCompleted, StageOfferExtended,
		StageHired, StageRejected,
	} {
		require.True(t, s.IsValid(), "%q should be valid", s)
	}
	require.False(t, Stage("banana").IsValid())
	require.False(t, Stage("").IsValid())
}

func TestStageTransitions(t *testing.T) {
	// Ladder order
	require.True(t, StageApplied.CanTransitionTo(StageScreeningPassed, false))
	require.True(t, StageScreeningPassed.CanTransitionTo(StageInterviewInvited, false))
	require.True(t, StageInterviewInvited.CanTransitionTo(StageInterviewCompleted, false))
	require.True(t, StageInterviewCompleted.CanTransitionTo(StageOfferExtended, false))
	require.True(t, StageOfferExtended.CanTransitionTo(StageHired, false))
	// Skip steps forward is allowed (recruiter jump)
	require.True(t, StageApplied.CanTransitionTo(StageOfferExtended, false))
	// Idempotent same-stage
	require.True(t, StageHired.CanTransitionTo(StageHired, false))
	// Terminal states reachable from anywhere
	require.True(t, StageInterviewInvited.CanTransitionTo(StageRejected, false))
	require.True(t, StageHired.CanTransitionTo(StageRejected, false))
	require.True(t, StageApplied.CanTransitionTo(StageScreeningFailed, false))
	require.True(t, StageInterviewCompleted.CanTransitionTo(StageScreeningFailed, false))
}

func TestStageBackward(t *testing.T) {
	// Backward moves are now allowed for manual reversion.
	require.True(t, StageHired.CanTransitionTo(StageOfferExtended, false))
	require.True(t, StageScreeningPassed.CanTransitionTo(StageApplied, false))
	require.True(t, StageRejected.CanTransitionTo(StageApplied, false))
}

func TestStageFromNil(t *testing.T) {
	// No decision yet — any valid stage is a first decision
	for _, s := range []Stage{StageApplied, StageHired, StageRejected, StageScreeningFailed} {
		require.True(t, s.CanTransitionTo(s, true), "first decision to %q should be allowed", s)
	}
}

func TestStageLadder(t *testing.T) {
	require.Equal(t, 6, len(StageLadder))
	require.Equal(t, StageApplied, StageLadder[0])
	require.Equal(t, StageHired, StageLadder[len(StageLadder)-1])
}
