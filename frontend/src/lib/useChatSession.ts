import { useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import { ChatClient, type ChatFrame, type PacingTelemetry } from "./ws"
import { getStoredInvitationToken } from "./interview-ticket"
import { api } from "./api"
import type { SandboxLanguage } from "@/types/api"

export interface ChatBubble {
  id: string
  kind: "question" | "answer" | "assistant" | "system" | "candidate_qa"
  content: string
  idx?: number
  isProbe?: boolean
  streaming?: boolean
  questionText?: string
  refused?: boolean
}

export interface CandidateQA {
  question: string
  answer: string
  refused: boolean
  answeredAt: string
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
  const submittingRef = useRef(false)
  const streamBufferRef = useRef("")
  // G2: per-attempt "socket reached OPEN" flag — the backend rejects an
  // expired ws_ticket with HTTP 401 BEFORE the upgrade, so a rejected
  // handshake is observable as a close that never saw onOpen.
  const openedRef = useRef(false)
  // G2: one re-mint per disconnect episode; re-armed on every successful open.
  const remintUsedRef = useRef(false)
  const mintingRef = useRef(false)
  // G11-StrictMode: refs survive the dev double-invoke (mount→cleanup→mount),
  // so these guards keep one ChatClient + one connection per hook instance —
  // exactly the Invite.tsx autoStartDoneRef pattern, applied to connectivity.
  const connectedRef = useRef(false)
  const clientKeyRef = useRef("")
  const effectTokenRef = useRef<symbol | null>(null)

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

  const [topicTurn, setTopicTurn] = useState(1)
  const [maxTopicTurns, setMaxTopicTurns] = useState(3)
  const [isTopicComplete, setIsTopicComplete] = useState(false)

  // J10 (B4): candidate Q&A — a sent candidate_question immediately gets a
  // streaming candidate_qa bubble (holds the question); the server answers
  // with qa_answer (refused=true on cap exhaustion).
  const [qaPending, setQaPending] = useState(false)
  // Ref mirror so the same-tick double-click guard never depends on a stale
  // state closure (Invite.tsx autoStartDoneRef pattern).
  const qaPendingRef = useRef(false)
  const qaPendingRefSetter = (v: boolean) => {
    qaPendingRef.current = v
    setQaPending(v)
  }

  useEffect(() => {
    disconnectedRef.current = disconnected
  }, [disconnected])

  useEffect(() => {
    if (!id || !ticket) {
      toast.error("Missing interview session credentials.")
      return
    }

    const key = `${id}::${ticket}`
    // Token per effect run: the deferred close compares against the CURRENT
    // token, so a StrictMode re-mount (setup→cleanup→setup, all synchronous)
    // cancels run 1's close, while a REAL unmount (last token stays) closes.
    const token = Symbol(key)
    effectTokenRef.current = token

    // Credential/interview changed — the previous client is stale and must be
    // closed NOW (a reconnect on another ticket must not outlive its effect).
    const prev = clientRef.current
    if (prev && clientKeyRef.current !== key) {
      connectedRef.current = false
      clientKeyRef.current = ""
      clientRef.current = null
      prev.close()
    }

    // StrictMode double-invoke of the SAME logical session: reuse the live
    // client + connection instead of opening a second socket.
    const reuse = clientRef.current != null && clientKeyRef.current === key && connectedRef.current
    if (!reuse) {
      const scheduleReconnect = () => {
        if (reconnectCountRef.current < MAX_RECONNECTS) {
          setReconnecting(true)
          const delay = Math.min(1000 * 2 ** reconnectCountRef.current, 10000)
          reconnectCountRef.current += 1
          reconnectTimerRef.current = setTimeout(() => {
            beginConnection()
          }, delay)
        } else {
          // Reconnect budget exhausted — enter a persistent disconnected state.
          // Answers are not being recorded; the UI must disable input and
          // offer a manual refresh (window.location.reload()) to resume.
          setReconnecting(false)
          setDisconnected(true)
        }
      }

      // Every connection attempt starts with a clean per-attempt open flag so
      // the close handler can classify "handshake rejected" reliably.
      const beginConnection = () => {
        openedRef.current = false
        if (!clientRef.current || !id) return
        clientRef.current.connect(id)
      }

      const client = new ChatClient({
        ticket,
        onOpen: () => {
          openedRef.current = true
          // G2: a successful handshake re-arms the single re-mint allowance —
          // the next disconnect is a new episode.
          remintUsedRef.current = false
          setReconnecting(false)
          // Replay the resume frame once the socket is actually OPEN —
          // session pinning never happened when sent during CONNECTING.
          if (sessionIdRef.current) {
            clientRef.current?.resume(sessionIdRef.current)
          }
        },
        onClose: () => {
          submittingRef.current = false
          if (evaluatedRef.current) return

        // G2: an unopened socket means the pre-upgrade ticket gate rejected
        // us — the original ws_ticket has most likely expired (10 min TTL vs
        // a 30 min interview). Re-mint ONCE with the long-lived invitation
        // token and retry with the fresh credential. Never loop.
        const handshakeRejected = !openedRef.current
        const invitationToken =
          handshakeRejected && !remintUsedRef.current && !mintingRef.current && id
            ? getStoredInvitationToken(id)
            : null
        if (invitationToken) {
          remintUsedRef.current = true
          mintingRef.current = true
          setReconnecting(true)
          void api
            .post<{ ticket: string }>(`/candidate/interviews/${id}/ticket`, {
              invitation_token: invitationToken,
            })
            .then((res) => {
              clientRef.current?.setTicket(res.ticket)
              beginConnection()
            })
            .catch(() => {
              // Mint failed (expired invite, network) — fall back to the
              // normal backoff budget; never retry the mint itself.
              scheduleReconnect()
            })
            .finally(() => {
              mintingRef.current = false
            })
          return
        }

        scheduleReconnect()
      },
      onFrame: (frame: ChatFrame) => {
        if (frame.type === "interview.start") {
          submittingRef.current = false
          setReconnecting(false)
          setDisconnected(false)
          // G3: a completed handshake + start/resume proves the credential
          // and transport are healthy again — the reconnect budget restarts.
          reconnectCountRef.current = 0
          sessionIdRef.current = frame.session_id
          setTotal(frame.total_questions)
          if (frame.session_budget_sec) {
            setSessionRemainingSec(frame.session_budget_sec)
          }
        } else if (frame.type === "question") {
          setReconnecting(false)
          setDisconnected(false)
          // G3: replayed questions after resume also prove session health.
          reconnectCountRef.current = 0
          setCurrentIdx(frame.idx)
          setCurrentQuestionText(frame.content)
          setTopicTurn(frame.topic_turn || 1)
          setMaxTopicTurns(frame.max_topic_turns || 3)
          setIsTopicComplete(false)
          if (frame.total_questions) {
            setTotal(frame.total_questions)
          } else {
            setTotal((prev) => Math.max(prev, frame.idx))
          }
          if (frame.time_limit_sec) setTimeLimitSec(frame.time_limit_sec)
          if (frame.archetype) setArchetype(frame.archetype)
          if (frame.session_remaining_sec !== undefined) setSessionRemainingSec(frame.session_remaining_sec)

          onQuestionRef.current?.()

          setStreaming(false)
          setPendingAnswer(false)
          submittingRef.current = false
          setBubbles((prev) => {
            // Deduplicate: If question is already present at end or same idx, don't duplicate on reconnect
            const last = prev[prev.length - 1]
            if (last && last.kind === "question" && (last.idx === frame.idx || last.content === frame.content)) {
              return prev
            }
            if (prev.some((b) => b.kind === "question" && b.idx === frame.idx && b.content === frame.content)) {
              return prev
            }
            return [
              ...prev,
              {
                id: crypto.randomUUID(),
                kind: "question",
                content: frame.content,
                idx: frame.idx,
                isProbe: frame.is_probe,
              },
            ]
          })
        } else if (frame.type === "token") {
          streamBufferRef.current += frame.content
          setStreaming(true)
        } else if (frame.type === "response") {
          setStreaming(false)
          setPendingAnswer(false)
          submittingRef.current = false
          if (frame.topic_turn) setTopicTurn(frame.topic_turn)
          if (frame.max_topic_turns) setMaxTopicTurns(frame.max_topic_turns)
          if (typeof frame.is_topic_complete === "boolean") setIsTopicComplete(frame.is_topic_complete)
          const responseContent = frame.content || streamBufferRef.current
          streamBufferRef.current = ""
          setBubbles((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.kind === "assistant") {
              return [...prev.slice(0, -1), { ...last, content: responseContent, streaming: false }]
            }
            return [...prev, { id: crypto.randomUUID(), kind: "assistant", content: responseContent, streaming: false }]
          })
        } else if (frame.type === "evaluation") {
          evaluatedRef.current = true
          setStreaming(false)
          submittingRef.current = false
          streamBufferRef.current = ""
          setEvaluation(frame)
        } else if (frame.type === "qa_answer") {
          // J10: the grounded answer for the pending candidate question. The
          // pair is persisted server-side regardless of refuse state; a
          // refused=true answer is the cap/expiry refusal copy.
          qaPendingRefSetter(false)
          setBubbles((prev) => {
            const last = prev[prev.length - 1]
            // The send callback already wrote the candidate_qa bubble, so the
            // answer just completes it (streaming=false → content lands).
            if (last && last.kind === "candidate_qa" && last.streaming) {
              return [
                ...prev.slice(0, -1),
                { ...last, content: frame.answer, streaming: false, refused: frame.refused === true },
              ]
            }
            // Fallback: complete the most recent candidate_qa bubble.
            for (let i = prev.length - 1; i >= 0; i -= 1) {
              const b = prev[i]
              if (b.kind === "candidate_qa") {
                return [
                  ...prev.slice(0, i),
                  { ...b, content: frame.answer, streaming: false, refused: frame.refused === true },
                  ...prev.slice(i + 1),
                ]
              }
            }
            return [...prev, { id: crypto.randomUUID(), kind: "candidate_qa", content: frame.answer, streaming: false, refused: frame.refused === true }]
          })
        } else if (frame.type === "error") {
          setStreaming(false)
          setPendingAnswer(false)
          submittingRef.current = false
          streamBufferRef.current = ""
          if (frame.code === "INTERVIEW_EXPIRED") {
            setExpired(true)
          }
          if (frame.code === "turn_in_progress") {
            // D19: the candidate answered while the interviewer was still
            // responding — a gentle, retryable notice, not a failure.
            toast.info("Interviewer is still responding: hold on a moment")
          } else {
            toast.error(frame.message || "An error occurred during the interview session.")
          }
        }
      },
    })

      clientRef.current = client
      clientKeyRef.current = key
      connectedRef.current = true
      beginConnection()
    }

    return () => {
      // StrictMode (dev): React runs setup→cleanup→setup synchronously, so
      // run 1's cleanup cannot yet see run 2's token — a synchronous compare
      // would close the socket run 2 is about to reuse. Defer one microtask:
      // if a re-setup replaced the token by then, the client is now owned by
      // run 2 and stays open (exactly one connection). A REAL unmount leaves
      // the last token in place, so the deferred close proceeds.
      queueMicrotask(() => {
        if (effectTokenRef.current !== token) return
        connectedRef.current = false
        clientKeyRef.current = ""
        if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
        clientRef.current?.close()
      })
    }
  }, [id, ticket])

  // Appends the candidate bubble, records pending state and sends over the
  // socket. Returns whether the frame was actually transmitted — consumers use
  // that to surface a "not sent" error without leaving the input disabled.
  const submitAnswer = useCallback((content: string, action: "reply" | "advance" = "reply", pacing?: PacingTelemetry): boolean => {
    if (disconnectedRef.current || submittingRef.current) return false
    submittingRef.current = true
    setPendingAnswer(true)
    setBubbles((prev) => [...prev, { id: crypto.randomUUID(), kind: "answer", content }])
    const sent = clientRef.current?.answer(content, action, pacing) ?? false
    if (!sent) {
      submittingRef.current = false
      setPendingAnswer(false)
    }
    return sent
  }, [])

  const interrupt = useCallback(() => {
    clientRef.current?.interrupt()
  }, [])

  // J10 (B4): submit a free-form candidate question. The question bubble is
  // rendered immediately (streaming) and completed by the qa_answer frame —
  // refused=true leaves the refusal copy in the transcript. Returns whether
  // the frame was transmitted so the UI can surface a lost connection.
  const askCandidateQuestion = useCallback((question: string): boolean => {
    if (disconnectedRef.current || qaPendingRef.current) return false
    const trimmed = question.trim()
    if (!trimmed) return false
    qaPendingRefSetter(true)
    setBubbles((prev) => [
      ...prev,
      {
        id: crypto.randomUUID(),
        kind: "candidate_qa",
        content: trimmed,
        questionText: trimmed,
        streaming: true,
      },
    ])
    const sent = clientRef.current?.sendCandidateQuestion(trimmed) ?? false
    if (!sent) {
      qaPendingRefSetter(false)
      setBubbles((prev) => {
        const last = prev[prev.length - 1]
        if (last && last.kind === "candidate_qa" && last.streaming) {
          return [...prev.slice(0, -1), { ...last, streaming: false, content: "Could not send your question: connection lost." }]
        }
        return prev
      })
    }
    return sent
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
    topicTurn,
    maxTopicTurns,
    isTopicComplete,
    evaluation,
    reconnecting,
    disconnected,
    expired,
    pendingAnswer,
    qaPending,
    submitAnswer,
    askCandidateQuestion,
    interrupt,
    sendCodeChange,
  }
}
