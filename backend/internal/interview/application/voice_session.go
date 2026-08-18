package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/interview/infrastructure/stt"
	"github.com/intivai/backend/internal/interview/infrastructure/tts"
	"github.com/intivai/backend/internal/interview/infrastructure/webrtc"
	"github.com/intivai/backend/pkg/metrics"
	"github.com/rs/zerolog/log"
)

type VoiceSession struct {
	ID          uuid.UUID
	InterviewID uuid.UUID
	PC          *webrtc.PeerConnection
	VAD         *webrtc.VAD
	STT         *stt.WhisperClient
	TTS         *tts.EdgeTTSClient

	// Audio buffer for VAD
	audioBuffer  []float32
	mu           sync.Mutex
	isSpeaking   bool
	silenceStart time.Time

	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	started   bool

	// OnAudio delivers synthesized speech (raw bytes) to the WS writer so the
	// client can play it — the MVP demo path until proper Opus/RTP encoding
	// lands (Phase 5, deferred).
	OnAudio func(audioBytes []byte)
}

func NewVoiceSession(interviewID uuid.UUID, pc *webrtc.PeerConnection) (*VoiceSession, error) {
	vad, err := webrtc.NewVAD("") // Uses default path
	if err != nil {
		return nil, fmt.Errorf("failed to init vad: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	session := &VoiceSession{
		ID:          uuid.New(),
		InterviewID: interviewID,
		PC:          pc,
		VAD:         vad,
		STT:         stt.NewWhisperClient(""),
		TTS:         tts.NewEdgeTTSClient("id-ID-ArdiNeural"),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Route incoming audio to our buffer
	pc.OnAudioData = session.handleIncomingAudio

	return session, nil
}

func (s *VoiceSession) Start() {
	s.startOnce.Do(func() {
		s.started = true
		metrics.WSActiveConnections.WithLabelValues("voice").Inc()
		log.Info().Str("session_id", s.ID.String()).Msg("Voice session started")

		// Send initial greeting (mock LLM response for now)
		go s.speak("Halo! Selamat datang di sesi wawancara. Silakan perkenalkan diri Anda.")
	})
}

func (s *VoiceSession) Stop() {
	if s.started {
		metrics.WSActiveConnections.WithLabelValues("voice").Dec()
	}
	log.Info().Str("session_id", s.ID.String()).Msg("Stopping voice session")
	s.cancel()
	_ = s.PC.Close()
}

func (s *VoiceSession) handleIncomingAudio(rtpPayload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In a real implementation, you MUST decode Opus (RTP payload) to 16kHz PCM float32.
	// Since Opus decoding requires CGO (e.g. ghr.io/hraban/opus) or complex pure-go implementations,
	// this MVP buffers raw bytes to demonstrate the pipeline structure.
	// We convert raw bytes to float32 just to satisfy the VAD signature for this demonstration.
	pcmFloat := make([]float32, len(rtpPayload))
	for i, b := range rtpPayload {
		pcmFloat[i] = float32(b) / 255.0
	}

	s.audioBuffer = append(s.audioBuffer, pcmFloat...)

	// Process in chunks of 512 samples for VAD
	chunkSize := 512
	if len(s.audioBuffer) >= chunkSize {
		chunk := s.audioBuffer[:chunkSize]
		s.audioBuffer = s.audioBuffer[chunkSize:]

		isSpeech, err := s.VAD.Process(chunk)
		if err != nil {
			log.Error().Err(err).Msg("VAD process error")
			return
		}

		if isSpeech {
			s.isSpeaking = true
			s.silenceStart = time.Time{}
		} else if s.isSpeaking {
			if s.silenceStart.IsZero() {
				s.silenceStart = time.Now()
			} else if time.Since(s.silenceStart) > 1500*time.Millisecond {
				// 1.5 seconds of silence -> trigger STT
				s.isSpeaking = false
				s.silenceStart = time.Time{}
				go s.processUtterance()
			}
		}
	}
}

func (s *VoiceSession) processUtterance() {
	log.Info().Str("session_id", s.ID.String()).Msg("Processing utterance (STT)")

	// MVP: Mocking the STT call since we bypassed proper Opus decoding.
	// In production, send the decoded PCM buffer to Whisper.
	// text, err := s.STT.Transcribe(s.ctx, pcmBytes, 16000)
	text := "Ini adalah teks hasil transkripsi (mock)."

	log.Info().Str("transcription", text).Msg("STT Result")

	// Send text to LLM (mocked here)
	llmResponse := "Terima kasih atas jawabannya. Pertanyaan selanjutnya: apa motivasi Anda melamar?"

	// Speak response
	s.speak(llmResponse)
}

func (s *VoiceSession) speak(text string) {
	log.Info().Str("session_id", s.ID.String()).Str("text", text).Msg("Synthesizing speech")

	audioBytes, err := s.TTS.Synthesize(s.ctx, text)
	if err != nil {
		log.Error().Err(err).Msg("TTS Synthesis error")
		return
	}

	// MVP demo path: hand the audio to the WS writer (base64 "audio" frame)
	// — the client plays it locally. Real Opus-over-RTP encoding is Phase 5
	// (deferred until a paying customer).
	if s.OnAudio != nil {
		s.OnAudio(audioBytes)
	}
	log.Info().Int("bytes", len(audioBytes)).Msg("TTS synthesized successfully")
}
