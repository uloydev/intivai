package domain_test

import (
	"testing"

	"github.com/intivai/backend/internal/billing/domain"
	"github.com/stretchr/testify/require"
)

func TestPlan_MonthlyInterviewLimit(t *testing.T) {
	cases := []struct {
		plan     domain.Plan
		expected int
	}{
		{domain.PlanFree, 10},
		{domain.PlanStarter, 100},
		{domain.PlanPro, 1000},
		{domain.PlanEnterprise, 1000000},
		{domain.Plan("unknown"), 10},
	}

	for _, tc := range cases {
		require.Equal(t, tc.expected, tc.plan.MonthlyInterviewLimit())
	}
}

func TestPlan_IsValid(t *testing.T) {
	require.True(t, domain.PlanFree.IsValid())
	require.True(t, domain.PlanStarter.IsValid())
	require.True(t, domain.PlanPro.IsValid())
	require.True(t, domain.PlanEnterprise.IsValid())
	require.False(t, domain.Plan("custom").IsValid())
	require.False(t, domain.Plan("").IsValid())
}
