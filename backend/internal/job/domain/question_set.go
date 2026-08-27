package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// QuestionSetVersion — the stored shape's version. Bump it whenever the JSON
// contract changes: a set written by an older/newer binary is treated as
// unusable and the deterministic templates take over instead of rendering a
// half-understood payload to a candidate.
const QuestionSetVersion = 1

// Provenance of a stored set. "llm" is the D5 default; "template" exists so a
// deterministically generated set can also be pinned to a job if a future
// batch decides to freeze the fallback.
const (
	QuestionSourceLLM      = "llm"
	QuestionSourceTemplate = "template"
)

// Question categories — same taxonomy the interview generator uses. Duplicated
// here on purpose: job/domain owns the STORED contract and must never import
// an interview package (the dependency runs interview -> job, one way only).
const (
	QuestionCategoryTechnical   = "technical"
	QuestionCategoryBehavioral  = "behavioral"
	QuestionCategorySituational = "situational"
)

// ErrQuestionSetInvalid — the payload cannot be trusted (missing framing,
// unknown category, empty set). Callers treat it as "regenerate / fall back",
// never as "store what we got".
var ErrQuestionSetInvalid = errors.New("question set invalid")

// Question is a self-explanatory interview question: the prompt the candidate
// hears plus the framing that makes it defensible.
//
//   - Context     — why this is asked for THIS job.
//   - Expectation — what a good answer demonstrates.
//
// Both are mandatory: a stored question without framing is exactly the opaque
// experience D5 set out to remove, so it is rejected at the boundary rather
// than rendered blank.
type Question struct {
	Category    string `json:"category"`
	Prompt      string `json:"prompt"`
	Context     string `json:"context"`
	Expectation string `json:"expectation"`
	Skill       string `json:"skill,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

// QuestionSet is the versioned set stored on the job (jobs.question_set). It is
// generated ONCE, at publish time, and reused for every candidate — that is
// what keeps candidates comparable and the interview start latency-free.
type QuestionSet struct {
	Version     int        `json:"version"`
	Source      string     `json:"source"`
	GeneratedAt time.Time  `json:"generated_at"`
	Questions   []Question `json:"questions"`
}

// NewQuestionSet stamps the current version on a freshly generated set.
func NewQuestionSet(source string, questions []Question, generatedAt time.Time) *QuestionSet {
	return &QuestionSet{
		Version:     QuestionSetVersion,
		Source:      source,
		GeneratedAt: generatedAt.UTC(),
		Questions:   questions,
	}
}

// Validate enforces the stored contract. The worker marshals through this, so
// a malformed provider response can never be half-written to the job row.
func (s *QuestionSet) Validate() error {
	if s == nil {
		return fmt.Errorf("%w: nil set", ErrQuestionSetInvalid)
	}
	if s.Version <= 0 {
		return fmt.Errorf("%w: version must be positive", ErrQuestionSetInvalid)
	}
	if s.Source != QuestionSourceLLM && s.Source != QuestionSourceTemplate {
		return fmt.Errorf("%w: unknown source %q", ErrQuestionSetInvalid, s.Source)
	}
	if len(s.Questions) == 0 {
		return fmt.Errorf("%w: no questions", ErrQuestionSetInvalid)
	}
	for i, q := range s.Questions {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("question %d: %w", i, err)
		}
	}
	return nil
}

// Validate — a question is only storable when it is fully self-explanatory.
func (q Question) Validate() error {
	if strings.TrimSpace(q.Prompt) == "" {
		return fmt.Errorf("%w: prompt is required", ErrQuestionSetInvalid)
	}
	if strings.TrimSpace(q.Context) == "" {
		return fmt.Errorf("%w: context (why we ask) is required", ErrQuestionSetInvalid)
	}
	if strings.TrimSpace(q.Expectation) == "" {
		return fmt.Errorf("%w: expectation (what a good answer shows) is required", ErrQuestionSetInvalid)
	}
	if !ValidQuestionCategory(q.Category) {
		return fmt.Errorf("%w: unknown category %q", ErrQuestionSetInvalid, q.Category)
	}
	return nil
}

// ValidQuestionCategory reports whether category is part of the taxonomy.
func ValidQuestionCategory(category string) bool {
	switch category {
	case QuestionCategoryTechnical, QuestionCategoryBehavioral, QuestionCategorySituational:
		return true
	}
	return false
}

// NormalizeQuestionCategory maps provider casing/whitespace onto the taxonomy;
// "" means the provider invented a category and the question is rejected.
func NormalizeQuestionCategory(category string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	if ValidQuestionCategory(c) {
		return c
	}
	return ""
}

// Marshal encodes the set for the jobs.question_set column. It validates
// first: the only write path to the column goes through here, so an invalid
// set is impossible to persist.
func (s *QuestionSet) Marshal() (json.RawMessage, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

// ParseQuestionSet decodes a stored payload without judging it — use
// SelectQuestionSet when you need the trust decision too.
func ParseQuestionSet(raw json.RawMessage) (*QuestionSet, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, fmt.Errorf("%w: empty payload", ErrQuestionSetInvalid)
	}
	var set QuestionSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuestionSetInvalid, err)
	}
	return &set, nil
}

// Usable reports whether this set can drive an interview: right version, valid
// content. Anything else means "use the deterministic templates".
func (s *QuestionSet) Usable() bool {
	return s != nil && s.Version == QuestionSetVersion && s.Validate() == nil
}

// SelectQuestionSet is the fallback-to-templates helper: it turns a raw
// jobs.question_set column into a trustworthy set, or (nil, false) meaning the
// caller must generate deterministic templates instead.
//
// Unusable covers every real-world case: NULL column (feature pre-dates the
// job), 'null', truncated JSON, a failed generation that stored nothing, and a
// set written by a different QuestionSetVersion.
func SelectQuestionSet(raw json.RawMessage) (*QuestionSet, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, false
	}
	set, err := ParseQuestionSet(raw)
	if err != nil || !set.Usable() {
		return nil, false
	}
	return set, true
}
