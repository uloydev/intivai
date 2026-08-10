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

	MsgAnswer    = "answer"
	MsgInterrupt = "interrupt"
	MsgPing      = "ping"
	MsgResume    = "resume"
)

// Server → client messages.
type InterviewStartMessage struct {
	Type           string `json:"type"`
	SessionID      string `json:"session_id"`
	TotalQuestions int    `json:"total_questions"`
}

type QuestionMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Idx     int    `json:"idx"`
}

type TokenMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ResponseMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type EvaluationMessage struct {
	Type   string             `json:"type"`
	Scores map[string]float64 `json:"scores"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Client → server messages.
type AnswerMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Idx     int    `json:"idx"`
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
	default:
		return nil, fmt.Errorf("unknown message type %q", envelope.Type)
	}
}
