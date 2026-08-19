import { useEffect, useRef, useState } from "react"
import {
  Sparkle,
  CheckCircle,
  PaperPlaneRight,
  Robot,
  User,
  ChartBar,
  WarningCircle,
} from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"

const FAKE_EVALUATION_DELAY_MS = 750

interface DemoScenario {
  id: string
  role: string
  category: string
  question: string
  keywords: string[]
  sampleGood: string
  sampleBasic: string
}

const DEMO_SCENARIOS: DemoScenario[] = [
  {
    id: "go-concurrency",
    role: "Senior Go / Backend Engineer",
    category: "Go Concurrency & Channels",
    question:
      "How do Goroutines communicate safely in Go without relying on manual mutex locking, and when would you prefer buffered over unbuffered channels?",
    keywords: ["channel", "csp", "unbuffered", "buffered", "blocking", "sync", "deadlock", "goroutine", "select", "queue", "throughput", "buffer"],
    sampleGood:
      "In Go, Goroutines follow CSP concurrency by communicating through channels. Unbuffered channels provide synchronous handoffs where the sender blocks until receiver is ready. I prefer buffered channels when producing bursts of work to decouple producer-consumer throughput without blocking.",
    sampleBasic:
      "Goroutines use channels to send data between each other. Buffered channels have a fixed size so you can send items into them.",
  },
  {
    id: "react-architecture",
    role: "Staff Frontend Architect",
    category: "React 19 & Server Architecture",
    question:
      "How does React 19 differentiate Server Components from Client Components, and how do you prevent hydration mismatches during server rendering?",
    keywords: ["server", "client", "rsc", "hydration", "bundle", "use client", "suspense", "dom", "isomorphic", "tree"],
    sampleGood:
      "React Server Components execute exclusively on the server and stream zero JavaScript to the browser bundle, whereas Client Components ('use client') provide interactive state and event handlers. To prevent hydration mismatches, ensure deterministic initial HTML and defer browser-only APIs like window or localStorage to useEffect.",
    sampleBasic:
      "Server components render on the server to make the page load faster and client components render on the client for buttons and clicks.",
  },
  {
    id: "system-design",
    role: "Principal Infrastructure Engineer",
    category: "High-Availability Distributed Systems",
    question:
      "How do you design a zero-downtime database schema migration strategy in a high-throughput multi-tenant SaaS application?",
    keywords: ["expand", "contract", "backward", "compatible", "dual", "write", "replica", "lock", "index", "cdc", "canary", "tenant"],
    sampleGood:
      "I apply the Expand and Contract pattern with phased backward-compatible releases. First expand: add the new column or table as nullable without breaking existing queries. Dual-write in application layer, backfill existing records asynchronously, switch reads, and finally contract by removing legacy columns.",
    sampleBasic:
      "I run migrations during off-peak maintenance windows and take a backup before running alter table.",
  },
]

interface SimulationResult {
  score: number
  depth: "exceptional" | "proficient" | "needs_depth" | "insufficient"
  feedback: string
  strengths: string[]
  gaps: string[]
  dimensions: { technical: number; communication: number; problem_solving: number }
}

