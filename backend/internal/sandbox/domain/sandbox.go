package domain

import (
	"strings"
	"time"
)

type Language string

const (
	LanguageGo         Language = "go"
	LanguagePython     Language = "python"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
)

type TestCase struct {
	ID             string `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Hidden         bool   `json:"hidden,omitempty"`
}

type TestCaseResult struct {
	TestCase     TestCase `json:"test_case"`
	ActualOutput string   `json:"actual_output"`
	Passed       bool     `json:"passed"`
	DurationMs   int64    `json:"duration_ms"`
	Error        string   `json:"error,omitempty"`
}

type ExecutionRequest struct {
	Language   Language   `json:"language"`
	Code       string     `json:"code"`
	Stdin      string     `json:"stdin,omitempty"`
	TestCases  []TestCase `json:"test_cases,omitempty"`
	TimeoutSec int        `json:"timeout_sec,omitempty"` // max 5s default
}

type ExecutionResult struct {
	Stdout      string           `json:"stdout"`
	Stderr      string           `json:"stderr"`
	ExitCode    int              `json:"exit_code"`
	DurationMs  int64            `json:"duration_ms"`
	MemoryKB    int64            `json:"memory_kb"`
	TestResults []TestCaseResult `json:"test_results,omitempty"`
	AllPassed   bool             `json:"all_passed"`
	Error       string           `json:"error,omitempty"`
}

type AICodeReview struct {
	TimeComplexity  string   `json:"time_complexity"`
	SpaceComplexity string   `json:"space_complexity"`
	QualityScore    int      `json:"quality_score"` // 0-100
	Summary         string   `json:"summary"`
	Strengths       []string `json:"strengths"`
	Improvements    []string `json:"improvements"`
}

type CodingSession struct {
	QuestionIdx  int             `json:"question_idx"`
	Language     Language        `json:"language"`
	Code         string          `json:"code"`
	FinalResult  ExecutionResult `json:"final_result"`
	AICodeReview *AICodeReview   `json:"ai_code_review,omitempty"`
	SubmittedAt  time.Time       `json:"submitted_at"`
}

// CompareOutputs normalizes and compares actual output against expected output.
func CompareOutputs(actual, expected string) bool {
	normActual := strings.TrimSpace(strings.ReplaceAll(actual, "\r\n", "\n"))
	normExpected := strings.TrimSpace(strings.ReplaceAll(expected, "\r\n", "\n"))
	return normActual == normExpected
}
