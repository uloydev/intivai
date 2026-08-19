import { useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import { ChatClient, type ChatFrame, type PacingTelemetry } from "./ws"
import type { SandboxLanguage } from "@/types/api"

export interface ChatBubble {
  id: string
  kind: "question" | "answer" | "assistant" | "system"
  content: string
  idx?: number
  streaming?: boolean
}

const MAX_RECONNECTS = 5
const DEFAULT_SESSION_BUDGET_SEC = 1800
const DEFAULT_QUESTION_LIMIT_SEC = 180

interface UseChatSessionOptions {
  id?: string
  ticket: string
  // Fired when a fresh question frame arrives — consumers reset pacing/typing
  // telemetry here.
  onQuestion?: () => void
}

// Owns the ChatClient lifecycle: connect, resume replay, exponential backoff
// reconnect, and the frame → bubble reducer. Stable bubble ids (crypto.randomUUID
// at creation) let React keep list identity across streaming edits.
export function useChatSession({ id, ticket, onQuestion }: UseChatSessionOptions) {
  const clientRef = useRef<ChatClient | null>(null)
  const sessionIdRef = useRef<string>("")
  const reconnectCountRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const evaluatedRef = useRef(false)
  const onQuestionRef = useRef(onQuestion)
  onQuestionRef.current = onQuestion
  // Mirror of `disconnected` for the stable submitAnswer callback (state read
  // inside a []-dep callback would be a stale closure).
  const disconnectedRef = useRef(false)

  const [bubbles, setBubbles] = useState<ChatBubble[]>([])
  const [streaming, setStreaming] = useState(false)
  const [total, setTotal] = useState(0)
  const [currentIdx, setCurrentIdx] = useState(0)
  const [currentQuestionText, setCurrentQuestionText] = useState("")
  const [sessionRemainingSec, setSessionRemainingSec] = useState(DEFAULT_SESSION_BUDGET_SEC)
  const [timeLimitSec, setTimeLimitSec] = useState(DEFAULT_QUESTION_LIMIT_SEC)
  const [archetype, setArchetype] = useState<"conversational" | "system_design" | "coding">("conversational")
  const [evaluation, setEvaluation] = useState<Extract<ChatFrame, { type: "evaluation" }> | null>(null)
  const [reconnecting, setReconnecting] = useState(false)
  const [disconnected, setDisconnected] = useState(false)
  const [expired, setExpired] = useState(false)
  const [pendingAnswer, setPendingAnswer] = useState(false)

  useEffect(() => {
    disconnectedRef.current = disconnected
  }, [disconnected])

  useEffect(() => {
    if (!id || !ticket) {
      toast.error("Missing interview session credentials.")
      return
    }

    const client = new ChatClient({
      ticket,
      onOpen: () => {
        // Replay the resume frame once the socket is actually OPEN —
        // session pinning never happened when sent during CONNECTING.
        if (sessionIdRef.current) {
          clientRef.current?.resume(sessionIdRef.current)
        }
      },
      onClose: () => {
        if (evaluatedRef.current) return
        if (reconnectCountRef.current < MAX_RECONNECTS) {
          setReconnecting(true)
          const delay = Math.min(1000 * 2 ** reconnectCountRef.current, 10000)
          reconnectCountRef.current += 1
          reconnectTimerRef.current = setTimeout(() => {
            if (sessionIdRef.current && clientRef.current && id) {
              clientRef.current.connect(id)
            }
          }, delay)
        } else {
          // Reconnect budget exhausted — enter a persistent disconnected state.
          // Answers are not being recorded; the UI must disable input and
          // offer a manual refresh (window.location.reload()) to resume.
          setReconnecting(false)
          setDisconnected(true)
        }
      },
      onFrame: (frame: ChatFrame) => {
        if (frame.type === "interview.start") {
          setReconnecting(false)
          setDisconnected(false)
          sessionIdRef.current = frame.session_id
          setTotal(frame.total_questions)
          if (frame.session_budget_sec) {
            setSessionRemainingSec(frame.session_budget_sec)
          }
        } else if (frame.type === "question") {
          setReconnecting(false)
          setDisconnected(false)
          setCurrentIdx(frame.idx)
          setCurrentQuestionText(frame.content)
          if (frame.time_limit_sec) setTimeLimitSec(frame.time_limit_sec)
          if (frame.archetype) setArchetype(frame.archetype)
          if (frame.session_remaining_sec) setSessionRemainingSec(frame.session_remaining_sec)

          onQuestionRef.current?.()

          setStreaming(false)
          setPendingAnswer(false)
          setBubbles((prev) => {
            // Deduplicate: If question is already present at end or same idx, don't duplicate on reconnect
            const last = prev[prev.length - 1]
            if (last && last.kind === "question" && (last.idx === frame.idx || last.content === frame.content)) {
              return prev
            }
            if (prev.some((b) => b.kind === "question" && b.idx === frame.idx && b.content === frame.content)) {
              return prev
            }
            return [...prev, { id: crypto.randomUUID(), kind: "question", content: frame.content, idx: frame.idx }]
          })
        } else if (frame.type === "token") {
          setStreaming(true)
          setBubbles((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.kind === "assistant" && last.streaming) {
              return [
                ...prev.slice(0, -1),
                { ...last, content: last.content + frame.content },
              ]
            }
            return [...prev, { id: crypto.randomUUID(), kind: "assistant", content: frame.content, streaming: true }]
          })
        } else if (frame.type === "response") {
          setStreaming(false)
          setBubbles((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.kind === "assistant") {
              return [...prev.slice(0, -1), { ...last, content: frame.content, streaming: false }]
            }
            return [...prev, { id: crypto.randomUUID(), kind: "assistant", content: frame.content, streaming: false }]
          })
        } else if (frame.type === "evaluation") {
          evaluatedRef.current = true
          setStreaming(false)
          setEvaluation(frame)
        } else if (frame.type === "error") {
          setStreaming(false)
          setPendingAnswer(false)
          if (frame.code === "INTERVIEW_EXPIRED") {
            setExpired(true)
          }
          toast.error(frame.message || "An error occurred during the interview session.")
        }
      },
    })

    clientRef.current = client
    client.connect(id)

    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
      client.close()
    }
  }, [id, ticket])

  // Appends the candidate bubble, records pending state and sends over the
  // socket. Returns whether the frame was actually transmitted — consumers use
  // that to surface a "not sent" error without leaving the input disabled.
  const submitAnswer = useCallback((content: string, pacing?: PacingTelemetry): boolean => {
    if (disconnectedRef.current) return false
    setPendingAnswer(true)
    setBubbles((prev) => [...prev, { id: crypto.randomUUID(), kind: "answer", content }])
    const sent = clientRef.current?.answer(content, pacing) ?? false
    if (!sent) setPendingAnswer(false)
    return sent
  }, [])

  const interrupt = useCallback(() => {
    clientRef.current?.interrupt()
  }, [])

  const sendCodeChange = useCallback(
    (language: SandboxLanguage, code: string, questionIdx?: number): boolean =>
      clientRef.current?.sendCodeChange(language, code, questionIdx) ?? false,
    []
  )

  return {
    clientRef,
    bubbles,
    streaming,
    total,
    currentIdx,
    currentQuestionText,
    sessionRemainingSec,
    timeLimitSec,
    archetype,
    evaluation,
    reconnecting,
    disconnected,
    expired,
    pendingAnswer,
    submitAnswer,
    interrupt,
    sendCodeChange,
  }
}
