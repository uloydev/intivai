import { useEffect, useState, useRef, useMemo } from "react"
import { Clock, Hourglass, Warning, Code, ChatCircleText, GitFork } from "@phosphor-icons/react"
import { cn } from "@/lib/utils"

export interface TimerGateProps {
  sessionRemainingSec: number
  timeLimitSec: number
  currentIdx: number
  total: number
  archetype?: "conversational" | "system_design" | "coding"
  active: boolean
  onExpire: () => void
}

const GRACE_PERIOD_SEC = 15

export function TimerGate({
  sessionRemainingSec,
  timeLimitSec,
  currentIdx,
  total,
  archetype = "conversational",
  active,
  onExpire,
}: TimerGateProps) {
  const [sessionClock, setSessionClock] = useState(sessionRemainingSec || 1800)
  const [questionClock, setQuestionClock] = useState(timeLimitSec || 180)
  const [graceClock, setGraceClock] = useState<number | null>(null)
  const onExpireRef = useRef(onExpire)
  const expiredFiredRef = useRef(false)

  // Keep the expire handler fresh — a stale closure (first render's
  // handleTimerExpire) would auto-submit an empty input.
  onExpireRef.current = onExpire

  // Sync initial/updated props from server frames
  useEffect(() => {
    if (sessionRemainingSec > 0) {
      setSessionClock(sessionRemainingSec)
    }
  }, [sessionRemainingSec])

  useEffect(() => {
    setQuestionClock(timeLimitSec || 180)
    setGraceClock(null)
    expiredFiredRef.current = false
  }, [currentIdx, timeLimitSec])

  // Global Session Countdown Timer
  useEffect(() => {
    if (!active || sessionClock <= 0) return

    const interval = setInterval(() => {
      setSessionClock((prev) => {
        if (prev <= 1) {
          clearInterval(interval)
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(interval)
  }, [active, sessionClock])

  // Question Stage Countdown
  useEffect(() => {
    if (!active) return

    const interval = setInterval(() => {
      setQuestionClock((prev) => (prev > 0 ? prev - 1 : prev))
    }, 1000)

    return () => clearInterval(interval)
  }, [active])

  // When the question clock hits 0, start the grace period (once per stage)
  useEffect(() => {
    if (!active || questionClock > 0 || graceClock !== null) return
    setGraceClock(GRACE_PERIOD_SEC)
  }, [active, questionClock, graceClock])

  // Grace countdown — separate ticker so the banner starts at a full 15s
  // instead of being decremented on the same tick that starts it.
  useEffect(() => {
    if (!active || graceClock === null) return

    const interval = setInterval(() => {
      setGraceClock((prev) => (prev === null || prev <= 0 ? prev : prev - 1))
    }, 1000)

    return () => clearInterval(interval)
  }, [active, graceClock])

  // Clean trigger when grace period expires
  useEffect(() => {
    if (graceClock === 0 && !expiredFiredRef.current) {
      expiredFiredRef.current = true
      onExpireRef.current()
    }
  }, [graceClock])

  // Formatting helpers
  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const initialLimit = timeLimitSec || 180
  const questionPercent = useMemo(() => {
    return Math.max(0, Math.min(100, (questionClock / initialLimit) * 100))
  }, [questionClock, initialLimit])

  // Archetype Badge Info
  const archetypeInfo = useMemo(() => {
    switch (archetype) {
      case "coding":
        return { label: "Live Coding Challenge", icon: Code, color: "text-purple-400 bg-purple-500/10 border-purple-500/20" }
      case "system_design":
        return { label: "System Architecture", icon: GitFork, color: "text-blue-400 bg-blue-500/10 border-blue-500/20" }
      default:
        return { label: "Conversational Stage", icon: ChatCircleText, color: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20" }
    }
  }, [archetype])

  const ArchetypeIcon = archetypeInfo.icon

  // Color state based on percent remaining
  const timerStatusColor = useMemo(() => {
    if (graceClock !== null || questionClock <= 15) return "text-rose-400 border-rose-500/30 bg-rose-500/10"
    if (questionPercent < 30) return "text-amber-400 border-amber-500/30 bg-amber-500/10"
    return "text-emerald-400 border-emerald-500/30 bg-emerald-500/10"
  }, [questionClock, questionPercent, graceClock])

  const progressBarColor = useMemo(() => {
    if (graceClock !== null || questionClock <= 15) return "bg-rose-500"
    if (questionPercent < 30) return "bg-amber-500"
    return "bg-emerald-500"
  }, [questionClock, questionPercent, graceClock])

  return (
    <div className="w-full space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
        {/* Stage Archetype & Question Progress */}
        <div className="flex items-center gap-2">
          <div className={cn("inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-medium", archetypeInfo.color)}>
            <ArchetypeIcon className="h-3.5 w-3.5" />
            <span>{archetypeInfo.label}</span>
          </div>
          {total > 0 && (
            <span className="text-muted-foreground font-mono">
              Stage {currentIdx} of {total}
            </span>
          )}
        </div>

        {/* Timers Row */}
        <div className="flex items-center gap-3 font-mono">
          {/* Question Countdown Badge */}
          <div
            className={cn(
              "inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 font-semibold transition-colors duration-300",
              timerStatusColor,
              (questionClock <= 15 || graceClock !== null) && "animate-pulse"
            )}
            title="Time remaining for this question"
          >
            <Clock className="h-3.5 w-3.5" />
            <span>Stage: {formatTime(questionClock)}</span>
          </div>

          {/* Global Session Cap Badge */}
          <div
            className="inline-flex items-center gap-1.5 rounded-md border border-border bg-card px-2.5 py-1 text-muted-foreground"
            title="Global interview time budget remaining"
          >
            <Hourglass className="h-3.5 w-3.5 text-muted-foreground" />
            <span>Session: {formatTime(sessionClock)}</span>
          </div>
        </div>
      </div>

      {/* Visual Progress Bar for Current Question */}
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary/60">
        <div
          className={cn("h-full transition-all duration-1000 ease-linear", progressBarColor)}
          style={{ width: `${questionPercent}%` }}
        />
      </div>

      {/* Grace Period Auto-Advance Banner */}
      {graceClock !== null && (
        <div className="flex items-center justify-between rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-1.5 text-xs text-rose-300 animate-in fade-in slide-in-from-top-1 duration-300">
          <div className="flex items-center gap-2 font-medium">
            <Warning className="h-4 w-4 text-rose-400 animate-bounce" />
            <span>Stage time expired. Auto-submitting response in:</span>
          </div>
          <span className="font-mono font-bold text-rose-200 text-sm">
            {graceClock}s
          </span>
        </div>
      )}
    </div>
  )
}
