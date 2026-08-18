package domain_test

import (
	"testing"

	"github.com/intivai/backend/internal/sandbox/domain"
	"github.com/stretchr/testify/assert"
)

func TestCompareOutputs(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		passed   bool
	}{
		{
			name:     "exact match",
			actual:   "[0, 1]",
			expected: "[0, 1]",
			passed:   true,
		},
		{
			name:     "crlf vs lf and trailing whitespace",
			actual:   "[0, 1]\r\n  ",
			expected: "[0, 1]\n",
			passed:   true,
		},
		{
			name:     "mismatch",
			actual:   "42",
			expected: "43",
			passed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.CompareOutputs(tt.actual, tt.expected)
			assert.Equal(t, tt.passed, got)
		})
	}
}
