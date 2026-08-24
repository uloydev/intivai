import { Link } from "react-router-dom"
import { ArrowLeft } from "@phosphor-icons/react"

interface LegalPageProps {
  title: string
  lastUpdated: string
  children: React.ReactNode
}

function LegalLayout({ title, lastUpdated, children }: LegalPageProps) {
  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-3xl px-6 py-16">
        <Link to="/" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors mb-8">
          <ArrowLeft className="h-4 w-4" /> Back to home
        </Link>
        <h1 className="font-display text-3xl font-bold tracking-tight mb-2">{title}</h1>
        <p className="text-sm text-muted-foreground mb-10">Last updated: {lastUpdated}</p>
        <div className="prose prose-sm max-w-none space-y-6 text-foreground/80 leading-relaxed">
          {children}
        </div>
      </div>
    </div>
  )
}

export function PrivacyPolicyPage() {
  return (
    <LegalLayout title="Privacy Policy" lastUpdated="August 2026">
      <h2 className="text-lg font-semibold text-foreground">1. Data We Collect</h2>
      <p>We collect the following data when you use Intivai:</p>
      <ul className="list-disc pl-5 space-y-1">
        <li><strong>Account information:</strong> Name, email, and organization details for recruiter accounts.</li>
        <li><strong>Candidate data:</strong> CVs, interview transcripts, evaluation scores, and proctoring events submitted during the interview process.</li>
        <li><strong>Usage data:</strong> API logs, webhook delivery logs, and system health metrics.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">2. How We Use Your Data</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li>To provide the AI interview and evaluation service.</li>
        <li>To generate scorecards and hiring recommendations.</li>
        <li>To detect and prevent cheating during interviews.</li>
        <li>To deliver webhook events to your configured endpoints.</li>
        <li>To maintain system security and audit trails.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">3. Data Security</h2>
      <p>All candidate data is encrypted at rest (AES-256) and in transit (TLS 1.3). We enforce multi-tenant isolation at the database level using PostgreSQL Row-Level Security. Access is controlled via role-based authentication (RBAC) with least-privilege database roles.</p>

      <h2 className="text-lg font-semibold text-foreground">4. Data Retention</h2>
      <p>Candidate data (CVs, transcripts, scores) is retained for 90 days after the last activity, then automatically purged. Webhook delivery logs are retained for 30 days. User accounts are retained until manually deleted.</p>

      <h2 className="text-lg font-semibold text-foreground">5. Your Rights</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li><strong>Access:</strong> Candidates can export all their data via the portal.</li>
        <li><strong>Erasure:</strong> Candidates can permanently delete their data via the portal.</li>
        <li><strong>Portability:</strong> Data export is available in JSON format.</li>
        <li><strong>Consent:</strong> Explicit consent is captured before each interview begins.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">6. Contact</h2>
      <p>For privacy-related inquiries, contact us at privacy@intivai.com.</p>
    </LegalLayout>
  )
}

export function TermsOfServicePage() {
  return (
    <LegalLayout title="Terms of Service" lastUpdated="August 2026">
      <h2 className="text-lg font-semibold text-foreground">1. Acceptance of Terms</h2>
      <p>By accessing or using Intivai, you agree to be bound by these Terms of Service. If you do not agree, do not use the service.</p>

      <h2 className="text-lg font-semibold text-foreground">2. Service Description</h2>
      <p>Intivai is an AI-powered technical interview platform that conducts live conversational interviews with coding assessments, generates evaluation scorecards, and manages the candidate screening workflow.</p>

      <h2 className="text-lg font-semibold text-foreground">3. Account Responsibilities</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li>You are responsible for maintaining the security of your account credentials.</li>
        <li>You must not share your account access with unauthorized individuals.</li>
        <li>You are responsible for all activity under your account.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">4. Candidate Consent</h2>
      <p>When inviting candidates to interview, you must ensure that candidates provide explicit consent for AI evaluation and telemetry collection. Intivai captures this consent automatically before each interview begins.</p>

      <h2 className="text-lg font-semibold text-foreground">5. Fair Use</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li>Interviews must be conducted ethically and in compliance with applicable employment laws.</li>
        <li>AI-generated evaluations are decision-support tools, not final hiring decisions.</li>
        <li>You must not use the service to discriminate against protected classes.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">6. Limitation of Liability</h2>
      <p>Intivai provides evaluations as advisory input. Final hiring decisions remain the sole responsibility of the recruiting organization. We are not liable for hiring outcomes based on AI-generated recommendations.</p>

      <h2 className="text-lg font-semibold text-foreground">7. Contact</h2>
      <p>For questions about these terms, contact us at legal@intivai.com.</p>
    </LegalLayout>
  )
}

export function SecurityPage() {
  return (
    <LegalLayout title="Security" lastUpdated="August 2026">
      <h2 className="text-lg font-semibold text-foreground">Infrastructure</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li><strong>Cloud:</strong> Hosted on Docker containers with isolated sandbox execution environments.</li>
        <li><strong>Database:</strong> PostgreSQL with Row-Level Security (RLS) enforced at the database kernel level.</li>
        <li><strong>Object Storage:</strong> MinIO with server-side encryption (SSE-S3) for candidate CVs.</li>
        <li><strong>Queue:</strong> Redis-backed Asynq for async task processing with at-least-once delivery.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">Encryption</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li><strong>At rest:</strong> AES-256 encryption for all candidate data (CVs, transcripts, scores).</li>
        <li><strong>In transit:</strong> TLS 1.3 for all API, WebSocket, and SMTP connections.</li>
        <li><strong>Webhook payloads:</strong> Signed with HMAC-SHA256 for verification.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">Authentication & Authorization</h2>
      <ul className="list-disc pl-5 space-y-1">
        <li><strong>Users:</strong> bcrypt password hashing, JWT tokens with 24h expiry, refresh token rotation.</li>
        <li><strong>Candidates:</strong> Passwordless OTP with SHA256 hashing, magic link single-use tokens.</li>
        <li><strong>RBAC:</strong> Role hierarchy (admin &gt; recruiter &gt; member) enforced at API and database layers.</li>
        <li><strong>Multi-tenancy:</strong> PostgreSQL RLS ensures complete data isolation between organizations.</li>
      </ul>

      <h2 className="text-lg font-semibold text-foreground">Code Execution Sandbox</h2>
      <p>Code submitted during interviews executes in isolated microVM sandboxes via gRPC with mutual TLS (mTLS) authentication. Sandboxes have no network access to the main application and are destroyed after each session.</p>

      <h2 className="text-lg font-semibold text-foreground">Incident Response</h2>
      <p>We maintain an incident response plan covering detection, triage, containment, communication, resolution, and post-mortem. Data breaches are reported to relevant authorities within 72 hours per GDPR Article 33.</p>

      <h2 className="text-lg font-semibold text-foreground">Reporting</h2>
      <p>To report a security vulnerability, email security@intivai.com. We aim to respond within 24 hours.</p>
    </LegalLayout>
  )
}
