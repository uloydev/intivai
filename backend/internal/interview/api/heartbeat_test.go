package api

import (
	"sync/atomic"
	"testing"
	"time"
)

type fakePingConn struct {
	pings      atomic.Int64
	writeDelay time.Duration
}

func (f *fakePingConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	f.pings.Add(1)
	if f.writeDelay > 0 {
		time.Sleep(f.writeDelay)
	}
	return nil
}

func (f *fakePingConn) pingCount() int64 { return f.pings.Load() }

// Server must keep pinging while the client answers pongs.
func TestHeartbeatKeepsPingingWithPong(t *testing.T) {
	conn := &fakePingConn{}
	pong := make(chan struct{}, 10)
	done := make(chan struct{})
	defer close(done)

	timeout := make(chan bool, 1)
	go func() { timeout <- heartbeat(conn, 20*time.Millisecond, 100*time.Millisecond, pong, done) }()

	// Answer every ping promptly.
	go func() {
		for range time.Tick(25 * time.Millisecond) {
			select {
			case pong <- struct{}{}:
			default:
			}
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	time.Sleep(150 * time.Millisecond)
	if conn.pingCount() < 3 {
		t.Fatalf("pings = %d, want >= 3 while ponging", conn.pingCount())
	}
	select {
	case <-timeout:
		t.Fatal("heartbeat stopped despite pongs")
	case <-time.After(20 * time.Millisecond):
	}
}

// Silent client (no pong within the wait window) must be dropped.
func TestHeartbeatDropsSilentClient(t *testing.T) {
	conn := &fakePingConn{}
	pong := make(chan struct{}, 1)
	done := make(chan struct{})
	defer close(done)

	start := time.Now()
	timedOut := heartbeat(conn, 10*time.Millisecond, 30*time.Millisecond, pong, done)
	if !timedOut {
		t.Fatal("heartbeat did not time out on silent client")
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Fatalf("timeout took %v, want ~pongWait", elapsed)
	}
	if conn.pingCount() < 1 {
		t.Fatal("no ping sent before timeout")
	}
}

// Cancellation (connection closed) stops the loop without timeout.
func TestHeartbeatStopsOnCancel(t *testing.T) {
	conn := &fakePingConn{}
	pong := make(chan struct{}, 1)
	done := make(chan struct{})

	timedOut := make(chan bool, 1)
	go func() { timedOut <- heartbeat(conn, 10*time.Millisecond, 30*time.Millisecond, pong, done) }()
	time.Sleep(15 * time.Millisecond) // let one ping fire
	close(done)

	select {
	case v := <-timedOut:
		if v {
			t.Fatal("cancel reported timeout")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("heartbeat did not stop on cancel")
	}
}

func TestHeartbeatConstants(t *testing.T) {
	if heartbeatInterval != 30*time.Second {
		t.Fatalf("heartbeatInterval = %v, want 30s (Research §2)", heartbeatInterval)
	}
	if pongWait != 10*time.Second {
		t.Fatalf("pongWait = %v, want 10s (Research §2)", pongWait)
	}
}
