package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamapp "github.com/intivai/backend/internal/iam/application"
	ivapp "github.com/intivai/backend/internal/interview/application"
	"github.com/intivai/backend/internal/interview/infrastructure/webrtc"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

// RegisterVoiceRoutes — WS /api/v1/interviews/:id/voice. Gated the same way
// as the chat socket: a ws_ticket (candidate) or a recruiter auth token, an
// Origin allowlist, and the shared SessionRegistry so voice and chat cannot
// both hold the interview at once.
func (h *ChatHandler) RegisterVoiceRoutes(router fiber.Router, origins []string) {
	router.Use("/interviews/:id/voice", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	router.Get("/interviews/:id/voice", h.RequireVoiceAuth, fiberws.New(h.handleVoiceSession, fiberws.Config{Origins: origins}))
}

// RequireVoiceAuth accepts either a ws_ticket bound to this interview
// (candidate) or a recruiter auth token (voice room).
func (h *ChatHandler) RequireVoiceAuth(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	token := ""
	if strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimPrefix(header, "Bearer ")
	} else if q := c.Query("ticket"); q != "" {
		token = q
	}
	if token == "" {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "missing ws ticket or auth token"))
	}
	claims, err := h.tokens.Parse(token)
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid token"))
	}
	switch claims.Type {
	case iamapp.TokenTypeWSTicket:
		if claims.Extra["interview_id"] != c.Params("id") {
			return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "ticket not bound to this interview"))
		}
	case iamapp.TokenTypeAuth:
		// Recruiter voice room — the token's org must own the interview.
		interviewID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid interview id"))
		}
		if err := h.svc.VerifyInterviewOrg(c.UserContext(), claims.OrgID, interviewID); err != nil {
			return httpapi.Error(c, sharederr.NewDomainError("FORBIDDEN", "interview not accessible in this org"))
		}
	default:
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "invalid token type"))
	}
	c.Locals("ws_claims", claims)
	return c.Next()
}

func (h *ChatHandler) handleVoiceSession(c *fiberws.Conn) {
	claims, _ := c.Locals("ws_claims").(*iamapp.Claims)
	if claims == nil {
		_ = c.Close()
		return
	}
	interviewIDParam := c.Params("id")
	interviewID, err := uuid.Parse(interviewIDParam)
	if err != nil {
		h.log.Error().Err(err).Msg("Invalid interview ID format")
		_ = c.Close()
		return
	}

	sessionID, _ := claims.Extra["session_id"].(string)
	if sessionID == "" {
		sessionID = uuid.NewString() // fallback for recruiter tokens
	}

	connCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ok, err := h.sessions.TryAcquire(connCtx, interviewID.String(), sessionID)
	if err != nil || !ok {
		_ = c.WriteJSON(webrtc.SignalingMessage{Type: "error", Data: "interview already active on another connection"})
		_ = c.Close()
		return
	}
	defer func() {
		_ = h.sessions.Release(context.Background(), interviewID.String(), sessionID)
	}()

	h.log.Info().Str("interview_id", interviewID.String()).Msg("New voice WebSocket connection")

	pc, err := webrtc.NewPeerConnection()
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create PeerConnection")
		_ = c.Close()
		return
	}

	session, err := ivapp.NewVoiceSession(interviewID, pc)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create voice session")
		_ = pc.Close()
		_ = c.Close()
		return
	}

	// Deliver synthesized speech as base64 "audio" frames (MVP demo path;
	// proper Opus/RTP is deferred Phase 5). writeMu serializes ALL socket
	// writes — OnAudio fires from the speak goroutine while the read loop
	// below writes answers/errors; concurrent WS writes corrupt the stream.
	var writeMu sync.Mutex
	writeJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return c.WriteJSON(v)
	}
	session.OnAudio = func(audioBytes []byte) {
		frame := webrtc.SignalingMessage{Type: "audio", Data: base64.StdEncoding.EncodeToString(audioBytes)}
		if err := writeJSON(frame); err != nil {
			h.log.Warn().Err(err).Msg("Failed to deliver audio frame")
		}
	}

	defer session.Stop()

	// Start reading signaling messages from WebSocket
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			h.log.Info().Err(err).Msg("WebSocket closed")
			break
		}

		if messageType == fiberws.TextMessage {
			var sigMsg webrtc.SignalingMessage
			if err := json.Unmarshal(message, &sigMsg); err != nil {
				h.log.Error().Err(err).Msg("Failed to parse signaling message")
				continue
			}

			switch sigMsg.Type {
			case "offer":
				// The FE sends {type:"offer", sdp:...} — the standard WebRTC
				// signaling shape (was: reading sigMsg.Data, never populated).
				answerJSON, err := pc.HandleOffer(sigMsg.SDP)
				if err != nil {
					h.log.Error().Err(err).Msg("Failed to handle offer")
					continue
				}

				// Send answer back on the same sdp field.
				resp := webrtc.SignalingMessage{
					Type: "answer",
					SDP:  answerJSON,
				}
				if err := writeJSON(resp); err != nil {
					h.log.Error().Err(err).Msg("Failed to write answer to websocket")
				}

				// After SDP exchange, we can start the session logic (send greeting)
				session.Start()

			case "candidate":
				// {type:"candidate", data:<candidate JSON>}
				if err := pc.AddICECandidate(sigMsg.Data); err != nil {
					h.log.Error().Err(err).Msg("Failed to add ICE candidate")
				}
			}
		}
	}
}