function evaluateSimulatedAnswer(answer: string, scenario: DemoScenario): SimulationResult {
  const text = answer.toLowerCase().trim()
  const words = text.split(/\s+/).filter(Boolean)

  if (words.length < 5) {
    return {
      score: 24,
      depth: "insufficient",
      feedback: "Answer is too brief or evasive. Technical competency could not be verified.",
      strengths: ["Answer submitted"],
      gaps: ["No core technical mechanisms explained", "Missing domain-specific terminology"],
      dimensions: { technical: 20, communication: 30, problem_solving: 20 },
    }
  }

  // Count keyword coverage
  const matchedKeywords = scenario.keywords.filter((kw) => text.includes(kw))
  const keywordRatio = matchedKeywords.length / Math.min(scenario.keywords.length, 6)

  // Evaluation heuristic
  let score = 40 + Math.round(keywordRatio * 45)
  if (words.length >= 25) score += 8
  if (words.length >= 45) score += 5
  if (text.includes("because") || text.includes("whereas") || text.includes("prevent") || text.includes("pattern")) {
    score += 4
  }

  if (score > 98) score = 98
  if (score < 30) score = 30

  const strengths: string[] = []
  const gaps: string[] = []

  if (matchedKeywords.length >= 3) {
    strengths.push(`Accurately addressed key concepts: ${matchedKeywords.slice(0, 3).join(", ")}`)
  }
  if (words.length >= 30) {
    strengths.push("Articulated trade-offs and structural architectural reasoning")
  }
  if (strengths.length === 0) {
    strengths.push("Attempted relevant topic discussion")
  }

  const missingKeywords = scenario.keywords.filter((kw) => !text.includes(kw))
  if (missingKeywords.length > 0 && score < 90) {
    gaps.push(`Could expand deeper on: ${missingKeywords.slice(0, 2).join(", ")}`)
  }
  if (words.length < 25) {
    gaps.push("Elaborate with concrete production examples or failure edge cases")
  }

  let depth: SimulationResult["depth"] = "insufficient"
  let feedback = ""

  if (score >= 88) {
    depth = "exceptional"
    feedback = `Exceptional depth. Demonstrates senior-level mastery with clear architectural causality and precise terminology for ${scenario.category}.`
  } else if (score >= 70) {
    depth = "proficient"
    feedback = `Solid technical competency. Covers primary fundamentals of ${scenario.category} with good understanding of standard trade-offs.`
  } else if (score >= 50) {
    depth = "needs_depth"
    feedback = `Basic conceptual grasp, but lacks detailed operational depth on key mechanisms and concurrency/lifecycle trade-offs.`
  } else {
    depth = "insufficient"
    feedback = `Surface-level response. Lacks critical technical foundations and precision expected for ${scenario.role}.`
  }

  return {
    score,
    depth,
    feedback,
    strengths,
    gaps,
    dimensions: {
      technical: Math.min(100, score + 2),
      communication: Math.min(100, Math.round(score * 0.95) + (words.length > 20 ? 5 : 0)),
      problem_solving: Math.min(100, Math.round(score * 0.9) + (matchedKeywords.length > 2 ? 8 : 0)),
    },
  }
}

