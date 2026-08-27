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
	sharederr "github.com/intivai/backend/internal/shared/errors"
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
	// runSemaphore (D24): bounds concurrent container executions GLOBALLY —
	// the HTTP /sandbox/execute path and the WS code.run path both spawn
	// containers; without a cap, N concurrent requests fan out N container
	// spawns. Saturation REFUSES (returns a domain error) instead of queuing:
	// a queued run would pin a connection past its timeout.
	runSemaphore chan struct{}
}

func NewSandboxService(pool *gorm.DB, r CodeRunner, p llm.Provider, ivRepo ivdomain.InterviewRepository) *SandboxService {
	return &SandboxService{
		pool:         pool,
		runner:       r,
		llm:          p,
		ivRepo:       ivRepo,
		runSemaphore: make(chan struct{}, maxConcurrentExecutions),
	}
}

// maxConcurrentExecutions bounds simultaneous sandbox container runs.
const maxConcurrentExecutions = 4

// maxTestCasesPerRequest caps test cases per execution request (D24) —
// mirrors the WS-side maxSandboxTestCases guard so the HTTP path cannot
// smuggle unbounded subprocess spawns.
const maxTestCasesPerRequest = 20

// Execute runs the code snippet in a throwaway container via the sidecar.
func (s *SandboxService) Execute(ctx context.Context, req domain.ExecutionRequest) (*domain.ExecutionResult, error) {
	select {
	case s.runSemaphore <- struct{}{}:
		defer func() { <-s.runSemaphore }()
	default:
		return nil, sharederr.NewDomainError("SANDBOX_BUSY", "too many sandbox executions in progress; try again in a moment")
	}
	if len(req.TestCases) > maxTestCasesPerRequest {
		req.TestCases = req.TestCases[:maxTestCasesPerRequest]
	}
	return s.runner.Execute(ctx, req)
}

// EvaluateCode performs AI code quality and time/space complexity analysis.
func (s *SandboxService) EvaluateCode(ctx context.Context, orgID string, language domain.Language, code, problemDescription string) (*domain.AICodeReview, error) {
	if s.llm == nil {
		return &domain.AICodeReview{
			TimeComplexity:  "N/A",
			SpaceComplexity: "N/A",
			QualityScore:    0,
			Summary:         "AI evaluation skipped due to missing provider configuration.",
			Strengths:       []string{},
			Improvements:    []string{},
		}, nil
	}

	userPrompt := fmt.Sprintf("Problem Description: %s\n\nLanguage: %s\n\nCandidate Code:\n%s", problemDescription, language, code)
	out, err := s.llm.StructuredOutput(ctx, llm.StructuredRequest{
		OrgID:  orgID,
		System: codeReviewSystem,
		User:   userPrompt,
		Schema: &domain.AICodeReview{},
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
	var finalRes *ivdomain.ExecutionResult
	if raw, err := json.Marshal(session.FinalResult); err != nil {
		return fmt.Errorf("marshal final coding result: %w", err)
	} else if err := json.Unmarshal(raw, &finalRes); err != nil {
		return fmt.Errorf("unmarshal final coding result: %w", err)
	}
	var aiReview *ivdomain.CodeReview
	if raw, err := json.Marshal(session.AICodeReview); err != nil {
		return fmt.Errorf("marshal coding review: %w", err)
	} else if err := json.Unmarshal(raw, &aiReview); err != nil {
		return fmt.Errorf("unmarshal coding review: %w", err)
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
