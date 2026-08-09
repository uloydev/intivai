// Typed WebSocket client for the candidate chat (B3).
// Frames mirror the backend protocol (OpenAPI /candidate/interviews/{id}/chat).
// The browser WebSocket auto-answers server pings — no manual pong needed.

export type ChatFrame =
  | { type: "interview.start"; session_id: string; total_questions: number }
  | { type: "question"; content: string; idx: number }
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
  private opts: ChatClientOptions

  constructor(opts: ChatClientOptions) {
    this.opts = opts
  }

  connect(interviewId: string): void {
    this.closed = false
    // Browsers cannot set WS headers — the ticket rides the query param
    // (?ticket=), accepted by RequireTicket alongside the header form.
    const url = `${WS_BASE}/candidate/interviews/${interviewId}/chat?ticket=${encodeURIComponent(this.opts.ticket)}`
    const ws = new WebSocket(url)
    this.ws = ws

    ws.onmessage = (ev) => {
      try {
        this.opts.onFrame(JSON.parse(ev.data as string) as ChatFrame)
      } catch {
        // ignore malformed frames
      }
    }
    ws.onclose = (ev) => {
      if (!this.closed) this.opts.onClose(ev.wasClean ? "closed" : "error")
    }
    ws.onerror = () => {
      if (!this.closed) this.opts.onClose("error")
    }
  }

  send(frame: Record<string, unknown>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(frame))
    }
  }

  answer(content: string): void {
    this.send({ type: "answer", content })
  }

  interrupt(): void {
    this.send({ type: "interrupt" })
  }

  resume(sessionId: string): void {
    this.send({ type: "resume", session_id: sessionId })
  }

  close(): void {
    this.closed = true
    this.ws?.close()
  }
}
