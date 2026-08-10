package api

import (
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
)

// RED: WS upgrade must honor the origin allowlist — disallowed origins fail
// the handshake.
func TestWSServerRejectsDisallowedOrigin(t *testing.T) {
	allowed := []string{"http://allowed.example"}
	app := fiber.New()
	app.Get("/ws", fiberws.New(func(c *fiberws.Conn) {
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}, fiberws.Config{Origins: allowed}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}

	// Allowed origin passes.
	conn2, _, err := dialer.Dial("ws://"+ln.Addr().String()+"/ws", map[string][]string{"Origin": {"http://allowed.example"}})
	if err != nil {
		t.Fatalf("allowed origin rejected: %v", err)
	}
	_ = conn2.Close()

	// Disallowed origin fails the handshake.
	_, _, err = dialer.Dial("ws://"+ln.Addr().String()+"/ws", map[string][]string{"Origin": {"http://evil.example"}})
	if err == nil {
		t.Fatal("disallowed origin accepted")
	}
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("unexpected error: %v", err)
	}
}

var _ = httptest.NewRequest
