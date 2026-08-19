import { useState, useEffect } from "react"
import { Link, useLocation } from "react-router-dom"
import {
  Sparkle,
  Briefcase,
  MicrophoneStage,
  ArrowRight,
  Cpu,
  ShieldCheck,
  LockKey,
  Scales,
  FilePdf,
  Lightning,
  CaretDown,
} from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { DemoSimulator } from "@/components/landing/DemoSimulator"
import { RoiCalculator } from "@/components/landing/RoiCalculator"

const SCROLL_DELAY_MS = 100

export function LandingPage() {
  const location = useLocation()
  const [openFaq, setOpenFaq] = useState<number | null>(0)

  useEffect(() => {
    if (location.hash) {
      const id = location.hash.replace("#", "")
      const el = document.getElementById(id)
      if (el) {
        setTimeout(() => {
          el.scrollIntoView({ behavior: "smooth" })
        }, SCROLL_DELAY_MS)
      }
    }
  }, [location])

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
      <DemoSimulator />

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
      <RoiCalculator />

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

      {/* 9. CALL TO ACTION BANNER */}
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
