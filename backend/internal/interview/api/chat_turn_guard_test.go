package api

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
)

// turnGuardLLM counts ChatStream invocations so a test can assert exactly one
// stream per accepted dialogue turn.
type turnGuardLLM struct {
	mu      sync.Mutex
	streams int
}

func (l *turnGuardLLM) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.streams
}

func (l *turnGuardLLM) Chat(context.Context, llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "ok"}, nil
}

func (l *turnGuardLLM) ChatStream(ctx context.Context, _ llm.ChatRequest) (<-chan string, error) {
	l.mu.Lock()
	l.streams++
	l.mu.Unlock()
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, tok := range []string{"tok-a", "tok-b"} {
			select {
			case <-ctx.Done():
				return
			case ch <- tok:
			}
		}
	}()
	return ch, nil
}

func (l *turnGuardLLM) StructuredOutput(context.Context, llm.StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}

func (l *turnGuardLLM) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("unused")
}

func (l *turnGuardLLM) CountTokens(string) int { return 0 }

// RED (D19): a second answer arriving while the previous turn's stream is
// still active must be rejected with a turn_in_progress error frame — never
// processed. Without the busy guard it overwrites s.turn/s.streamCancel
// mid-stream: two LLM streams, interleaved tokens, duplicate next_question
// dispatches, corrupted transcript cursor.
func TestChatOverlappingAnswerRejectedWithTurnInProgress(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	mock := &turnGuardLLM{}
	app := chatApp(svc, mock, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()

	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	// Two answers back-to-back — the second lands while turn 1 still streams.
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "first answer text"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "second answer text"}); err != nil {
		t.Fatal(err)
	}

	questions, turnErr := 0, false
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m, ok := readFrame(t, conn, 300*time.Millisecond)
		if !ok {
			break // read failure is terminal for gorilla clients
		}
		switch m["type"] {
		case ivdomain.MsgQuestion:
			questions++
		case ivdomain.MsgError:
			if m["code"] == "turn_in_progress" {
				turnErr = true
			}
		}
		if turnErr && questions >= 1 {
			break
		}
	}
	if !turnErr {
		t.Fatal("second answer not rejected with a turn_in_progress error frame")
	}
	if n := mock.count(); n != 1 {
		t.Fatalf("expected exactly 1 LLM stream for the accepted turn, got %d — overlapping answer was processed", n)
	}
	// Drain briefly: no additional question may surface after the guarded turn.
	extra := 0
	for {
		m, ok := readFrame(t, conn, 700*time.Millisecond)
		if !ok {
			break
		}
		if m["type"] == ivdomain.MsgQuestion {
			extra++
		}
	}
	if total := questions + extra; total != 1 {
		t.Fatalf("expected exactly 1 next_question dispatch, got %d", total)
	}
}

// Regression (D19): interrupt must clear the turn-active guard — the answer
// sent after "Interrupted." + next question must be accepted and streamed.
func TestChatInterruptClearsTurnGuardForNextAnswer(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	app := chatApp(svc, slowStreamLLM{}, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	conn, code := dialChatWS(t, ln.Addr().String(), ivID, ticket, "")
	if conn == nil {
		t.Fatalf("dial failed: %d", code)
	}
	defer func() { _ = conn.Close() }()

	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no start frame")
	}
	if _, ok := readFrame(t, conn, 5*time.Second); !ok {
		t.Fatal("no question frame")
	}

	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "initial answer"}); err != nil {
		t.Fatal(err)
	}
	firstToken := false
	for !firstToken {
		m, ok := readFrame(t, conn, 5*time.Second)
		if !ok {
			t.Fatal("stream never produced a token")
		}
		firstToken = m["type"] == ivdomain.MsgToken
	}

	if err := conn.WriteJSON(map[string]string{"type": "interrupt"}); err != nil {
		t.Fatal(err)
	}
	gotInterrupted, gotQuestion := false, false
	for !(gotInterrupted && gotQuestion) { //nolint:staticcheck // QF1001: both flags read clearer as positive conjunction here
		m, ok := readFrame(t, conn, 3*time.Second)
		if !ok {
			t.Fatalf("interrupt flow incomplete: interrupted=%v question=%v", gotInterrupted, gotQuestion)
		}
		switch m["type"] {
		case ivdomain.MsgResponse:
			if m["content"] == "Interrupted." {
				gotInterrupted = true
			}
		case ivdomain.MsgQuestion:
			gotQuestion = true
		case ivdomain.MsgError:
			t.Fatalf("unexpected error frame during interrupt: %v", m)
		}
	}

	// The follow-up answer must proceed — the guard was cleared by the
	// interrupted stream's teardown.
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "post-interrupt answer"}); err != nil {
		t.Fatal(err)
	}
	gotTokenAfter := false
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !gotTokenAfter {
		m, ok := readFrame(t, conn, 500*time.Millisecond)
		if !ok {
			break // read failure is terminal for gorilla clients
		}
		if m["type"] == ivdomain.MsgToken {
			gotTokenAfter = true
		}
		if m["type"] == ivdomain.MsgError {
			t.Fatalf("post-interrupt answer rejected: %v", m)
		}
	}
	if !gotTokenAfter {
		t.Fatal("answer after interrupt never streamed — turn guard was not cleared")
	}
}
