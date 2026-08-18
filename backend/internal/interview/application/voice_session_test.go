package application

import (
	"testing"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/interview/infrastructure/webrtc"
)

func TestVoiceSessionLifecycle(t *testing.T) {
	pc, err := webrtc.NewPeerConnection()
	if err != nil {
		t.Skipf("skipping WebRTC test if peer connection init fails: %v", err)
	}

	session, err := NewVoiceSession(uuid.New(), pc)
	if err != nil {
		t.Fatalf("failed to create voice session: %v", err)
	}

	if session.ID == uuid.Nil {
		t.Fatal("expected valid session ID")
	}

	// Test audio buffering
	rawBytes := []byte{0x10, 0x20, 0x30, 0x40}
	session.handleIncomingAudio(rawBytes)

	session.mu.Lock()
	bufLen := len(session.audioBuffer)
	session.mu.Unlock()

	if bufLen != 4 {
		t.Fatalf("expected 4 audio samples in buffer, got %d", bufLen)
	}

	session.Stop()
}
