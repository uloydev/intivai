package api

import (
	"context"
	"errors"
	"testing"
	"time"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
)

// brokenJSONConn fails every write — simulates a transport that died under
// the writer goroutine's feet.
type brokenJSONConn struct{}

func (brokenJSONConn) WriteJSON(any) error { return errors.New("transport gone") }

// ivTokenFrame is a cheap placeholder outbound frame.
func ivTokenFrame() any {
	return ivdomain.TokenMessage{Type: ivdomain.MsgToken, Content: "t"}
}

// RED (D18): close() must cancel connCtx BEFORE taking the writer mutex.
// With the old order (mutex first; context cancellation owned by an outer
// defer that runs after close), senders parked on a full channel hold the
// mutex forever: their only wakeup is connCtx.Done, which cannot fire until
// close returns — a deadlock. The handler then never exits and the session
// lock strands until TTL.
func TestWSWriterCloseUnblocksParkedSendersAfterWriterDeath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := newWSWriter(brokenJSONConn{}, ctx, cancel)

	// The writer goroutine dies on its first (failing) write; saturate the
	// 64-slot buffer, then park a sender holding the mutex.
	for i := 0; i < 64; i++ {
		w.send(ivTokenFrame())
	}
	parked := make(chan struct{})
	go func() {
		close(parked)
		w.send(ivTokenFrame())
	}()
	<-parked
	time.Sleep(50 * time.Millisecond) // let the parked sender grab the mutex

	closed := make(chan struct{})
	go func() {
		w.close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("wsWriter.close deadlocked: parked sender holds the mutex and connCtx never fires")
	}
}
