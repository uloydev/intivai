package domain

import (
	"encoding/json"
	"fmt"
)

// WebSocket message types — contract from AI_Interviewer_Phases.md §WebSocket
// Protocol. Server→client and client→server directions.
const (
	MsgStart      = "interview.start"
	MsgQuestion   = "question"
	MsgToken      = "token"
	MsgResponse   = "response"
	MsgEvaluation = "evaluation"
	MsgError      = "error"
	MsgPong       = "pong"

	MsgAnswer     = "answer"
	MsgInterrupt  = "interrupt"
	MsgPing       = "ping"
	MsgResume     = "resume"
	MsgTelemetry  = "telemetry"
	MsgCodeChange = "code.change"
	MsgCodeRun    = "code.run"
	MsgCodeResult = "code.result"
)

// Server → client messages.
type InterviewStartMessage struct {
	Type             string `json:"type"`
	SessionID        string `json:"session_id"`
	TotalQuestions   int    `json:"total_questions"`
	SessionBudgetSec int    `json:"session_budget_sec,omitempty"` // e.g. 1800 (30 mins)
}

type QuestionMessage struct {
	Type                string `json:"type"`
	Content             string `json:"content"`
	Idx                 int    `json:"idx"`
	Archetype           string `json:"archetype,omitempty"`             // "conversational" | "system_design" | "coding"
	TimeLimitSec        int    `json:"time_limit_sec,omitempty"`        // allocated seconds for this question
	SessionRemainingSec int    `json:"session_remaining_sec,omitempty"` // global interview clock remaining
}

type TokenMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ResponseMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Evaluation status values.
const (
	EvalComplete = "complete"
	EvalPending  = "pending"
)

type EvaluationMessage struct {
	Type           string             `json:"type"`
	Scores         map[string]float64 `json:"scores"`
	Overall        float64            `json:"overall,omitempty"`
	Recommendation string             `json:"recommendation,omitempty"`
	Status         string             `json:"status"` // EvalComplete | EvalPending
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"` // machine-readable domain error code
	Message string `json:"message"`
}

// PacingMetrics captures candidate latency and typing authenticity telemetry.
type PacingMetrics struct {
	TimeToFirstKeystrokeMs int64   `json:"time_to_first_keystroke_ms,omitempty"`
	DurationMs             int64   `json:"duration_ms,omitempty"`
	TypedChars             int     `json:"typed_chars,omitempty"`
	PastedChars            int     `json:"pasted_chars,omitempty"`
	PastedRatio            float64 `json:"pasted_ratio,omitempty"`
}

// Client → server messages.
type AnswerMessage struct {
	Type            string         `json:"type"`
	Content         string         `json:"content"`
	Idx             int            `json:"idx"`
	PacingTelemetry *PacingMetrics `json:"pacing_telemetry,omitempty"`
}

type InterruptMessage struct {
	Type string `json:"type"`
}

type PingMessage struct {
	Type string `json:"type"`
}

type ResumeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

type TelemetryDetails struct {
	PastedTextLength int    `json:"pasted_text_length,omitempty"`
	WindowWidth      int    `json:"window_width,omitempty"`
	WindowHeight     int    `json:"window_height,omitempty"`
	AudioStatus      string `json:"audio_status,omitempty"`
}

type TelemetryMessage struct {
	Type        string            `json:"type"`
	EventType   string            `json:"event_type"` // tab_switch, paste, focus_lost, focus_regained, window_resize, audio_anomaly
	Timestamp   string            `json:"timestamp"`
	QuestionIdx int               `json:"question_idx,omitempty"`
	Details     *TelemetryDetails `json:"details,omitempty"`
}

type CodeChangeMessage struct {
	Type        string `json:"type"`
	QuestionIdx int    `json:"question_idx,omitempty"`
	Language    string `json:"language"`
	Code        string `json:"code"`
}

type CodeRunTestCase struct {
	ID             string `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Hidden         bool   `json:"hidden,omitempty"`
}

type CodeRunMessage struct {
	Type        string            `json:"type"`
	QuestionIdx int               `json:"question_idx,omitempty"`
	Language    string            `json:"language"`
	Code        string            `json:"code"`
	Stdin       string            `json:"stdin,omitempty"`
	TestCases   []CodeRunTestCase `json:"test_cases,omitempty"`
}

type TestResult struct {
	ID             string `json:"id"`
	Passed         bool   `json:"passed"`
	ActualOutput   string `json:"actual_output,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	Error          string `json:"error,omitempty"`
}

type ExecutionResult struct {
	Stdout      string       `json:"stdout"`
	Stderr      string       `json:"stderr"`
	ExitCode    int          `json:"exit_code"`
	DurationMs  int64        `json:"duration_ms"`
	AllPassed   bool         `json:"all_passed"`
	TestResults []TestResult `json:"test_results,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// CodeReview — the WS frame view of an AI code review. Deliberately NOT the
// sandbox domain's AICodeReview (which carries complexity/quality analysis):
// this is the wire shape the chat client renders, and the two evolve
// independently across the gRPC/WS boundaries.
type CodeReview struct {
	Score           float64  `json:"score"`
	Strengths       []string `json:"strengths,omitempty"`
	Weaknesses      []string `json:"weaknesses,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

type CodeResultMessage struct {
	Type        string       `json:"type"`
	Stdout      string       `json:"stdout"`
	Stderr      string       `json:"stderr"`
	ExitCode    int          `json:"exit_code"`
	DurationMs  int64        `json:"duration_ms"`
	AllPassed   bool         `json:"all_passed"`
	TestResults []TestResult `json:"test_results,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type CodingSession struct {
	QuestionIdx  int              `json:"question_idx"`
	Language     string           `json:"language"`
	Code         string           `json:"code"`
	FinalResult  *ExecutionResult `json:"final_result,omitempty"`
	AICodeReview *CodeReview      `json:"ai_code_review,omitempty"`
	SubmittedAt  string           `json:"submitted_at"`
}

func NewInterviewStart(sessionID string, total int) InterviewStartMessage {
	return InterviewStartMessage{Type: MsgStart, SessionID: sessionID, TotalQuestions: total}
}

// ParseClientMessage decodes a client frame and validates its type.
// Unknown or malformed types are rejected — the server never tolerates
// unexpected frames on the candidate socket.
func ParseClientMessage(raw []byte) (any, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("malformed frame: %w", err)
	}
	switch envelope.Type {
	case MsgAnswer:
		var m AnswerMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if m.Content == "" {
			return nil, fmt.Errorf("answer without content")
		}
		return m, nil
	case MsgInterrupt:
		var m InterruptMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	case MsgPing:
		var m PingMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	case MsgResume:
		var m ResumeMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	case MsgTelemetry:
		var m TelemetryMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if m.EventType == "" {
			return nil, fmt.Errorf("telemetry without event_type")
		}
		return m, nil
	case MsgCodeChange:
		var m CodeChangeMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	case MsgCodeRun:
		var m CodeRunMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unknown message type %q", envelope.Type)
	}
}
