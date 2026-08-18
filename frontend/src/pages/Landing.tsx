import { useState, useEffect, useRef } from "react"
import { Link, useLocation } from "react-router-dom"
import {
  Sparkle,
  Briefcase,
  MicrophoneStage,
  CheckCircle,
  ArrowRight,
  Cpu,
  PaperPlaneRight,
  Robot,
  User,
  ChartBar,
  ShieldCheck,
  WarningCircle,
  LockKey,
  Scales,
  FilePdf,
  Lightning,
  Calculator,
  CaretDown,
} from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"

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

export function LandingPage() {
  const [activeScenarioId, setActiveScenarioId] = useState<string>("go-concurrency")
  const [simAnswer, setSimAnswer] = useState("")
  const [simSubmitted, setSimSubmitted] = useState(false)
  const [evalResult, setEvalResult] = useState<SimulationResult | null>(null)
  const [isEvaluating, setIsEvaluating] = useState(false)
  const location = useLocation()
  const [openFaq, setOpenFaq] = useState<number | null>(0)

  // Interactive ROI Calculator State
  const [hiresPerMonth, setHiresPerMonth] = useState(15)
  const hoursSavedPerMonth = hiresPerMonth * 4.5
  const costSavingsPerMonth = hoursSavedPerMonth * 85

  useEffect(() => {
    if (location.hash) {
      const id = location.hash.replace("#", "")
      const el = document.getElementById(id)
      if (el) {
        setTimeout(() => {
          el.scrollIntoView({ behavior: "smooth" })
        }, 100)
      }
    }
  }, [location])

  const currentScenario =
    DEMO_SCENARIOS.find((s) => s.id === activeScenarioId) ?? DEMO_SCENARIOS[0]

  const simTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (simTimerRef.current) clearTimeout(simTimerRef.current)
    }
  }, [])

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
    }, 750)
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
    <div className="space-y-24 pb-20 animate-in fade-in duration-700">
      {/* 1. HERO SECTION */}
      <section className="relative pt-12 md:pt-20 px-6 text-center max-w-5xl mx-auto space-y-8">
        <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-4 py-1.5 text-xs font-semibold text-primary shadow-sm backdrop-blur-md">
          <Sparkle className="h-4 w-4" weight="fill" />
          <span>Next-Gen Autonomous Technical Interview Platform</span>
        </div>

        <h1 className="font-display text-4xl font-extrabold tracking-tight sm:text-6xl md:text-7xl leading-[1.1]">
          Screen, Probe, and Grade Engineers with{" "}
          <span className="bg-gradient-to-r from-primary via-blue-500 to-indigo-500 bg-clip-text text-transparent">
            Real-Time AI
          </span>
        </h1>

        <p className="mx-auto max-w-2xl text-base sm:text-lg text-muted-foreground leading-relaxed">
          Intivai conducts adaptive voice and chat technical interviews, detects cheating in real-time, matches resumes using vector semantics, and generates boardroom-ready executive scorecards.
        </p>

        {/* Dual Call-to-Action */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-2">
          <Button asChild size="lg" variant="gradient" className="h-12 px-8 font-semibold shadow-lg shadow-primary/25 rounded-xl text-sm">
            <Link to="/careers">
              <Briefcase className="mr-2 h-4 w-4" weight="bold" /> Explore Careers & Apply
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="h-12 px-8 font-semibold rounded-xl text-sm border-border/80 hover:bg-muted">
            <Link to="/login">
              Recruiter Console Demo <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>

        {/* Live Metrics Row */}
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4 pt-8 max-w-4xl mx-auto border-y border-border/50 py-6">
          <div className="space-y-1">
            <p className="font-display text-3xl font-extrabold text-primary">10x</p>
            <p className="text-xs text-muted-foreground">Faster Candidate Turnaround</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-3xl font-extrabold text-foreground">100%</p>
            <p className="text-xs text-muted-foreground">Deterministic Safety Rails</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-3xl font-extrabold text-emerald-500">&lt; 3.0s</p>
            <p className="text-xs text-muted-foreground">Voice Latency with STT/TTS</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-3xl font-extrabold text-blue-500">384-Dim</p>
            <p className="text-xs text-muted-foreground">Vector Semantic Matching</p>
          </div>
        </div>
      </section>

      {/* 2. DYNAMIC INTERACTIVE DEMO SIMULATOR */}
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

      {/* 3. HOW IT WORKS LIFECYCLE */}
      <section id="how-it-works" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
            End-to-End Workflow
          </Badge>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            How Autonomous Screening Works
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            A frictionless candidate-first journey backed by rigorous multi-modal AI evaluation and deterministic enterprise rails.
          </p>
        </div>

        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <Card className="glass border-border/60 p-5 space-y-3 relative overflow-hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary font-display font-bold text-sm">
              01
            </div>
            <h3 className="font-display font-bold text-base">Job & Rail Setup</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Define required skills, experience thresholds, scoring weights, and company context prompts in your workspace.
            </p>
          </Card>

          <Card className="glass border-border/60 p-5 space-y-3 relative overflow-hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-500 font-display font-bold text-sm">
              02
            </div>
            <h3 className="font-display font-bold text-base">Semantic CV Matching</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Resumes are parsed with OCR and vector-embedded across 384 dimensions to rank competency without keyword bias.
            </p>
          </Card>

          <Card className="glass border-border/60 p-5 space-y-3 relative overflow-hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-500 font-display font-bold text-sm">
              03
            </div>
            <h3 className="font-display font-bold text-base">Voice, Chat & Sandbox</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Candidates complete adaptive voice or chat interviews with live pair-programming in Monaco editor and anti-cheat guardrails.
            </p>
          </Card>

          <Card className="glass border-border/60 p-5 space-y-3 relative overflow-hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-500 font-display font-bold text-sm">
              04
            </div>
            <h3 className="font-display font-bold text-base">Scorecards & ATS Sync</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Instant executive scorecards synthesize technical depth, problem-solving, code complexity, and export Maroto PDFs.
            </p>
          </Card>
        </div>
      </section>

      {/* 4. ENTERPRISE PROCTORING & ANTI-CHEATING SHOWCASE */}
      <section id="proctoring" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
            Integrity Guardrails
          </Badge>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Enterprise Anti-Cheating & AI Proctoring
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            Hiring decisions require absolute integrity. Intivai actively monitors telemetry across browser focus, clipboard activity, and audio streams to prevent AI-generated ghostwriting.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card className="glass border-border/60 p-6 space-y-4 hover:border-primary/40 transition-all shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
              <ShieldCheck className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Focus & Tab-Switch Tracking</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Continuously logs window blur, tab switching, and away duration. Frequent departures trigger automated penalty calculations and recruiter audit flags.
            </p>
          </Card>

          <Card className="glass border-border/60 p-6 space-y-4 hover:border-primary/40 transition-all shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500">
              <Lightning className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Clipboard Paste Telemetry</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Detects sudden multi-paragraph code or text pastes within seconds of question dispatch, preventing candidates from copy-pasting answers from external LLMs.
            </p>
          </Card>

          <Card className="glass border-border/60 p-6 space-y-4 hover:border-primary/40 transition-all shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-500">
              <MicrophoneStage className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Voice Stream Audio Anomaly</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Analyzes incoming WebRTC speech frequencies to detect secondary background speakers, whisper prompting, or synthetic proxy voices during live voice rounds.
            </p>
          </Card>
        </div>
      </section>

      {/* 5. CORE PLATFORM PILLARS */}
      <section id="features" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
            Complete Architecture
          </Badge>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Built for Modern Engineering Hiring
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            From the moment a CV is uploaded to final score synthesis, Intivai operates with complete autonomy and enterprise isolation.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card className="glass border-border/60 hover:border-primary/40 transition-all hover:shadow-xl hover:shadow-primary/5 p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-500/10 text-blue-500">
              <Cpu className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Semantic CV Vector Screening</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Embeds candidates into 384-dimensional vector spaces using pgvector and cosine distance to match technical competencies against required role weights.
            </p>
          </Card>

          <Card className="glass border-border/60 hover:border-primary/40 transition-all hover:shadow-xl hover:shadow-primary/5 p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
              <MicrophoneStage className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">WebRTC Voice Interviewing</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Full duplex audio calling powered by Whisper.cpp Speech-to-Text and Edge TTS neural synthesis, allowing seamless natural spoken technical discussions.
            </p>
          </Card>

          <Card className="glass border-border/60 hover:border-primary/40 transition-all hover:shadow-xl hover:shadow-primary/5 p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-500">
              <FilePdf className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Executive PDF Scorecards</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Generates PDF reports breaking down Technical Proficiency, Problem Solving, Communication Clarity, and Culture Fit with per-question reasoning.
            </p>
          </Card>
        </div>
      </section>

      {/* 6. INTERACTIVE RECRUITER ROI CALCULATOR */}
      <section id="calculator" className="scroll-mt-24 px-6 max-w-4xl mx-auto">
        <Card className="glass border-primary/30 p-8 md:p-10 shadow-xl shadow-primary/5 rounded-3xl space-y-6">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-border/50 pb-6">
            <div className="space-y-1">
              <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
                Recruitment Efficiency Calculator
              </Badge>
              <h3 className="font-display font-bold text-2xl">Calculate Your Engineering Time Saved</h3>
            </div>
            <div className="flex items-center gap-2 text-primary">
              <Calculator className="h-8 w-8" weight="duotone" />
            </div>
          </div>

          <div className="grid gap-6 md:grid-cols-2 items-center">
            <div className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-foreground uppercase tracking-wider block mb-2">
                  Technical Interviews Conducted Per Month: <span className="text-primary font-bold text-base ml-1">{hiresPerMonth}</span>
                </label>
                <input
                  type="range"
                  min="5"
                  max="100"
                  step="5"
                  value={hiresPerMonth}
                  onChange={(e) => setHiresPerMonth(Number(e.target.value))}
                  className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
                />
                <div className="flex justify-between text-[11px] text-muted-foreground mt-1">
                  <span>5 interviews</span>
                  <span>50 interviews</span>
                  <span>100+ interviews</span>
                </div>
              </div>

              <div className="space-y-2 text-xs text-muted-foreground">
                <p className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" weight="fill" />
                  <span>Saves 4.5 engineering hours per candidate screened</span>
                </p>
                <p className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" weight="fill" />
                  <span>Eliminates recruiter scheduling bottlenecks</span>
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 p-4 rounded-2xl bg-muted/40 border border-border/50 text-center">
              <div className="p-3 bg-card rounded-xl border border-border/40">
                <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
                  Dev Hours Saved
                </span>
                <span className="font-display text-2xl sm:text-3xl font-extrabold text-primary block mt-1">
                  {hoursSavedPerMonth}h
                </span>
                <span className="text-[10px] text-muted-foreground">per month</span>
              </div>
              <div className="p-3 bg-card rounded-xl border border-border/40">
                <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
                  Estimated Savings
                </span>
                <span className="font-display text-2xl sm:text-3xl font-extrabold text-emerald-500 block mt-1">
                  ${costSavingsPerMonth.toLocaleString()}
                </span>
                <span className="text-[10px] text-muted-foreground">in eng bandwidth</span>
              </div>
            </div>
          </div>
        </Card>
      </section>

      {/* 7. ENTERPRISE COMPLIANCE & SECURITY */}
      <section id="security" className="scroll-mt-24 px-6 max-w-5xl mx-auto space-y-8">
        <div className="text-center space-y-3">
          <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
            Enterprise Grade
          </Badge>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Security, Privacy & Bias-Free Compliance
          </h2>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-2xl border border-border/60 bg-card p-5 space-y-2">
            <LockKey className="h-6 w-6 text-primary" weight="bold" />
            <h4 className="font-display font-bold text-sm">Postgres Row-Level Security (RLS)</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Strict multi-tenant organization isolation enforced at the database kernel level with cryptographic tenant session bounds.
            </p>
          </div>

          <div className="rounded-2xl border border-border/60 bg-card p-5 space-y-2">
            <Scales className="h-6 w-6 text-emerald-500" weight="bold" />
            <h4 className="font-display font-bold text-sm">GDPR & AI Bias Mitigation</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Mandatory candidate consent gates before interview execution. Questions are stripped of demographic bias and anchored purely on technical rubrics.
            </p>
          </div>

          <div className="rounded-2xl border border-border/60 bg-card p-5 space-y-2">
            <ShieldCheck className="h-6 w-6 text-blue-500" weight="bold" />
            <h4 className="font-display font-bold text-sm">Automated Mailer & ATS Webhooks</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Direct SMTP notifications via Mailpit with invitation magic links and structured JSON payloads ready for ATS pipeline synchronization.
            </p>
          </div>
        </div>
      </section>

      {/* 8. FREQUENTLY ASKED QUESTIONS (FAQ) */}
      <section id="faq" className="scroll-mt-24 px-6 max-w-3xl mx-auto space-y-6">
        <div className="text-center space-y-2">
          <h2 className="font-display text-2xl sm:text-3xl font-bold">Frequently Asked Questions</h2>
          <p className="text-xs sm:text-sm text-muted-foreground">
            Everything you need to know about autonomous technical screening with Intivai.
          </p>
        </div>

        <div className="space-y-3 pt-2">
          {[
            {
              q: "How does the AI adapt during live interviews?",
              a: "Intivai generates gap-verification questions tailored to the candidate's resume and job requirements. If an answer is brief, it autonomously probes deeper into specific failure modes and trade-offs.",
            },
            {
              q: "How does anti-cheating detection work?",
              a: "During both chat and voice rounds, our client and backend monitor window focus state, clipboard paste sizes vs elapsed time, and audio stream frequency anomalies to flag potential external assistance.",
            },
            {
              q: "Can recruiters customize the grading rubric?",
              a: "Yes. Recruiters can specify custom company context prompts, mandatory technical skills, required experience levels, and question count per role requisition.",
            },
            {
              q: "Is candidate data isolated per tenant?",
              a: "Yes. All resumes, transcripts, and evaluation scorecards are guarded by strict PostgreSQL Row-Level Security (RLS) policies scoped exclusively to your organization workspace.",
            },
          ].map((item, idx) => (
            <div
              key={idx}
              className="rounded-2xl border border-border/60 bg-card p-4 transition-all cursor-pointer"
              onClick={() => setOpenFaq(openFaq === idx ? null : idx)}
            >
              <div className="flex items-center justify-between gap-2">
                <h4 className="font-display font-bold text-xs sm:text-sm text-foreground">{item.q}</h4>
                <CaretDown
                  className={`h-4 w-4 text-muted-foreground transition-transform ${
                    openFaq === idx ? "rotate-180 text-primary" : ""
                  }`}
                  weight="bold"
                />
              </div>
              {openFaq === idx && (
                <p className="text-xs text-muted-foreground pt-2.5 leading-relaxed border-t border-border/40 mt-2.5">
                  {item.a}
                </p>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* 8. CALL TO ACTION BANNER */}
      <section className="px-6 max-w-5xl mx-auto">
        <div className="glass rounded-3xl border border-primary/30 bg-gradient-to-r from-primary/15 via-card to-blue-500/15 p-10 md:p-16 text-center space-y-6 shadow-2xl shadow-primary/10">
          <h2 className="font-display text-3xl sm:text-5xl font-extrabold tracking-tight">
            Ready to Automate Your Technical Hiring?
          </h2>
          <p className="text-sm sm:text-base text-muted-foreground max-w-xl mx-auto">
            Browse our open job openings as a candidate or launch your organization workspace today.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4 pt-2">
            <Button asChild size="lg" variant="gradient" className="h-12 px-8 font-semibold shadow-lg shadow-primary/25 rounded-xl">
              <Link to="/careers">Browse Open Roles</Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="h-12 px-8 font-semibold rounded-xl">
              <Link to="/register">Create Recruiter Workspace</Link>
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
