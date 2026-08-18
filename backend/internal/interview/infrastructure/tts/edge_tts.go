package tts

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const edgeTTSURL = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1?TrustedClientToken=6A5AA1D4EAFF4E9FB37E23D68491D6F4"

type EdgeTTSClient struct {
	Voice string
}

func NewEdgeTTSClient(voice string) *EdgeTTSClient {
	if voice == "" {
		voice = "id-ID-ArdiNeural"
	}
	return &EdgeTTSClient{Voice: voice}
}

func (c *EdgeTTSClient) Synthesize(ctx context.Context, text string) ([]byte, error) {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, edgeTTSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Edge TTS: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Send config
	configMsg := fmt.Sprintf("X-Timestamp:%s\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n{\"context\":{\"synthesis\":{\"audio\":{\"metadataOptions\":{\"sentenceBoundaryEnabled\":\"false\",\"wordBoundaryEnabled\":\"true\"},\"outputFormat\":\"audio-24khz-48kbitrate-mono-mp3\"}}}}", time.Now().Format(time.RFC3339Nano))
	if err := conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Send SSML — the transcript is candidate/LLM text; XML-escape it so a
	// '<' or '&' cannot corrupt the SSML payload or inject markup.
	reqID := strings.ReplaceAll(uuid.New().String(), "-", "")
	ssml := fmt.Sprintf("<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='id-ID'><voice name='%s'><prosody rate='+0%%' pitch='+0%%'>%s</prosody></voice></speak>", c.Voice, html.EscapeString(text))

	ssmlMsg := fmt.Sprintf("X-RequestId:%s\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:%s\r\nPath:ssml\r\n\r\n%s", reqID, time.Now().Format(time.RFC3339Nano), ssml)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ssmlMsg)); err != nil {
		return nil, fmt.Errorf("failed to write ssml: %w", err)
	}

	var audioBuffer bytes.Buffer

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read error: %w", err)
		}

		if msgType == websocket.TextMessage {
			textMsg := string(msg)
			if strings.Contains(textMsg, "Path:turn.end") {
				break
			}
		} else if msgType == websocket.BinaryMessage {
			// Binary message contains headers followed by \r\n\r\n then the audio data
			separator := []byte("Path:audio\r\n")
			idx := bytes.Index(msg, separator)
			if idx != -1 {
				audioData := msg[idx+len(separator):]
				audioBuffer.Write(audioData)
			}
		}
	}

	return audioBuffer.Bytes(), nil
}
