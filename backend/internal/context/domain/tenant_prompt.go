package domain

import (
	"fmt"
	"strings"

	"github.com/intivai/backend/internal/shared/errors"
)

const MaxPromptLength = 4000

var forbiddenPromptTokens = []string{
	"ignore previous instructions",
	"ignore all instructions",
	"disregard previous",
	"disregard all",
	"jailbreak",
	"override your instructions",
	"override instructions",
	"you are now",
	"leak your system",
	"reveal your instructions",
}

// ContainsInjection reports whether content carries prompt-injection rails.
// Shared by tenant prompt validation and company context indexing (context
// content is composed into the interview system prompt at M3).
func ContainsInjection(content string) bool {
	lower := strings.ToLower(content)
	for _, token := range forbiddenPromptTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// ValidatePrompt rejects prompt injection rails: length + forbidden keywords.
func ValidatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.NewDomainError("PROMPT_REQUIRED", "system prompt is required")
	}
	if len(prompt) > MaxPromptLength {
		return errors.NewDomainError("PROMPT_TOO_LONG", fmt.Sprintf("system prompt exceeds %d characters", MaxPromptLength))
	}
	if ContainsInjection(prompt) {
		return errors.NewDomainError("PROMPT_INJECTION", "system prompt contains forbidden content")
	}
	return nil
}

func DefaultPrompt() string {
	return "You are Intivai, an AI interviewer conducting structured job interviews. " +
		"Ask one question at a time, stay professional, and evaluate answers objectively."
}
