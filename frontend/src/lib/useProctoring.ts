import { useEffect, useRef, useCallback } from "react"
import { api } from "./api"
import type { ChatClient } from "./ws"

export interface UseProctoringOptions {
  interviewId?: string
  ticket?: string
  invitationToken?: string
  currentQuestionIdx?: number
  client?: ChatClient | null
  active?: boolean
}

export function useProctoring({
  interviewId,
  ticket,
  invitationToken,
  currentQuestionIdx = 1,
  client,
  active = true,
}: UseProctoringOptions) {
  const lastBlurTimeRef = useRef<number | null>(null)
  const questionDispatchTimeRef = useRef<number>(Date.now())

  // Update question timer when question index changes
  useEffect(() => {
    questionDispatchTimeRef.current = Date.now()
  }, [currentQuestionIdx])

  const dispatchEvent = useCallback(
    (eventType: string, details?: Record<string, unknown>) => {
      if (!interviewId || !active) return

      const payload = {
        event_type: eventType,
        timestamp: new Date().toISOString(),
        question_idx: currentQuestionIdx,
        details: details ?? {},
      }

      // 1. Try sending over active WebSocket
      if (client && client.isOpen()) {
        client.sendTelemetry(eventType, currentQuestionIdx, details)
        return
      }

      // 2. Fallback to REST API
      const token = ticket || invitationToken
      if (token) {
        api
          .post(`/candidate/interviews/${interviewId}/telemetry`, {
            ...payload,
            ticket: ticket || undefined,
            invitation_token: invitationToken || undefined,
          })
          .catch(() => {
            // Silently ignore network telemetry drops
          })
      }
    },
    [interviewId, ticket, invitationToken, currentQuestionIdx, client, active]
  )

  const trackPaste = useCallback(
    (charCount: number) => {
      const elapsedSec = (Date.now() - questionDispatchTimeRef.current) / 1000
      const isRapid = elapsedSec < 3.0
      dispatchEvent("clipboard_paste", {
        char_count: charCount,
        elapsed_sec_since_question: Math.round(elapsedSec * 10) / 10,
        is_rapid_paste: isRapid,
      })
    },
    [dispatchEvent]
  )

  const trackAudioAnomaly = useCallback(
    (anomalyType: string) => {
      dispatchEvent("audio_anomaly", {
        anomaly: anomalyType,
      })
    },
    [dispatchEvent]
  )

  useEffect(() => {
    if (!active || !interviewId) return

    const handleVisibilityChange = () => {
      if (document.hidden) {
        lastBlurTimeRef.current = Date.now()
        dispatchEvent("tab_switch", { source: "visibility_hidden" })
      } else {
        const awayMs = lastBlurTimeRef.current ? Date.now() - lastBlurTimeRef.current : 0
        dispatchEvent("focus_regained", {
          away_duration_sec: Math.round(awayMs / 1000),
        })
        lastBlurTimeRef.current = null
      }
    }

    const handleBlur = () => {
      if (!document.hidden && !lastBlurTimeRef.current) {
        lastBlurTimeRef.current = Date.now()
        dispatchEvent("focus_lost", { source: "window_blur" })
      }
    }

    const handleFocus = () => {
      if (lastBlurTimeRef.current) {
        const awayMs = Date.now() - lastBlurTimeRef.current
        dispatchEvent("focus_regained", {
          away_duration_sec: Math.round(awayMs / 1000),
        })
        lastBlurTimeRef.current = null
      }
    }

    // Resize storms (drag-resize, mobile rotation) must not flood the WS/API
    // with one event per frame — debounce with a 1s trailing edge.
    let resizeTimer: ReturnType<typeof setTimeout> | null = null
    const handleResize = () => {
      if (resizeTimer) clearTimeout(resizeTimer)
      resizeTimer = setTimeout(() => {
        dispatchEvent("window_resize", {
          width: window.innerWidth,
          height: window.innerHeight,
        })
        resizeTimer = null
      }, 1000)
    }

    document.addEventListener("visibilitychange", handleVisibilityChange)
    window.addEventListener("blur", handleBlur)
    window.addEventListener("focus", handleFocus)
    window.addEventListener("resize", handleResize)

    return () => {
      if (resizeTimer) clearTimeout(resizeTimer)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      window.removeEventListener("blur", handleBlur)
      window.removeEventListener("focus", handleFocus)
      window.removeEventListener("resize", handleResize)
    }
  }, [active, interviewId, dispatchEvent])

  return {
    trackPaste,
    trackAudioAnomaly,
    dispatchEvent,
  }
}
