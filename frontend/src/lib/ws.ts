// Typed WebSocket client for the candidate chat (B3).
// Frames mirror the backend protocol (OpenAPI /candidate/interviews/{id}/chat).
// The browser WebSocket auto-answers server pings — no manual pong needed.

export interface PacingTelemetry {
  time_to_first_keystroke_ms?: number
  duration_ms?: number
  typed_chars?: number
  pasted_chars?: number
  pasted_ratio?: number
}

export type ChatFrame =
  | {
      type: "interview.start"
      session_id: string
      total_questions: number
      session_budget_sec?: number
    }
  | {
      type: "question"
      content: string
      idx: number
      archetype?: "conversational" | "system_design" | "coding"
      time_limit_sec?: number
      session_remaining_sec?: number
    }
  | { type: "token"; content: string }
  | { type: "response"; content: string }
  | {
      type: "evaluation"
      scores: Record<string, number>
      overall?: number
      recommendation?: string
      status: "complete" | "pending"
    }
  | { type: "error"; code?: string; message: string }
  | { type: "pong" }

export interface ChatClientOptions {
  ticket: string
  onFrame: (frame: ChatFrame) => void
  onClose: (reason: "closed" | "error" | "timeout") => void
  onOpen?: () => void
  sessionId?: string
}

// Absolute ws URL — the WebSocket constructor rejects relative paths.
// Through the Vite proxy the host stays :5173 and /api is forwarded with ws.
const WS_BASE = (() => {
  const base = import.meta.env.VITE_API_BASE ?? "/api/v1"
  if (base.startsWith("http")) return base.replace(/^http/, "ws")
  const proto = window.location.protocol === "https:" ? "wss" : "ws"
  return `${proto}://${window.location.host}${base}`
})()

export class ChatClient {
  private ws: WebSocket | null = null
  private closed = false
  private closeNotified = false
  private opts: ChatClientOptions

  constructor(opts: ChatClientOptions) {
    this.opts = opts
  }

  connect(interviewId: string): void {
    this.closed = false
    this.closeNotified = false
    // Browsers cannot set WS headers — the ticket rides the query param
    // (?ticket=), accepted by RequireTicket alongside the header form.
    const url = `${WS_BASE}/candidate/interviews/${interviewId}/chat?ticket=${encodeURIComponent(this.opts.ticket)}`
    const ws = new WebSocket(url)
    this.ws = ws

    ws.onopen = () => {
      // Only fires AFTER readyState is OPEN — the right moment to replay the
      // resume frame (sending during CONNECTING drops it silently).
      this.opts.onOpen?.()
    }

    ws.onmessage = (ev) => {
      try {
        this.opts.onFrame(JSON.parse(ev.data as string) as ChatFrame)
      } catch (err) {
        // Malformed frames must not kill the socket — log and drop.
        console.error("Ignoring malformed chat frame", err)
      }
    }
    ws.onclose = (ev) => {
      if (!this.closed && !this.closeNotified) {
        this.closeNotified = true
        this.opts.onClose(ev.wasClean ? "closed" : "error")
      }
    }
    // onerror fires BEFORE onclose for the same failure — notify only once
    // or reconnect logic would schedule twice.
    ws.onerror = () => {
      if (!this.closed && !this.closeNotified) {
        this.closeNotified = true
        this.opts.onClose("error")
      }
    }
  }

  isOpen(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  send(frame: Record<string, unknown>): boolean {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(frame))
      return true
    }
    return false
  }

  answer(content: string, pacing?: PacingTelemetry): boolean {
    return this.send({
      type: "answer",
      content,
      ...(pacing ? { pacing_telemetry: pacing } : {}),
    })
  }

  interrupt(): void {
    this.send({ type: "interrupt" })
  }

  resume(sessionId: string): void {
    this.send({ type: "resume", session_id: sessionId })
  }

  sendTelemetry(eventType: string, questionIdx?: number, details?: Record<string, unknown>): boolean {
    return this.send({
      type: "telemetry",
      event_type: eventType,
      timestamp: new Date().toISOString(),
      question_idx: questionIdx,
      details,
    })
  }

  sendCodeChange(language: string, code: string, questionIdx?: number): boolean {
    return this.send({
      type: "code.change",
      language,
      code,
      question_idx: questionIdx,
    })
  }

  close(): void {
    this.closed = true
    this.ws?.close()
  }
}
