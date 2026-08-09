import { useEffect, useRef, useState } from "react"
import { useParams, useSearchParams } from "react-router-dom"
import { ChatClient, type ChatFrame } from "@/lib/ws"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

interface Bubble {
  kind: "question" | "answer" | "assistant" | "system"
  content: string
  streaming?: boolean
}

export function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const ticket = params.get("t") ?? ""

  const clientRef = useRef<ChatClient | null>(null)
  const [bubbles, setBubbles] = useState<Bubble[]>([])
  const [input, setInput] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [total, setTotal] = useState(0)
  const [currentIdx, setCurrentIdx] = useState(0)
  const [evaluation, setEvaluation] = useState<Extract<ChatFrame, { type: "evaluation" }> | null>(null)
  const [reconnecting, setReconnecting] = useState(false)
  const sessionIdRef = useRef<string>("")
  const currentQuestionRef = useRef<string>("")
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!id || !ticket) return
    const client = new ChatClient({
      ticket,
      sessionId: sessionIdRef.current,
      onFrame: (frame) => handleFrame(frame),
      onClose: (reason) => {
        if (reason === "error") {
          setReconnecting(true)
          // Auto-resume: same ticket stays valid; server re-sends start + question.
          setTimeout(() => {
            if (!clientRef.current) return
            clientRef.current.connect(id)
            if (sessionIdRef.current) clientRef.current.resume(sessionIdRef.current)
          }, 1500)
        }
      },
    })
    clientRef.current = client
    client.connect(id)
    return () => client.close()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, ticket])

  function handleFrame(frame: ChatFrame) {
    switch (frame.type) {
      case "interview.start":
        sessionIdRef.current = frame.session_id
        setTotal(frame.total_questions)
        break
      case "question":
        currentQuestionRef.current = frame.content
        setCurrentIdx(frame.idx)
        setReconnecting(false)
        setStreaming(false)
        setBubbles((prev) => [...prev, { kind: "question", content: frame.content }])
        break
      case "token":
        setStreaming(true)
        setBubbles((prev) => {
          const next = [...prev]
          const last = next[next.length - 1]
          if (last && last.kind === "assistant" && last.streaming) {
            last.content += frame.content
            next[next.length - 1] = { ...last }
            return next
          }
          return [...next, { kind: "assistant", content: frame.content, streaming: true }]
        })
        break
      case "response":
        setStreaming(false)
        setBubbles((prev) => {
          const next = [...prev]
          const last = next[next.length - 1]
          if (last && last.kind === "assistant" && last.streaming) {
            next[next.length - 1] = { ...last, streaming: false }
          } else if (last && last.kind === "assistant") {
            next[next.length - 1] = { ...last, content: frame.content }
          }
          return next
        })
        break
      case "evaluation":
        setEvaluation(frame)
        setStreaming(false)
        break
      case "error":
        setStreaming(false)
        if (frame.code === "CONSENT_REQUIRED") {
          setBubbles((prev) => [...prev, { kind: "system", content: "Consent is required — go back to the invite page." }])
        } else {
          setBubbles((prev) => [...prev, { kind: "system", content: frame.message }])
        }
        break
      case "pong":
        break
    }
  }

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" })
  }, [bubbles])

  function sendAnswer() {
    const content = input.trim()
    if (!content || streaming) return
    setInput("")
    setBubbles((prev) => [...prev, { kind: "answer", content }])
    clientRef.current?.answer(content)
  }

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-[720px] flex-col bg-background">
      <header className="flex h-14 items-center justify-between border-b border-border bg-card px-4">
        <span className="font-display font-semibold text-primary">Intivai</span>
        {total > 0 && (
          <span className="text-sm text-muted-foreground">
            Question {currentIdx} of {total}
          </span>
        )}
      </header>

      {reconnecting && (
        <div className="bg-amber-100 px-4 py-2 text-center text-sm text-amber-800 dark:bg-amber-950 dark:text-amber-200">
          Connection lost — reconnecting…
        </div>
      )}

      <div ref={scrollRef} className="flex-1 space-y-4 overflow-y-auto p-4">
        {bubbles.length === 0 && !streaming && (
          <p className="pt-16 text-center text-sm text-muted-foreground">Connecting…</p>
        )}
        {bubbles.map((b, i) => (
          <div
            key={i}
            className={cn(
              "max-w-[85%] rounded-lg px-3 py-2 text-sm leading-relaxed",
              b.kind === "answer" && "ml-auto bg-accent text-accent-foreground",
              b.kind === "question" && "bg-primary text-primary-foreground",
              b.kind === "assistant" && "bg-muted text-foreground",
              b.kind === "system" && "mx-auto bg-amber-100 text-center text-amber-800 dark:bg-amber-950 dark:text-amber-200",
            )}
          >
            {b.kind === "assistant" && b.streaming && <span className="animate-pulse">▍</span>}
            {b.content}
          </div>
        ))}
      </div>

      {evaluation ? (
        <div className="border-t border-border bg-card p-4">
          <h2 className="font-display text-lg">Interview complete</h2>
          {evaluation.status === "complete" ? (
            <p className="mt-1 text-sm text-muted-foreground">
              Overall score {evaluation.overall}. Recommendation: {evaluation.recommendation ?? "—"}. The recruiter
              will be in touch — you can download your own transcript from this session.
            </p>
          ) : (
            <p className="mt-1 text-sm text-muted-foreground">Your report is being prepared — the recruiter will share the outcome.</p>
          )}
        </div>
      ) : (
        <div className="border-t border-border bg-card p-3">
          <div className="flex items-end gap-2">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault()
                  sendAnswer()
                }
              }}
              placeholder="Type your answer… (Enter to send)"
              rows={2}
              disabled={streaming || !!evaluation}
              className="min-h-[48px] resize-none"
            />
            {streaming && (
              <Button variant="outline" size="icon" aria-label="Stop response" onClick={() => clientRef.current?.interrupt()}>
                ■
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
