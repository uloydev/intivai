package service

// Context management (Research §2, InterviewContext):
// sliding window of the last N Q&A pairs + a total token budget tracked with
// the same counter the LLM provider uses (tiktoken cl100k_base, 5-10% drift).

// Role constants for history messages.
const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
)

// ContextMessage — one conversation turn in the LLM history window.
type ContextMessage struct {
	Role    string
	Content string
}

const (
	// DefaultContextWindow — Q&A pairs kept in the sliding window (10 Q&A).
	DefaultContextWindow = 10
	// DefaultTokenBudget — max context tokens for the whole message list.
	DefaultTokenBudget = 8000
)

// TrimContext keeps the last `window` Q&A pairs (assistant + user) plus a
// leading system message when one precedes the kept range. History is assumed
// well-formed: each user message is preceded by its assistant question.
func TrimContext(msgs []ContextMessage, window int) []ContextMessage {
	if len(msgs) == 0 {
		return nil
	}
	if window <= 0 {
		for _, m := range msgs {
			if m.Role == RoleSystem {
				return []ContextMessage{m}
			}
		}
		return nil
	}

	pairs := 0
	start := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != RoleUser {
			continue
		}
		pairs++
		if pairs == window {
			start = i - 1 // assistant immediately precedes the user pair
			if start < 0 {
				start = 0
			}
			break
		}
	}
	if pairs < window {
		start = 0
	}

	// Pull a leading system message into the kept range if one exists.
	for i := start - 1; i >= 0; i-- {
		if msgs[i].Role == RoleSystem {
			out := make([]ContextMessage, 0, 1+len(msgs)-start)
			out = append(out, msgs[i])
			out = append(out, msgs[start:]...)
			return out
		}
	}
	return msgs[start:]
}

// CountContextTokens sums the token counter over every message content.
func CountContextTokens(counter func(string) int, msgs []ContextMessage) int {
	total := 0
	for _, m := range msgs {
		total += counter(m.Content)
	}
	return total
}

// ExceedsBudget reports whether the message list overruns the budget.
func ExceedsBudget(msgs []ContextMessage, budget int, counter func(string) int) bool {
	return CountContextTokens(counter, msgs) > budget
}
