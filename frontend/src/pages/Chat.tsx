import { useEffect, useMemo, useRef, useState } from "react"
import { useMutation } from "@tanstack/react-query"
import { Link, useParams, useSearchParams } from "react-router-dom"
import {
  PaperPlaneRight,
  Stop,
  User,
  CheckCircle,
  WarningCircle,
  ArrowClockwise,
  Target,
  ChatCircleDots,
  ArrowRight,
  UserFocus,
} from "@phosphor-icons/react"
import { Code2 } from "lucide-react"
import type { PacingTelemetry } from "@/lib/ws"
import { useChatSession } from "@/lib/useChatSession"
import { useProctoring } from "@/lib/useProctoring"
import { aiReview, runCode } from "@/lib/sandbox"
import { createTrailingDebounce } from "@/lib/debounce"
import { getStoredInvitationToken } from "@/lib/interview-ticket"
import { TimerGate } from "@/components/interview/TimerGate"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { CodingSandbox } from "@/components/sandbox/CodingSandbox"
import { Markdown } from "@/components/markdown/Markdown"
import { cn } from "@/lib/utils"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { SandboxLanguage, SandboxTestCase } from "@/types/api"

export function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const ticket = params.get("t") ?? ""

  const [input, setInput] = useState("")
  const [showSandbox, setShowSandbox] = useState(false)
  const [isInterrupting, setIsInterrupting] = useState(false)
  const pastedFlagRef = useRef(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const answerRef = useRef<HTMLTextAreaElement>(null)

  // J10 (B4): candidate free-form question — mirrored from the server-side
  // cap (1000 runes, backend/internal/interview/domain/interview.go).
  const [qaInput, setQaInput] = useState("")
  const [showQaInput, setShowQaInput] = useState(false)

  // Authenticity & Pacing Telemetry Refs
  const questionDisplayedAtRef = useRef<number>(Date.now())
  const firstKeystrokeAtRef = useRef<number | null>(null)
  const typedCharsCountRef = useRef<number>(0)
  const pastedCharsCountRef = useRef<number>(0)

  const resetPacing = () => {
    questionDisplayedAtRef.current = Date.now()
    firstKeystrokeAtRef.current = null
    typedCharsCountRef.current = 0
    pastedCharsCountRef.current = 0
    pastedFlagRef.current = false
  }

  const session = useChatSession({
    id,
    ticket,
    onQuestion: resetPacing,
  })

  // G11: code-change frames are debounced (300ms trailing) so keystrokes do
  // not flood the socket; submit flushes the latest editor state first.
  const debouncedCodeChangeRef = useRef(
    createTrailingDebounce((lang: SandboxLanguage, code: string) => {
      session.sendCodeChange(lang, code, currentIdxRef.current)
    }, 300),
  )
  useEffect(() => () => debouncedCodeChangeRef.current.cancel(), [])

  useEffect(() => {
    if (!session.streaming) setIsInterrupting(false)
  }, [session.streaming])

  const {
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
  } = session

  // Live question idx for the stable debounced sender callback.
  const currentIdxRef = useRef(currentIdx)
  currentIdxRef.current = currentIdx

  const { trackPaste } = useProctoring({
    interviewId: id,
    ticket,
    currentQuestionIdx: currentIdx,
    client: session.clientRef.current,
    active: !evaluation && !expired,
  })

  const reduceMotion = useMemo(
    () => window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    []
  )

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior: reduceMotion ? "auto" : "smooth",
      })
    }
  }, [bubbles, streaming, reduceMotion])

  // Refocus the answer box whenever a fresh question arrives so the candidate
  // can keep typing without reaching for the input again.
  useEffect(() => {
    if (!streaming && !pendingAnswer && !evaluation && !expired && !disconnected) {
      answerRef.current?.focus()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentIdx])

  const collectPacingTelemetry = (): PacingTelemetry => {
    const now = Date.now()
    const typed = typedCharsCountRef.current
    const pasted = pastedCharsCountRef.current
    const totalChars = typed + pasted
    return {
      time_to_first_keystroke_ms: firstKeystrokeAtRef.current
        ? firstKeystrokeAtRef.current - questionDisplayedAtRef.current
        : undefined,
      duration_ms: now - questionDisplayedAtRef.current,
      typed_chars: typed,
      pasted_chars: pasted,
      pasted_ratio: totalChars > 0 ? pasted / totalChars : 0,
    }
  }

  const sendWithAction = (action: "reply" | "advance") => {
    const trimmed = input.trim()
    if (!trimmed || streaming || pendingAnswer || !!evaluation || expired || disconnected) return
    // G11: the pending debounced code-change goes out BEFORE the answer so
    // the server snapshots the latest editor state for this question.
    debouncedCodeChangeRef.current.flush()
    const sent = session.submitAnswer(trimmed, action, collectPacingTelemetry())
    setInput("")
    if (!sent) {
      toast.error("Connection lost: your answer was not sent. Please try again.")
    }
  }

  const sendAnswer = () => sendWithAction("reply")
  const advanceTopic = () => sendWithAction("advance")

  // J10: submit the candidate question and surface a soft cap hint. The server
  // refuses further questions via a refused qa_answer frame (transcript copy).
  const sendCandidateQuestion = () => {
    const trimmed = qaInput.trim()
    if (!trimmed || qaPending || !!evaluation || expired || disconnected) return
    const sent = session.askCandidateQuestion(trimmed)
    if (sent) {
      setQaInput("")
      setShowQaInput(false)
    } else {
      toast.error("Connection lost: your question was not sent. Please try again.")
    }
  }

  const handleTimerExpire = () => {
    if (streaming || pendingAnswer || !!evaluation || expired || disconnected) return
    const trimmed = input.trim()
    const submissionText = trimmed.length > 0 ? trimmed : "My time for this question ran out: nothing was submitted."
    const action = isTopicComplete ? "advance" : "reply"
    // G11: honest toast — only claim auto-submission when it actually left.
    const sent = session.submitAnswer(submissionText, action, collectPacingTelemetry())
    setInput("")
    if (sent) {
      toast.info("Stage time limit elapsed. Response auto-submitted.")
    } else {
      toast.error("Stage time limit elapsed, but connection is down. Nothing was submitted. Reconnect and try again.")
    }
  }

  const handleExecuteSandbox = (language: SandboxLanguage, code: string, testCases: SandboxTestCase[]) =>
    runCode(language, code, testCases)

  const handleAIReview = (language: SandboxLanguage, code: string) =>
    aiReview(language, code, currentQuestionText || "Technical Interview Problem")

  const [showHumanRequest, setShowHumanRequest] = useState(false)
  const [humanRequested, setHumanRequested] = useState(false)
  const requestHumanMutation = useMutation({
    mutationFn: () => {
      // G11: the endpoint validates the INVITATION token, not the ws_ticket —
      // send the stored invitation_token under its own key.
      const invitationToken = id ? (getStoredInvitationToken(id) ?? "") : ""
      return api.post<unknown>(`/candidate/interviews/${id}/request-human`, {
        invitation_token: invitationToken,
      })
    },
    onSuccess: () => {
      setHumanRequested(true)
      setShowHumanRequest(false)
      toast.success("Your request has been noted. A team member will follow up.")
    },
    onError: () => toast.error("Failed to submit request"),
  })

  return (
    <div className="flex h-screen flex-col bg-background text-foreground selection:bg-primary/20 selection:text-primary">
      {/* Top Header Bar */}
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-border bg-card px-4 sm:px-6 z-10">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold font-display text-sm shadow-sm">
            I
          </div>
          <div>
            <h1 className="font-display font-bold text-sm tracking-tight flex items-center gap-2">
              Intivai Live Assessment
            </h1>
            <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <span
                className={cn(
                  "h-1.5 w-1.5 rounded-full",
                  disconnected
                    ? "bg-red-500"
                    : reconnecting
                    ? "bg-warning animate-pulse"
                    : "bg-success"
                )}
              />
              <span>{disconnected ? "Connection Lost" : reconnecting ? "Reconnecting…" : "Real-Time AI Session"}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Split Screen Code Sandbox Toggle */}
          <Button
            variant={showSandbox ? "secondary" : "outline"}
            size="sm"
            onClick={() => setShowSandbox(!showSandbox)}
            className="text-xs h-8 gap-1.5 border-border/70 shadow-sm"
          >
            <Code2 className="w-3.5 h-3.5 text-indigo-400" />
            <span>{showSandbox ? "Hide Code Sandbox" : "Code Sandbox"}</span>
          </Button>

          {total > 0 && (
            <div className="flex items-center gap-3">
              <div className="text-right">
                <span className="font-display text-xs font-bold text-foreground">
                  Question {currentIdx} of {Math.max(total, currentIdx)}
                </span>
                <div className="h-1.5 w-24 rounded-full bg-muted mt-1 overflow-hidden">
                  <div
                    className="h-full bg-primary transition-all duration-500"
                    style={{ width: `${Math.min(100, (currentIdx / Math.max(total, currentIdx)) * 100)}%` }}
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Notice Banners */}
      {expired && (
        <div className="bg-destructive/10 border-b border-destructive/20 px-4 py-2.5 text-center text-xs font-medium text-destructive flex items-center justify-center gap-2">
          <WarningCircle className="h-4 w-4" weight="fill" />
          Session ticket expired: please reopen your candidate invite link to resume.
        </div>
      )}
      {reconnecting && (
        <div className="bg-warning/10 border-b border-warning/20 px-4 py-2 text-center text-xs font-medium text-warning flex items-center justify-center gap-2 animate-pulse">
          <ArrowClockwise className="h-4 w-4 animate-spin" /> Connection lost: resuming session…
        </div>
      )}
      {disconnected && (
        <div
          role="alert"
          className="bg-destructive/10 border-b border-destructive/20 px-4 py-2.5 text-center text-xs font-medium text-destructive flex flex-col sm:flex-row items-center justify-center gap-2"
        >
          <WarningCircle className="h-4 w-4" weight="fill" />
          <span>Connection lost: answers are not being recorded.</span>
          <Button
            variant="outline"
            size="sm"
            className="h-7 text-[11px] text-destructive border-destructive/30 hover:bg-destructive/10 ml-1"
            onClick={() => window.location.reload()}
          >
            Refresh to resume
          </Button>
        </div>
      )}

      {/* Stage Timer Gate & Assessment Progress Bar */}
      {!evaluation && !expired && currentIdx > 0 && (
        <div className="border-b border-border bg-card px-4 py-2.5">
          <div className="max-w-7xl mx-auto">
            <TimerGate
              sessionRemainingSec={sessionRemainingSec}
              timeLimitSec={timeLimitSec}
              currentIdx={currentIdx}
              total={Math.max(total, currentIdx)}
              archetype={archetype}
              active={!evaluation && !expired}
              isProcessing={streaming || pendingAnswer}
              onExpire={handleTimerExpire}
            />
          </div>
        </div>
      )}

      {/* Main Workspace Body: Single or Split View */}
      <div className={cn(
        "flex-1 flex min-h-0 overflow-hidden",
        showSandbox ? "flex-col lg:flex-row" : "flex-col"
      )}>
        {/* Chat Conversation Column */}
        <div
          className={cn(
            "relative flex flex-col h-full overflow-hidden transition-all duration-300",
            showSandbox ? "lg:w-[45%] lg:border-r lg:border-border border-b lg:border-b-0" : "w-full max-w-4xl mx-auto"
          )}
        >
          {/* Active Question Context Pill (if active) */}
          {currentQuestionText && !evaluation && (
            <div className="border-b border-primary/20 bg-primary/5 px-4 py-2.5 flex items-center justify-between text-xs shrink-0 shadow-xs">
              <div className="flex items-center gap-2 overflow-hidden flex-1 mr-2">
                <Badge variant="outline" className="border-primary/40 text-primary bg-primary/10 font-mono text-[10px] shrink-0 font-bold">
                  Q{currentIdx}
                </Badge>
                <span className="text-foreground/90 truncate font-medium text-xs">
                  {currentQuestionText}
                </span>
              </div>
              <Badge variant="secondary" className="text-[10px] bg-success/10 text-success font-semibold uppercase tracking-wider shrink-0 border border-success/20">
                Active Challenge
              </Badge>
            </div>
          )}

          {/* Chat Log Window */}
          <div
            ref={scrollRef}
            role="log"
            aria-live="polite"
            aria-label="Interview transcript"
            className="flex-1 space-y-5 overflow-y-auto p-4 sm:p-6"
          >
            {bubbles.length === 0 && !streaming && (
              <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <Target className="h-6 w-6" weight="bold" />
                </div>
                <p className="font-display font-semibold text-sm">Connecting to Interviewer...</p>
                <p className="text-xs text-muted-foreground max-w-xs">
                  Formulating competence probing questions tailored to your application.
                </p>
              </div>
            )}

            {bubbles.map((b) => (
              <div
                key={b.id}
                  className={cn(
                    "flex gap-3 text-xs sm:text-sm leading-relaxed animate-in fade-in duration-300",
                    b.kind === "answer" && "justify-end",
                    b.kind === "candidate_qa" && "justify-end",
                    b.kind === "question" && "justify-start",
                    b.kind === "assistant" && "justify-start",
                    b.kind === "system" && "justify-center"
                  )}
              >
                {/* AI Avatar for Assistant & Question */}
                {(b.kind === "question" || b.kind === "assistant") && (
                  <div className={cn(
                    "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg font-bold shadow-sm",
                    b.kind === "question" ? "bg-primary text-primary-foreground" : "bg-muted text-foreground border border-border"
                  )}>
                    {b.kind === "question" ? <Target className="h-4 w-4" weight="bold" /> : <ChatCircleDots className="h-4 w-4" weight="bold" />}
                  </div>
                )}

                {/* Question Message Card */}
                {b.kind === "question" && (
                  <div className={cn(
                    "max-w-[85%] rounded-2xl border bg-card p-4 space-y-2 shadow-sm",
                    b.isProbe ? "border-dashed border-warning/40" : "border-primary/30"
                  )}>
                    <div className="flex items-center justify-between border-b border-border/40 pb-1.5">
                      <span className="font-display font-bold text-xs text-primary flex items-center gap-1.5">
                        <Target className="h-3.5 w-3.5" weight="bold" />
                        {b.isProbe ? (
                          <span>Adaptive Follow-Up Probe ({b.idx || currentIdx} of {Math.max(total, b.idx || currentIdx)})</span>
                        ) : (
                          <span>Question {b.idx || currentIdx} {total > 0 ? `of ${Math.max(total, b.idx || currentIdx)}` : ""}</span>
                        )}
                      </span>
                      <Badge variant="outline" className="text-[10px] text-muted-foreground border-border/60">
                        {b.isProbe ? "Follow-Up Probe" : "Technical Challenge"}
                      </Badge>
                    </div>
                    <Markdown content={b.content} />
                  </div>
                )}

                {/* Assistant Commentary / Follow-up Bubble */}
                {b.kind === "assistant" && (
                  <div className="max-w-[85%] rounded-2xl rounded-tl-sm bg-muted/40 border border-border/50 p-3.5 space-y-1.5 text-foreground shadow-sm">
                    <div className="flex items-center gap-1.5 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                      <ChatCircleDots className="h-3.5 w-3.5 text-primary" weight="fill" />
                      <span>Interviewer Follow-Up & Context</span>
                    </div>
                    <Markdown content={b.content} />
                    {b.streaming && (
                      <span className="inline-flex gap-1 ml-1.5 align-middle">
                        <span className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce [animation-delay:-0.3s]" />
                        <span className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce [animation-delay:-0.15s]" />
                        <span className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce" />
                      </span>
                    )}
                  </div>
                )}

                {/* System Feedback Bubble */}
                {b.kind === "system" && (
                  <div className="rounded-xl border border-border/60 bg-muted/30 px-3.5 py-1.5 text-[11px] text-muted-foreground font-medium">
                    {b.content}
                  </div>
                )}

                {/* Candidate Answer Bubble */}
                {b.kind === "answer" && (
                  <div className="max-w-[85%] rounded-2xl rounded-tr-sm bg-primary text-primary-foreground p-3.5 text-xs sm:text-sm shadow-sm whitespace-pre-wrap leading-relaxed">
                    {b.content}
                  </div>
                )}

                {/* Candidate Free-Form Question & Grounded Answer (J10) */}
                {b.kind === "candidate_qa" && (
                  <div className="max-w-[85%] space-y-2 min-w-0">
                    <div className="flex justify-end">
                      <div className="rounded-2xl rounded-tr-sm bg-primary/15 border border-primary/30 p-3.5 text-xs sm:text-sm text-foreground whitespace-pre-wrap leading-relaxed max-w-full">
                        {b.questionText ?? b.content}
                      </div>
                    </div>
                    {b.streaming ? (
                      <div className="flex justify-start">
                        <div className="flex items-center gap-1.5 rounded-2xl rounded-tl-sm bg-muted/40 border border-border/50 px-4 py-2.5 text-[11px] text-muted-foreground">
                          <ArrowClockwise className="h-3.5 w-3.5 animate-spin" />
                          Grounding answer from your company context…
                        </div>
                      </div>
                    ) : (
                      <div className="flex justify-start items-end gap-2">
                        <div
                          className={cn(
                            "rounded-2xl rounded-tl-sm bg-muted/40 border p-3.5 text-xs sm:text-sm text-foreground whitespace-pre-wrap leading-relaxed max-w-full",
                            b.refused ? "border-warning/40 bg-warning/5" : "border-border/50"
                          )}
                        >
                          <div className="flex items-center gap-1.5 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-1">
                            <ChatCircleDots className="h-3.5 w-3.5 text-primary" weight="fill" />
                            {b.refused ? "Question limit reached" : "Grounded Answer"}
                          </div>
                          <Markdown content={b.content} />
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* Candidate Avatar */}
                {b.kind === "answer" && (
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground border border-border">
                    <User className="h-4 w-4" />
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Scroll-to-bottom FAB */}
          <button
            type="button"
            onClick={() => scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" })}
            className="absolute bottom-20 right-4 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-opacity hover:opacity-90 md:hidden"
            aria-label="Scroll to bottom"
          >
            <ArrowRight className="h-4 w-4 rotate-90" />
          </button>

          {/* Bottom Bar: Evaluation or Answer Input */}
          {evaluation ? (
            <div className="border-t border-border bg-card p-5 space-y-3">
              <div className="flex items-center gap-2 text-success font-display font-bold text-sm">
                <CheckCircle className="h-5 w-5" weight="fill" />
                <span>Interview Complete</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Your interview is complete. Our team will review the results and reach out within a few business days.
              </p>
              {evaluation.overall !== undefined && evaluation.overall !== null && (
                <div className="rounded-xl border border-border/50 bg-muted/30 p-3 text-xs text-muted-foreground flex items-center justify-between">
                  <span>Session evaluation score</span>
                  <Badge variant="secondary" className="font-mono font-semibold text-foreground">
                    {Math.round(evaluation.overall)}/100
                  </Badge>
                </div>
              )}
              <Button asChild size="sm" className="shadow-sm">
                <Link to="/candidate/portal">
                  Track your application in the Candidate Portal
                </Link>
              </Button>
            </div>
          ) : (
            <div className="border-t border-border bg-card p-3.5 sticky bottom-0 z-10">
              {/* Turn progress indicator */}
              {!isTopicComplete && currentIdx > 0 && (
                <div className="flex items-center justify-between mb-2 px-0.5">
                  <span className="text-[11px] text-muted-foreground font-medium">
                    Topic Discussion: Turn {topicTurn} of {maxTopicTurns}
                  </span>
                  <div className="flex gap-1">
                    {Array.from({ length: maxTopicTurns }).map((_, i) => (
                      <div
                        key={i}
                        className={cn(
                          "h-1.5 w-5 rounded-full transition-all",
                          i < topicTurn ? "bg-primary" : "bg-muted"
                        )}
                      />
                    ))}
                  </div>
                </div>
              )}
              <Label htmlFor="chat-input" className="sr-only">
                Your answer
              </Label>
              <div className="flex items-end gap-2.5">
                <Textarea
                  id="chat-input"
                  ref={answerRef}
                  value={input}
                  maxLength={4000}
                  onChange={(e) => {
                    const next = e.target.value
                    if (firstKeystrokeAtRef.current === null) {
                      firstKeystrokeAtRef.current = Date.now()
                    }
                    // Count only real keystrokes: the onChange fired right
                    // after a paste already counted the full pasted length.
                    if (pastedFlagRef.current) {
                      pastedFlagRef.current = false
                    } else if (next.length > input.length) {
                      typedCharsCountRef.current += next.length - input.length
                    }
                    setInput(next)
                  }}
                  onPaste={(e) => {
                    const text = e.clipboardData.getData("text")
                    if (text) {
                      pastedCharsCountRef.current += text.length
                      pastedFlagRef.current = true
                      trackPaste(text.length)
                    }
                  }}
                  onKeyDown={(e) => {
                    if (e.nativeEvent.isComposing) return
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault()
                      sendAnswer()
                    }
                  }}
                  placeholder={
                    streaming
                      ? "AI Interviewer is responding…"
                      : !isTopicComplete && topicTurn < maxTopicTurns
                      ? `Ask a clarification or continue your answer... (Enter to reply, Shift+Enter for new line)`
                      : `Type your answer to Question ${currentIdx || 1}... (Enter to submit)`
                  }
                  rows={2}
                  disabled={streaming || pendingAnswer || !!evaluation || expired || disconnected}
                  className="min-h-[48px] resize-none bg-card rounded-xl border-border/60 text-xs sm:text-sm p-2.5 focus-visible:ring-primary"
                />
                <div className="flex flex-col gap-1.5 shrink-0">
                  {streaming ? (
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-[48px] w-[48px] rounded-xl border-destructive/30 text-destructive hover:bg-destructive/10"
                      title="Skip AI speech / Advance immediately"
                      aria-label="Stop response"
                      onClick={() => {
                        if (isInterrupting) return
                        setIsInterrupting(true)
                        session.interrupt()
                      }}
                      disabled={isInterrupting}
                    >
                      <Stop className="h-5 w-5" weight="fill" />
                    </Button>
                  ) : (
                    <>
                      {/* Primary: Send Reply / Clarification (in-topic dialogue) */}
                      <Button
                        size="icon"
                        className="h-[48px] w-[48px] rounded-lg shadow-sm"
                        title="Send reply / clarify (Enter)"
                        aria-label="Send reply"
                        onClick={sendAnswer}
                        disabled={!input.trim() || pendingAnswer || expired || disconnected}
                      >
                        <PaperPlaneRight className="h-5 w-5" weight="bold" />
                      </Button>
                      {/* Secondary: Complete topic & advance (only visible if in multi-turn mode) */}
                      {!isTopicComplete && currentIdx > 0 && (
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-[48px] px-3 rounded-lg border-success/40 text-success hover:bg-success/10 text-xs font-semibold leading-tight shrink-0"
                          title="Complete this topic & advance to next question"
                          aria-label="Complete topic and advance"
                          onClick={advanceTopic}
                          disabled={!input.trim() || pendingAnswer || expired || disconnected}
                        >
                          Next Topic
                        </Button>
                      )}
                    </>
                  )}
                </div>
              </div>
              {/* Hint text */}
              {!streaming && !isTopicComplete && currentIdx > 0 && (
                <p className="text-[10px] text-muted-foreground mt-1.5 px-0.5">
                  <span className="font-medium text-foreground/60">↵ Reply / Clarify</span>
                  {" · "}
                  <span className="font-medium text-success/80">Next Topic: Complete &amp; advance</span>
                </p>
              )}
              {/* Request Human Interviewer */}
              {!humanRequested && !expired && (
                <button
                  type="button"
                  onClick={() => setShowHumanRequest(true)}
                  className="text-[10px] text-muted-foreground hover:text-foreground mt-1.5 px-0.5 underline-offset-2 hover:underline flex items-center gap-1 transition-colors"
                >
                  <UserFocus className="h-3 w-3" /> Prefer a human interviewer?
                </button>
              )}
              {humanRequested && (
                <p className="text-[10px] text-success mt-1.5 px-0.5 flex items-center gap-1">
                  <CheckCircle className="h-3 w-3" /> Human interviewer requested: a team member will follow up.
                </p>
              )}
              {/* Ask a Question (J10) */}
              {!evaluation && !expired && !disconnected && (
                <div className="mt-2 border-t border-border/40 pt-2">
                  {showQaInput ? (
                    <div className="space-y-1.5">
                      <div className="flex items-end gap-2">
                        <Textarea
                          id="candidate-question-input"
                          value={qaInput}
                          maxLength={1000}
                          onChange={(e) => setQaInput(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.nativeEvent.isComposing) return
                            if (e.key === "Enter" && !e.shiftKey) {
                              e.preventDefault()
                              sendCandidateQuestion()
                            }
                          }}
                          placeholder="Ask about the role, team, or process… (max 1000 characters)"
                          rows={2}
                          disabled={qaPending || streaming}
                          className="min-h-[48px] resize-none bg-card rounded-xl border-border/60 text-xs sm:text-sm p-2.5 focus-visible:ring-primary"
                        />
                        <Button
                          size="icon"
                          className="h-[48px] w-[48px] rounded-lg shrink-0 shadow-sm"
                          title="Send question (Enter)"
                          aria-label="Send question"
                          onClick={sendCandidateQuestion}
                          disabled={!qaInput.trim() || qaPending || streaming}
                        >
                          <PaperPlaneRight className="h-5 w-5" weight="bold" />
                        </Button>
                      </div>
                      <div className="flex items-center justify-between px-0.5">
                        <span className="text-[10px] text-muted-foreground mr-2">
                          {qaPending ? "Waiting for an answer…" : "Your question is answered from our company information only."}
                        </span>
                        <span className={cn("text-[10px] font-mono", qaInput.length > 1000 ? "text-destructive" : "text-muted-foreground")}>
                          {qaInput.length}/1000
                        </span>
                      </div>
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setShowQaInput(true)}
                      className="text-[10px] text-muted-foreground hover:text-foreground mt-1.5 px-0.5 underline-offset-2 hover:underline flex items-center gap-1 transition-colors"
                    >
                      <ChatCircleDots className="h-3 w-3" /> Ask a question about the role
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right Split Column: Live Coding Sandbox */}
        {showSandbox && (
          <div className="lg:w-[55%] h-[40vh] lg:h-full overflow-hidden flex flex-col bg-neutral-950">
            <CodingSandbox
              questionIdx={currentIdx}
              onExecute={handleExecuteSandbox}
              onRequestAIReview={handleAIReview}
              onCodeChange={(lang, code) => {
                debouncedCodeChangeRef.current(lang, code)
              }}
            />
          </div>
        )}
      </div>

      {/* Request Human Interviewer Confirmation Modal */}
      {showHumanRequest && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="bg-card border border-border rounded-2xl p-6 max-w-sm w-full space-y-4 shadow-xl">
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                <UserFocus className="h-5 w-5 text-primary" />
              </div>
              <div>
                <h3 className="font-semibold text-foreground">Request Human Interviewer</h3>
                <p className="text-xs text-muted-foreground">The AI interview will be paused.</p>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              A team member will follow up with you. Your progress so far will be saved.
            </p>
            <div className="flex gap-2">
              <Button
                onClick={() => requestHumanMutation.mutate()}
                disabled={requestHumanMutation.isPending}
              >
                {requestHumanMutation.isPending ? "Submitting..." : "Confirm Request"}
              </Button>
              <Button variant="outline" onClick={() => setShowHumanRequest(false)}>
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
