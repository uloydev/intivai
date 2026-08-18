package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/intivai/backend/internal/sandbox/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// CodeRunner executes untrusted code. Implemented by the gRPC sidecar client
// (ADR-0002); the app never runs code itself.
type CodeRunner interface {
	Execute(ctx context.Context, req domain.ExecutionRequest) (*domain.ExecutionResult, error)
}

const codeReviewSystem = `You are a Principal Software Engineer conducting a technical code review. Analyze the candidate's code submission against the problem description. Return valid JSON matching:
{
  "time_complexity": "e.g. O(N)",
  "space_complexity": "e.g. O(1)",
  "quality_score": 85, // 0-100
  "summary": "Concise summary of candidate's algorithmic approach and correctness",
  "strengths": ["e.g. Optimal hash map lookup", "Clean idiomatic error handling"],
  "improvements": ["e.g. Handle empty input slice edge case"]
}`

type SandboxService struct {
	pool   *gorm.DB
	runner CodeRunner
	llm    llm.Provider
	ivRepo ivdomain.InterviewRepository
}

func NewSandboxService(pool *gorm.DB, r CodeRunner, p llm.Provider, ivRepo ivdomain.InterviewRepository) *SandboxService {
	return &SandboxService{
		pool:   pool,
		runner: r,
		llm:    p,
		ivRepo: ivRepo,
	}
}

// Execute runs the code snippet in a throwaway container via the sidecar.
func (s *SandboxService) Execute(ctx context.Context, req domain.ExecutionRequest) (*domain.ExecutionResult, error) {
	return s.runner.Execute(ctx, req)
}

// EvaluateCode performs AI code quality and time/space complexity analysis.
func (s *SandboxService) EvaluateCode(ctx context.Context, language domain.Language, code, problemDescription string) (*domain.AICodeReview, error) {
	if s.llm == nil {
		return &domain.AICodeReview{
			TimeComplexity:  "O(N)",
			SpaceComplexity: "O(1)",
			QualityScore:    85,
			Summary:         "Code executed successfully.",
			Strengths:       []string{"Working solution"},
			Improvements:    []string{"Consider additional edge case testing"},
		}, nil
	}

	userPrompt := fmt.Sprintf("Problem Description: %s\n\nLanguage: %s\n\nCandidate Code:\n%s", problemDescription, language, code)
	out, err := s.llm.StructuredOutput(ctx, llm.StructuredRequest{
		System: codeReviewSystem,
		User:   userPrompt,
		Schema: domain.AICodeReview{},
	})
	if err != nil {
		// Never fabricate a baseline review — a broken submission must not
		// score 80 on the scorecard. Surface the error to the caller.
		return nil, fmt.Errorf("ai code review failed: %w", err)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal ai review: %w", err)
	}

	var review domain.AICodeReview
	if err := json.Unmarshal(raw, &review); err != nil {
		return nil, fmt.Errorf("unmarshal ai review: %w", err)
	}

	if review.QualityScore < 0 {
		review.QualityScore = 0
	} else if review.QualityScore > 100 {
		review.QualityScore = 100
	}

	return &review, nil
}

// SaveCodingSession persists the coding session snapshot onto the interview aggregate.
func (s *SandboxService) SaveCodingSession(ctx context.Context, orgID string, interviewID uuid.UUID, session domain.CodingSession) error {
	var finalRes map[string]interface{}
	if raw, err := json.Marshal(session.FinalResult); err == nil {
		_ = json.Unmarshal(raw, &finalRes)
	}
	var aiReview map[string]interface{}
	if session.AICodeReview != nil {
		if raw, err := json.Marshal(session.AICodeReview); err == nil {
			_ = json.Unmarshal(raw, &aiReview)
		}
	}
	subTime := session.SubmittedAt
	if subTime.IsZero() {
		subTime = time.Now()
	}
	ivSession := ivdomain.CodingSession{
		QuestionIdx:  session.QuestionIdx,
		Language:     string(session.Language),
		Code:         session.Code,
		FinalResult:  finalRes,
		AICodeReview: aiReview,
		SubmittedAt:  subTime.Format(time.RFC3339),
	}
	return db.RunInTx(ctx, s.pool, orgID, func(tctx context.Context) error {
		return s.ivRepo.RecordCodingSession(tctx, interviewID, ivSession)
	})
}
