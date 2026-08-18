package webrtc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"
)

const iceGatherTimeout = 10 * time.Second

type SignalingMessage struct {
	Type string `json:"type"`           // "offer", "answer", "candidate", "audio", "error"
	Data string `json:"data,omitempty"` // opaque payload (ICE candidate, base64 audio)
	SDP  string `json:"sdp,omitempty"`  // SDP for offer/answer — the FE speaks this field
}

// PeerConnection wraps pion/webrtc logic for voice interviews.
type PeerConnection struct {
	PC          *webrtc.PeerConnection
	AudioTrack  *webrtc.TrackLocalStaticSample
	OnAudioData func([]byte)
}

func NewPeerConnection() (*PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create a local track for TTS audio playback to the browser
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	_, err = pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	wrapper := &PeerConnection{
		PC:         pc,
		AudioTrack: audioTrack,
	}

	// Setup incoming track handler (Browser Mic -> Backend)
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			// Read RTP packets from the incoming track
			for {
				rtp, _, readErr := track.ReadRTP()
				if readErr != nil {
					return
				}

				// In a full implementation, you would decode the Opus RTP payload to PCM here
				// before sending it to the VAD. For now, we pass the raw payload.
				if wrapper.OnAudioData != nil {
					wrapper.OnAudioData(rtp.Payload)
				}
			}
		}
	})

	return wrapper, nil
}

func (p *PeerConnection) HandleOffer(offerJSON string) (string, error) {
	var offer webrtc.SessionDescription
	if err := json.Unmarshal([]byte(offerJSON), &offer); err != nil {
		return "", err
	}

	if err := p.PC.SetRemoteDescription(offer); err != nil {
		return "", err
	}

	answer, err := p.PC.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	gatherComplete := webrtc.GatheringCompletePromise(p.PC)

	if err := p.PC.SetLocalDescription(answer); err != nil {
		return "", err
	}

	// Wait for ICE gathering to complete — bounded: a stuck STUN round must
	// not block the WS read loop forever (HandleOffer runs on it).
	select {
	case <-gatherComplete:
	case <-time.After(iceGatherTimeout):
	}

	answerBytes, err := json.Marshal(p.PC.LocalDescription())
	if err != nil {
		return "", err
	}

	return string(answerBytes), nil
}

func (p *PeerConnection) AddICECandidate(candidateJSON string) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(candidateJSON), &candidate); err != nil {
		return err
	}
	return p.PC.AddICECandidate(candidate)
}

func (p *PeerConnection) Close() error {
	return p.PC.Close()
}
