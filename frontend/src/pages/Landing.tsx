import { useState, useEffect } from "react"
import { Link, useLocation } from "react-router-dom"
import {
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
    <div className="space-y-16 md:space-y-24 pb-20 animate-in fade-in duration-700">
      {/* 1. HERO SECTION */}
      <section className="relative pt-12 md:pt-20 px-6 text-center max-w-5xl mx-auto space-y-8">
        <h1 className="font-display text-4xl font-extrabold tracking-tight sm:text-6xl md:text-7xl leading-[1.1] text-foreground">
          Autonomous Technical Interviews Backed by Objective Engineering Rubrics
        </h1>

        <p className="mx-auto max-w-2xl text-base sm:text-lg text-muted-foreground leading-relaxed">
          Intivai pairs adaptive technical interviewing and live code evaluation with vector-based resume matching and auditable scoring rubrics.
        </p>

        {/* Dual Call-to-Action */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-2">
          <Button asChild size="lg" className="h-11 px-6 font-semibold shadow-sm text-sm">
            <Link to="/careers">
              <Briefcase className="mr-2 h-4 w-4" weight="bold" /> Explore Careers & Apply
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="h-11 px-6 font-semibold text-sm border-border hover:bg-muted">
            <Link to="/login">
              Recruiter Console Demo <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>

        {/* Live Metrics Row */}
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4 pt-8 max-w-4xl mx-auto border-y border-border py-6 text-left">
          <div className="space-y-1">
            <p className="font-display text-base font-bold text-foreground">Adaptive Chat & Sandbox</p>
            <p className="text-xs text-muted-foreground">Interactive questioning with Monaco editor & automated test runs</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-base font-bold text-foreground">Auditable Rubrics</p>
            <p className="text-xs text-muted-foreground">Full transcripts, code execution logs, and per-question rationale</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-base font-bold text-foreground">384-Dim Vector Match</p>
            <p className="text-xs text-muted-foreground">Semantic resume scoring via pgvector cosine distance</p>
          </div>
          <div className="space-y-1">
            <p className="font-display text-base font-bold text-foreground">Tenant-Isolated</p>
            <p className="text-xs text-muted-foreground">Kernel-enforced PostgreSQL Row-Level Security per organization</p>
          </div>
        </div>
      </section>

      {/* 2. DYNAMIC INTERACTIVE DEMO SIMULATOR */}
      <DemoSimulator />

      {/* 3. HOW IT WORKS LIFECYCLE */}
      <section id="how-it-works" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            How Autonomous Screening Works
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            A structured candidate evaluation process backed by deterministic technical rubrics and telemetry validation.
          </p>
        </div>

        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <Card className="border-border bg-card p-5 space-y-3 relative overflow-hidden shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary font-display font-bold text-sm">
              01
            </div>
            <h3 className="font-display font-bold text-base">Job & Rail Setup</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Define required skills, experience thresholds, scoring weights, and company context prompts in your workspace.
            </p>
          </Card>

          <Card className="border-border bg-card p-5 space-y-3 relative overflow-hidden shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10 text-info font-display font-bold text-sm">
              02
            </div>
            <h3 className="font-display font-bold text-base">Semantic CV Matching</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Resumes are parsed with OCR and vector-embedded across 384 dimensions to rank competency without keyword bias.
            </p>
          </Card>

          <Card className="border-border bg-card p-5 space-y-3 relative overflow-hidden shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent/10 text-accent font-display font-bold text-sm">
              03
            </div>
            <h3 className="font-display font-bold text-base">Chat & Coding Sandbox</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Candidates complete adaptive chat interviews with live pair-programming in Monaco editor and anti-cheat guardrails.
            </p>
          </Card>

          <Card className="border-border bg-card p-5 space-y-3 relative overflow-hidden shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success/10 text-success font-display font-bold text-sm">
              04
            </div>
            <h3 className="font-display font-bold text-base">Scorecards & ATS Sync</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Executive scorecards synthesize technical depth, problem-solving, code complexity, and export Maroto PDFs.
            </p>
          </Card>
        </div>
      </section>

      {/* 4. ENTERPRISE PROCTORING & ANTI-CHEATING SHOWCASE */}
      <section id="proctoring" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Integrity Guardrails & Telemetry
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            Hiring decisions require verifiable integrity. Intivai logs environment signals across window focus, clipboard activity, and audio streams.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card className="border-border bg-card p-6 space-y-4 shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <ShieldCheck className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Focus & Tab-Switch Tracking</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Logs window blur, tab switching, and away duration. Frequent departures trigger automated penalty flags for recruiter review.
            </p>
          </Card>

          <Card className="border-border bg-card p-6 space-y-4 shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning/10 text-warning">
              <Lightning className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Clipboard Paste Telemetry</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Detects large text and code pastes within short intervals after question dispatch to flag unverified external assistance.
            </p>
          </Card>

          <Card className="border-border bg-card p-6 space-y-4 shadow-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-success/10 text-success">
              <MicrophoneStage className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Voice Stream Audio Anomaly</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Planned capability for voice rounds (early-access demo): frequency analysis to detect secondary background speakers or synthetic proxies.
            </p>
          </Card>
        </div>
      </section>

      {/* 5. CORE PLATFORM PILLARS */}
      <section id="features" className="scroll-mt-24 px-6 max-w-6xl mx-auto space-y-12">
        <div className="text-center space-y-3">
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Built for Modern Engineering Hiring
          </h2>
          <p className="text-sm text-muted-foreground max-w-2xl mx-auto">
            From CV ingestion to final scorecard synthesis, Intivai operates with complete autonomy and tenant isolation.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card className="border-border bg-card shadow-sm p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-info/10 text-info">
              <Cpu className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">Semantic CV Vector Screening</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Embeds candidates into 384-dimensional vector spaces using pgvector and cosine distance to match technical competencies against required role weights.
            </p>
          </Card>

          <Card className="border-border bg-card shadow-sm p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <MicrophoneStage className="h-6 w-6" weight="bold" />
            </div>
            <h3 className="font-display font-bold text-lg">WebRTC Voice Interviewing</h3>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Early-access demo: full duplex audio calling powered by Whisper.cpp Speech-to-Text and Edge TTS neural synthesis for natural spoken technical discussions.
            </p>
          </Card>

          <Card className="border-border bg-card shadow-sm p-6 space-y-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-success/10 text-success">
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
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
            Security, Privacy & Bias-Free Compliance
          </h2>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-lg border border-border bg-card p-5 space-y-2 shadow-sm">
            <LockKey className="h-6 w-6 text-primary" weight="bold" />
            <h4 className="font-display font-bold text-sm">Postgres Row-Level Security (RLS)</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Strict multi-tenant organization isolation enforced at the database kernel level with cryptographic tenant session bounds.
            </p>
          </div>

          <div className="rounded-lg border border-border bg-card p-5 space-y-2 shadow-sm">
            <Scales className="h-6 w-6 text-success" weight="bold" />
            <h4 className="font-display font-bold text-sm">GDPR & AI Bias Mitigation</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Mandatory candidate consent gates before interview execution. Questions are stripped of demographic bias and anchored purely on technical rubrics.
            </p>
          </div>

          <div className="rounded-lg border border-border bg-card p-5 space-y-2 shadow-sm">
            <ShieldCheck className="h-6 w-6 text-info" weight="bold" />
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
              a: "During chat interviews, our client and backend monitor window focus state and clipboard paste sizes vs elapsed time to flag potential external assistance. Audio frequency anomaly detection is planned for voice rounds.",
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
              className="rounded-lg border border-border bg-card p-4 transition-all cursor-pointer shadow-sm"
              onClick={() => setOpenFaq(openFaq === idx ? null : idx)}
              role="button"
              tabIndex={0}
              aria-expanded={openFaq === idx}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault()
                  setOpenFaq(openFaq === idx ? null : idx)
                }
              }}
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
              <div className={`overflow-hidden transition-all duration-300 ease-in-out ${openFaq === idx ? "max-h-40 opacity-100" : "max-h-0 opacity-0"}`}>
                <p className="text-xs text-muted-foreground pt-2.5 leading-relaxed border-t border-border mt-2.5">
                  {item.a}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* 9. CALL TO ACTION BANNER */}
      <section className="px-6 max-w-5xl mx-auto">
        <div className="rounded-xl border border-border bg-card p-8 md:p-12 text-center space-y-4 shadow-sm">
          <h2 className="font-display text-2xl sm:text-4xl font-bold tracking-tight">
            Ready to Automate Your Technical Hiring?
          </h2>
          <p className="text-sm sm:text-base text-muted-foreground max-w-xl mx-auto">
            Browse our open job openings as a candidate or launch your organization workspace today.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4 pt-2">
            <Button asChild size="lg" className="h-11 px-6 font-semibold shadow-sm">
              <Link to="/careers">Browse Open Roles</Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="h-11 px-6 font-semibold border-border">
              <Link to="/register">Create Recruiter Workspace</Link>
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
