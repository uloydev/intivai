package api

import (
	"net"
	"net/http"
	"testing"
	"time"

	"net/http/httptest"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
)

// Harness proof: fiber + gofiber/contrib/websocket upgrade works, and a
// gorilla client can complete the handshake + round-trip frames. This is the
// base for the M3 protocol tests (ticket auth, resume, interrupt follow in
// the chat handler tests).

func newHarnessApp() *fiber.App {
	app := fiber.New()
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/echo", fiberws.New(func(c *fiberws.Conn) {
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			if string(msg) == "ping" {
				if err := c.WriteMessage(mt, []byte("pong")); err != nil {
					return
				}
				continue
			}
			if err := c.WriteMessage(mt, msg); err != nil {
				return
			}
		}
	}))
	return app
}

func startHarness(t *testing.T) (string, func()) {
	t.Helper()
	app := newHarnessApp()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	addr := ln.Addr().String()
	return addr, func() { _ = app.Shutdown() }
}

func dialWS(t *testing.T, addr, path string) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, _, err := dialer.Dial("ws://"+addr+path, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestWSUpgradeAndEcho(t *testing.T) {
	addr, stop := startHarness(t)
	defer stop()

	conn := dialWS(t, addr, "/ws/echo")
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "hello" {
		t.Fatalf("echo = %q", msg)
	}
}

func TestWSPingPong(t *testing.T) {
	addr, stop := startHarness(t)
	defer stop()

	conn := dialWS(t, addr, "/ws/echo")
	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "pong" {
		t.Fatalf("pong = %q", msg)
	}
}

func TestWSRejectsNonUpgrade(t *testing.T) {
	app := newHarnessApp()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ws/echo", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, want 426", resp.StatusCode)
	}
}
