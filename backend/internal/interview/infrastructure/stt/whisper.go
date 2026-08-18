package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const whisperTimeout = 15 * time.Second

type WhisperClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewWhisperClient(baseURL string) *WhisperClient {
	if baseURL == "" {
		baseURL = "http://whisper:8080"
	}
	return &WhisperClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: whisperTimeout},
	}
}

// TranscribeResponse is the OpenAI-compatible response.
type TranscribeResponse struct {
	Text string `json:"text"`
}

// Transcribe takes a PCM buffer, wraps it in a WAV container, and sends it to the STT API.
func (c *WhisperClient) Transcribe(ctx context.Context, pcmData []byte, sampleRate int) (string, error) {
	// Create WAV headers for 16-bit PCM, mono
	wavBuffer := new(bytes.Buffer)
	writeWavHeader(wavBuffer, uint32(sampleRate), uint16(1), uint16(16), uint32(len(pcmData)))
	wavBuffer.Write(pcmData)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, wavBuffer); err != nil {
		return "", fmt.Errorf("failed to copy audio data: %w", err)
	}

	_ = writer.WriteField("model", "tiny")
	_ = writer.WriteField("response_format", "json")

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/audio/transcriptions", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to whisper server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tr TranscribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("failed to decode whisper response: %w", err)
	}

	return tr.Text, nil
}

func writeWavHeader(out *bytes.Buffer, sampleRate uint32, numChannels uint16, bitsPerSample uint16, dataSize uint32) {
	out.WriteString("RIFF")
	out.Write(toLittleEndian(dataSize+36, 4))
	out.WriteString("WAVE")
	out.WriteString("fmt ")
	out.Write(toLittleEndian(16, 4)) // Subchunk1Size (16 for PCM)
	out.Write(toLittleEndian(1, 2))  // AudioFormat (1 for PCM)
	out.Write(toLittleEndian(uint32(numChannels), 2))
	out.Write(toLittleEndian(sampleRate, 4))

	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample/8)
	out.Write(toLittleEndian(byteRate, 4))

	blockAlign := numChannels * (bitsPerSample / 8)
	out.Write(toLittleEndian(uint32(blockAlign), 2))
	out.Write(toLittleEndian(uint32(bitsPerSample), 2))

	out.WriteString("data")
	out.Write(toLittleEndian(dataSize, 4))
}

func toLittleEndian(val uint32, bytes int) []byte {
	buf := make([]byte, bytes)
	for i := 0; i < bytes; i++ {
		buf[i] = byte(val >> (8 * i))
	}
	return buf
}
