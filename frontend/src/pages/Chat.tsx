import { useEffect, useMemo, useRef, useState } from "react"
import { Link, useParams, useSearchParams } from "react-router-dom"
import {
  Sparkle,
  PaperPlaneRight,
  Stop,
  Robot,
  User,
  CheckCircle,
  WarningCircle,
  ArrowClockwise,
  Target,
  ChatCircleDots,
} from "@phosphor-icons/react"
import { Code2 } from "lucide-react"
import type { PacingTelemetry } from "@/lib/ws"
import { useChatSession } from "@/lib/useChatSession"
import { useProctoring } from "@/lib/useProctoring"
import { aiReview, runCode } from "@/lib/sandbox"
import { TimerGate } from "@/components/interview/TimerGate"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { CodingSandbox } from "@/components/sandbox/CodingSandbox"
import { Markdown } from "@/components/markdown/Markdown"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import type { SandboxLanguage, SandboxTestCase } from "@/types/api"

export function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const ticket = params.get("t") ?? ""

  const [input, setInput] = useState("")
  const [showSandbox, setShowSandbox] = useState(false)
  const pastedFlagRef = useRef(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const answerRef = useRef<HTMLTextAreaElement>(null)

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

  const { bubbles, streaming, total, currentIdx, currentQuestionText, sessionRemainingSec, timeLimitSec, archetype, evaluation, reconnecting, disconnected, expired, pendingAnswer } = session

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

  const sendAnswer = () => {
    const trimmed = input.trim()
    if (!trimmed || streaming || pendingAnswer || !!evaluation || expired || disconnected) return
    const sent = session.submitAnswer(trimmed, collectPacingTelemetry())
    setInput("")
    if (!sent) {
      // Socket not open (reconnect window) — never leave the input disabled
      // with a silently dropped answer.
      toast.error("Connection lost — your answer was not sent. Please try again.")
    }
  }

  const handleTimerExpire = () => {
    if (streaming || pendingAnswer || !!evaluation || expired || disconnected) return
    const trimmed = input.trim()
    const submissionText = trimmed.length > 0 ? trimmed : "My time for this question ran out — nothing was submitted."
    session.submitAnswer(submissionText, collectPacingTelemetry())
    setInput("")
    toast.info("Stage time limit elapsed. Response auto-submitted.")
  }

  const handleExecuteSandbox = (language: SandboxLanguage, code: string, testCases: SandboxTestCase[]) =>
    runCode(language, code, testCases)

  const handleAIReview = (language: SandboxLanguage, code: string) =>
    aiReview(language, code, currentQuestionText || "Technical Interview Problem")

  return (
    <div className="flex h-screen flex-col bg-background text-foreground selection:bg-primary/20 selection:text-primary">
      {/* Top Header Bar */}
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-border/60 bg-background/80 px-4 sm:px-6 backdrop-blur-xl z-10">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary text-primary-foreground font-bold font-display text-sm shadow-md shadow-primary/25">
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
                    ? "bg-amber-400 animate-pulse"
                    : "bg-emerald-500"
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
                  Question {currentIdx} of {total}
                </span>
                <div className="h-1.5 w-24 rounded-full bg-muted mt-1 overflow-hidden">
                  <div
                    className="h-full bg-primary transition-all duration-500"
                    style={{ width: `${(currentIdx / total) * 100}%` }}
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
          Session ticket expired — please reopen your candidate invite link to resume.
        </div>
      )}
      {reconnecting && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 px-4 py-2 text-center text-xs font-medium text-amber-600 dark:text-amber-400 flex items-center justify-center gap-2 animate-pulse">
          <ArrowClockwise className="h-4 w-4 animate-spin" /> Connection lost — resuming session…
        </div>
      )}
      {disconnected && (
        <div
          role="alert"
          className="bg-destructive/10 border-b border-destructive/20 px-4 py-2.5 text-center text-xs font-medium text-destructive flex flex-col sm:flex-row items-center justify-center gap-2"
        >
          <WarningCircle className="h-4 w-4" weight="fill" />
          <span>Connection lost — answers are not being recorded.</span>
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
        <div className="border-b border-border/70 bg-card/60 px-4 py-2.5 backdrop-blur-md">
          <div className="max-w-7xl mx-auto">
            <TimerGate
              sessionRemainingSec={sessionRemainingSec}
              timeLimitSec={timeLimitSec}
              currentIdx={currentIdx}
              total={total}
              archetype={archetype}
              active={!evaluation && !expired}
              isProcessing={streaming || pendingAnswer}
              onExpire={handleTimerExpire}
            />
          </div>
        </div>
      )}

      {/* Main Workspace Body: Single or Split View */}
      <div className="flex-1 flex min-h-0 overflow-hidden">
        {/* Chat Conversation Column */}
        <div
          className={cn(
            "flex flex-col h-full overflow-hidden transition-all duration-300",
            showSandbox ? "w-[45%] border-r border-border" : "w-full max-w-4xl mx-auto"
          )}
        >
          {/* Active Question Context Pill (if active) */}
          {currentQuestionText && !evaluation && (
            <div className="border-b border-border/50 bg-muted/20 px-4 py-2 flex items-center justify-between text-xs shrink-0">
              <div className="flex items-center gap-2 overflow-hidden">
                <Badge variant="outline" className="border-primary/30 text-primary bg-primary/5 font-mono text-[10px] shrink-0">
                  Target Q{currentIdx}
                </Badge>
                <span className="text-muted-foreground truncate font-medium">
                  {currentQuestionText}
                </span>
              </div>
              <span className="text-[10px] text-emerald-500 font-semibold uppercase tracking-wider shrink-0 ml-2">
                Active Problem
              </span>
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
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary animate-pulse">
                  <Sparkle className="h-6 w-6" weight="fill" />
                </div>
                <p className="font-display font-semibold text-sm">Connecting to AI Interviewer...</p>
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
                  b.kind === "question" && "justify-start",
                  b.kind === "assistant" && "justify-start",
                  b.kind === "system" && "justify-center"
                )}
              >
                {/* AI Avatar for Assistant & Question */}
                {(b.kind === "question" || b.kind === "assistant") && (
                  <div className={cn(
                    "flex h-8 w-8 shrink-0 items-center justify-center rounded-xl font-bold shadow-md",
                    b.kind === "question" ? "bg-primary text-primary-foreground shadow-primary/20" : "bg-muted text-foreground border border-border"
                  )}>
                    {b.kind === "question" ? <Target className="h-4 w-4" weight="bold" /> : <Robot className="h-4 w-4" weight="fill" />}
                  </div>
                )}

                {/* Question Message Card */}
                {b.kind === "question" && (
                  <div className="max-w-[85%] rounded-2xl border border-primary/30 bg-card p-4 space-y-2 shadow-sm">
                    <div className="flex items-center justify-between border-b border-border/40 pb-1.5">
                      <span className="font-display font-bold text-xs text-primary flex items-center gap-1.5">
                        <Sparkle className="h-3.5 w-3.5" weight="fill" />
                        Question {b.idx || currentIdx} {total > 0 ? `of ${total}` : ""}
                      </span>
                      <Badge variant="outline" className="text-[10px] text-muted-foreground border-border/60">
                        Technical Challenge
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
                  <div className="max-w-[85%] rounded-2xl rounded-tr-sm bg-primary text-primary-foreground p-3.5 text-xs sm:text-sm shadow-md shadow-primary/10 whitespace-pre-wrap leading-relaxed">
                    {b.content}
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

          {/* Bottom Bar: Evaluation or Answer Input */}
          {evaluation ? (
            <div className="border-t border-border/80 bg-card/70 backdrop-blur-xl p-5 space-y-3">
              <div className="flex items-center gap-2 text-emerald-500 font-display font-bold text-sm">
                <CheckCircle className="h-5 w-5" weight="fill" />
                <span>Interview Complete</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Your interview is complete — our team will review the results and reach out within a few business days.
              </p>
              {evaluation.overall !== undefined && evaluation.overall !== null && (
                <div className="rounded-xl border border-border/50 bg-muted/30 p-3 text-xs text-muted-foreground flex items-center justify-between">
                  <span>Session evaluation score</span>
                  <Badge variant="secondary" className="font-mono font-semibold text-foreground">
                    {Math.round(evaluation.overall)}/100
                  </Badge>
                </div>
              )}
              <Button asChild variant="gradient" size="sm" className="shadow-md shadow-primary/20">
                <Link to="/candidate/portal">
                  Track your application in the Candidate Portal →
                </Link>
              </Button>
            </div>
          ) : (
            <div className="border-t border-border/60 bg-background/80 backdrop-blur-xl p-3.5">
              <Label htmlFor="chat-input" className="sr-only">
                Your answer
              </Label>
              <div className="flex items-end gap-2.5">
                <Textarea
                  id="chat-input"
                  ref={answerRef}
                  value={input}
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
                      ? "AI Interviewer is formulating feedback & question..."
                      : `Type your answer to Question ${currentIdx || 1}... (Press Enter to submit, Shift+Enter for new line)`
                  }
                  rows={2}
                  disabled={streaming || pendingAnswer || !!evaluation || expired || disconnected}
                  className="min-h-[48px] resize-none bg-card rounded-xl border-border/60 text-xs sm:text-sm p-2.5 focus-visible:ring-primary"
                />
                {streaming ? (
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-[48px] w-[48px] rounded-xl border-destructive/30 text-destructive hover:bg-destructive/10 shrink-0"
                    title="Skip AI speech / Advance immediately"
                    aria-label="Stop response"
                    onClick={() => session.interrupt()}
                  >
                    <Stop className="h-5 w-5" weight="fill" />
                  </Button>
                ) : (
                  <Button
                    variant="gradient"
                    size="icon"
                    className="h-[48px] w-[48px] rounded-xl shadow-md shadow-primary/20 shrink-0"
                    title="Submit answer (Enter)"
                    onClick={sendAnswer}
                    disabled={!input.trim() || pendingAnswer || expired || disconnected}
                  >
                    <PaperPlaneRight className="h-5 w-5" weight="bold" />
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Right Split Column: Live Coding Sandbox */}
        {showSandbox && (
          <div className="w-[55%] h-full overflow-hidden flex flex-col bg-neutral-950">
            <CodingSandbox
              questionIdx={currentIdx}
              onExecute={handleExecuteSandbox}
              onRequestAIReview={handleAIReview}
              onCodeChange={(lang, code) => {
                session.sendCodeChange(lang, code, currentIdx)
              }}
            />
          </div>
        )}
      </div>
    </div>
  )
}