export function DemoSimulator() {
  const [activeScenarioId, setActiveScenarioId] = useState<string>("go-concurrency")
  const [simAnswer, setSimAnswer] = useState("")
  const [simSubmitted, setSimSubmitted] = useState(false)
  const [evalResult, setEvalResult] = useState<SimulationResult | null>(null)
  const [isEvaluating, setIsEvaluating] = useState(false)

  const simTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (simTimerRef.current) clearTimeout(simTimerRef.current)
    }
  }, [])

  const currentScenario =
    DEMO_SCENARIOS.find((s) => s.id === activeScenarioId) ?? DEMO_SCENARIOS[0]

  function handleSimulate() {
    if (!simAnswer.trim()) return
    setIsEvaluating(true)
    setSimSubmitted(true)
    if (simTimerRef.current) clearTimeout(simTimerRef.current)
    simTimerRef.current = setTimeout(() => {
      const res = evaluateSimulatedAnswer(simAnswer, currentScenario)
      setEvalResult(res)
      setIsEvaluating(false)
      simTimerRef.current = null
    }, FAKE_EVALUATION_DELAY_MS)
  }

  function handleScenarioChange(id: string) {
    // A pending evaluation must not land on the newly selected scenario.
    if (simTimerRef.current) {
      clearTimeout(simTimerRef.current)
      simTimerRef.current = null
      setIsEvaluating(false)
    }
    setActiveScenarioId(id)
    setSimSubmitted(false)
    setEvalResult(null)
    setSimAnswer("")
  }

  return (
    <section id="demo" className="px-6 max-w-4xl mx-auto">
      <div className="text-center space-y-2 mb-8">
        <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
          Live AI Competency Engine
        </Badge>
        <h2 className="font-display text-2xl sm:text-3xl font-bold">Interactive AI Technical Evaluator</h2>
        <p className="text-xs sm:text-sm text-muted-foreground max-w-xl mx-auto">
          Test the live scoring engine below. Choose an engineering discipline, type an answer (or insert a sample), and see real-time competency breakdown.
        </p>
      </div>

      {/* Scenario Selector Tabs */}
      <div className="flex flex-wrap items-center justify-center gap-2 mb-4">
        {DEMO_SCENARIOS.map((scenario) => (
          <button
            key={scenario.id}
            onClick={() => handleScenarioChange(scenario.id)}
            className={`px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              activeScenarioId === scenario.id
                ? "bg-primary text-primary-foreground shadow-md shadow-primary/20"
                : "bg-card border border-border/60 text-muted-foreground hover:text-foreground hover:bg-muted/50"
            }`}
          >
            {scenario.category}
          </button>
        ))}
      </div>

      <Card className="glass border-primary/30 shadow-2xl shadow-primary/10 overflow-hidden">
        <div className="border-b border-border/50 bg-muted/40 px-6 py-3.5 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground text-xs font-bold font-display">
              I
            </div>
            <div>
              <span className="font-display font-semibold text-xs text-foreground block">
                Intivai AI Interviewer
              </span>
              <span className="text-[10px] text-muted-foreground block">{currentScenario.role}</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-[10px]">
              Adaptive Rubric Active
            </Badge>
          </div>
        </div>

        <CardContent className="p-6 space-y-5">
          {/* AI Question */}
          <div className="flex items-start gap-3">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
              <Robot className="h-4 w-4" weight="fill" />
            </div>
            <div className="rounded-2xl rounded-tl-sm bg-card border border-primary/20 p-4 text-xs sm:text-sm leading-relaxed max-w-2xl">
              <p className="font-medium text-foreground">"{currentScenario.question}"</p>
            </div>
          </div>

          {/* Quick Answer Insertion Presets */}
          {!simSubmitted && (
            <div className="flex flex-wrap items-center gap-2 pt-1">
              <span className="text-[11px] text-muted-foreground font-medium">Quick Insert:</span>
              <button
                type="button"
                onClick={() => setSimAnswer(currentScenario.sampleGood)}
                className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1 text-[11px] text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20 font-medium"
              >
                ✓ Strong Senior Answer
              </button>
              <button
                type="button"
                onClick={() => setSimAnswer(currentScenario.sampleBasic)}
                className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-2.5 py-1 text-[11px] text-amber-600 dark:text-amber-400 hover:bg-amber-500/20 font-medium"
              >
                ⚠ Basic / Brief Answer
              </button>
              <button
                type="button"
                onClick={() => setSimAnswer("I have worked with this before in some projects.")}
                className="rounded-lg border border-destructive/30 bg-destructive/10 px-2.5 py-1 text-[11px] text-destructive hover:bg-destructive/20 font-medium"
              >
                ✕ Evasive Answer
              </button>
            </div>
          )}

          {/* Candidate Response Output if submitted */}
          {simSubmitted && (
            <div className="flex items-start justify-end gap-3 animate-in fade-in duration-300">
              <div className="rounded-2xl rounded-tr-sm bg-primary text-primary-foreground p-4 text-xs sm:text-sm leading-relaxed max-w-xl">
                <p>{simAnswer}</p>
              </div>
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground border border-border">
                <User className="h-4 w-4" weight="bold" />
              </div>
            </div>
          )}

          {/* AI Real-time Dynamic Feedback */}
          {isEvaluating && (
            <div className="flex items-center gap-3 p-4 rounded-xl border border-primary/20 bg-primary/5 text-xs text-primary animate-pulse">
              <Sparkle className="h-5 w-5 animate-spin" weight="bold" />
              <span>Analyzing response against technical rubric and trade-off vectors...</span>
            </div>
          )}

          {evalResult && (
            <div className="space-y-3 animate-in fade-in zoom-in-95 duration-300">
              <div className="rounded-2xl border border-border/80 bg-card p-5 space-y-4 shadow-sm">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border/50 pb-3">
                  <div className="flex items-center gap-2">
                    <div
                      className={`flex h-8 w-8 items-center justify-center rounded-xl font-bold text-xs ${
                        evalResult.score >= 80
                          ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                          : evalResult.score >= 60
                          ? "bg-amber-500/15 text-amber-600 dark:text-amber-400"
                          : "bg-destructive/15 text-destructive"
                      }`}
                    >
                      <ChartBar className="h-4 w-4" weight="bold" />
                    </div>
                    <div>
                      <h4 className="font-display font-bold text-sm">Dynamic AI Competency Analysis</h4>
                      <p className="text-[11px] text-muted-foreground capitalize">
                        Verdict: {evalResult.depth.replace("_", " ")}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <Badge
                      variant="outline"
                      className={`font-mono text-xs px-2.5 py-1 font-bold ${
                        evalResult.score >= 80
                          ? "border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/5"
                          : evalResult.score >= 60
                          ? "border-amber-500/30 text-amber-600 dark:text-amber-400 bg-amber-500/5"
                          : "border-destructive/30 text-destructive bg-destructive/5"
                      }`}
                    >
                      Score: {evalResult.score} / 100
                    </Badge>
                  </div>
                </div>

                <p className="text-xs text-foreground/90 leading-relaxed">{evalResult.feedback}</p>

                {/* Dimension Ratings */}
                <div className="grid grid-cols-3 gap-2 pt-1 text-center">
                  <div className="rounded-lg bg-muted/40 p-2 border border-border/40">
                    <span className="text-[10px] text-muted-foreground uppercase font-semibold block">
                      Tech Depth
                    </span>
                    <span className="font-mono font-bold text-xs text-foreground">
                      {evalResult.dimensions.technical}/100
                    </span>
                  </div>
                  <div className="rounded-lg bg-muted/40 p-2 border border-border/40">
                    <span className="text-[10px] text-muted-foreground uppercase font-semibold block">
                      Communication
                    </span>
                    <span className="font-mono font-bold text-xs text-foreground">
                      {evalResult.dimensions.communication}/100
                    </span>
                  </div>
                  <div className="rounded-lg bg-muted/40 p-2 border border-border/40">
                    <span className="text-[10px] text-muted-foreground uppercase font-semibold block">
                      Problem Solving
                    </span>
                    <span className="font-mono font-bold text-xs text-foreground">
                      {evalResult.dimensions.problem_solving}/100
                    </span>
                  </div>
                </div>

                {/* Strengths & Gaps */}
                <div className="grid gap-2 sm:grid-cols-2 pt-1 text-xs">
                  {evalResult.strengths.length > 0 && (
                    <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3 space-y-1">
                      <span className="font-bold text-[11px] text-emerald-600 dark:text-emerald-400 flex items-center gap-1 uppercase tracking-wider">
                        <CheckCircle className="h-3.5 w-3.5" weight="fill" /> Strengths
                      </span>
                      {evalResult.strengths.map((s, idx) => (
                        <p key={idx} className="text-muted-foreground text-[11px]">
                          • {s}
                        </p>
                      ))}
                    </div>
                  )}
                  {evalResult.gaps.length > 0 && (
                    <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-3 space-y-1">
                      <span className="font-bold text-[11px] text-amber-600 dark:text-amber-400 flex items-center gap-1 uppercase tracking-wider">
                        <WarningCircle className="h-3.5 w-3.5" weight="fill" /> Growth Areas
                      </span>
                      {evalResult.gaps.map((g, idx) => (
                        <p key={idx} className="text-muted-foreground text-[11px]">
                          • {g}
                        </p>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Simulation Input */}
          {!simSubmitted ? (
            <div className="space-y-2 pt-2">
              <Textarea
                placeholder="Type your response to test the AI evaluation model, or click one of the quick presets above..."
                value={simAnswer}
                onChange={(e) => setSimAnswer(e.target.value)}
                className="bg-background text-xs sm:text-sm min-h-[72px] resize-none"
              />
              <div className="flex justify-end">
                <Button
                  variant="gradient"
                  className="h-10 px-6 font-semibold"
                  onClick={handleSimulate}
                  disabled={!simAnswer.trim() || isEvaluating}
                >
                  <PaperPlaneRight className="mr-1.5 h-4 w-4" weight="bold" /> Evaluate Competency
                </Button>
              </div>
            </div>
          ) : (
            <div className="pt-2 text-center">
              <Button
                variant="outline"
                size="sm"
                className="text-xs rounded-xl"
                onClick={() => {
                  setSimSubmitted(false)
                  setEvalResult(null)
                  setSimAnswer("")
                }}
              >
                Try Another Response / Scenario
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  )
}
