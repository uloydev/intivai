package webrtc

// VAD provides a simple energy-based Voice Activity Detection
// fallback since silero-vad requires CGO and ONNX Runtime headers
// which breaks local development compilation without C++ dependencies.
type VAD struct {
	energyThreshold float32
}

func NewVAD(modelPath string) (*VAD, error) {
	// Fallback to simple energy thresholding (0.01 is arbitrary)
	return &VAD{
		energyThreshold: 0.01,
	}, nil
}

// Process returns true if speech is detected in the 16kHz float32 audio chunk
func (v *VAD) Process(audioData []float32) (bool, error) {
	var energy float32
	for _, sample := range audioData {
		energy += sample * sample
	}
	energy = energy / float32(len(audioData))

	return energy > v.energyThreshold, nil
}

func (v *VAD) Reset() {
	// No-op for energy VAD
}
