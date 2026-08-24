package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/internal/llm"
	"github.com/rs/zerolog"
)

// floodLLM streams 300 tokens — far more than the 64-frame writer buffer — so
// killing the transport mid-stream guarantees senders end up parked on a full
// channel whose drain goroutine is dead (the D18 precondition).
type floodLLM struct{}

func (floodLLM) Chat(context.Context, llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "flood"}, nil
}

func (floodLLM) ChatStream(ctx context.Context, _ llm.ChatRequest) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for i := 0; i < 300; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- fmt.Sprintf("tok%d", i):
			}
		}
	}()
	return ch, nil
}

func (floodLLM) StructuredOutput(context.Context, llm.StructuredRequest) (any, error) {
	return nil, errors.New("unused")
}

func (floodLLM) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("unused")
}

func (floodLLM) CountTokens(string) int { return 0 }

// releaseSpyRegistry records Release calls — the observable proxy for "the
// handler goroutine ran its teardown to completion" (Release is deferred
// after w.close, so it can only fire once close has returned).
type releaseSpyRegistry struct {
	*MemorySessionRegistry
	released chan struct{}
}

func (r *releaseSpyRegistry) Release(ctx context.Context, key, sessionID string) error {
	err := r.MemorySessionRegistry.Release(ctx, key, sessionID)
	select {
	case r.released <- struct{}{}:
	default:
	}
	return err
}

// RED (D18): transport death mid-stream must never wedge the handler. The
// broken teardown order (writer close deferred before cancel) deadlocks:
// senders park holding the writer mutex on a full buffer nobody drains,
// close() waits for that mutex forever, and cancel() — the only thing that
// would wake the senders — runs only after close() returns. The session lock
// then strands until TTL and reconnects are rejected with "already active".
func TestChatTransportKillReleasesSessionLock(t *testing.T) {
	_, svc, orgID, appID := seedChatOrg(t)
	ivID, ticket := createInterviewAndTicket(t, svc, orgID, appID)

	reg := &releaseSpyRegistry{MemorySessionRegistry: NewMemorySessionRegistry(), released: make(chan struct{}, 1)}
	app := fiber.New()
	handler := NewChatHandler(svc, floodLLM{}, auth.NewJWTProvider("test-secret-for-chat-flow"), zerolog.Nop(), reg)
	app.Get("/candidate/interviews/:id/chat", handler.RequireTicket, handler.Chat(nil))

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
	if err := conn.WriteJSON(map[string]any{"type": "answer", "content": "my answer", "idx": 1}); err != nil {
		t.Fatal(err)
	}
	firstToken := false
	for !firstToken {
		m, ok := readFrame(t, conn, 5*time.Second)
		if !ok {
			t.Fatal("stream never produced a token")
		}
		if m["type"] == ivdomain.MsgToken {
			firstToken = true
		}
	}

	// Kill the transport mid-stream: the writer goroutine dies on its next
	// WriteJSON, remaining tokens overflow the 64-slot buffer, and the read
	// loop exits on the dead socket — handler teardown begins under load.
	if err := conn.UnderlyingConn().Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-reg.released:
		// Handler tore down and released the session lock in time.
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit within 2s of transport death: wsWriter close deadlocked, session lock stranded")
	}
}
