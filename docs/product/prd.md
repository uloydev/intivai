# Intivai — Product Requirements Document

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Positioning per `docs/reviews/2026-08-22-competitive.md`: **AI Technical
> Interview Platform — not an ATS.** Project completion status lives ONLY in
> [`../engineering/beta-gate.md`](../engineering/beta-gate.md); this document
> makes no completion claims. Pricing lives in [`pricing.md`](pricing.md).

**Category:** AI Technical Interview Platform
**Target quality gate:** revenue-generating production grade (strict types, TDD, ADR-driven)

---

## 1. Problem & Solution

### 1.1 Problem Statement

Traditional technical recruitment is slow and inconsistent:

- **Lengthy time-to-hire:** technical hiring cycles commonly span 45+ days, with up to 14 days lost to resume reviews and phone screens.
- **Engineering bandwidth drain:** senior engineers spend hours per week on repetitive screening calls instead of shipping.
- **Inconsistent, biased screening:** resume skimming introduces brand/school bias and misses strong non-traditional candidates.
- **Cheating in take-homes:** unproctored take-home challenges suffer pervasive LLM ghostwriting.

### 1.2 The Intivai Solution

Intivai conducts **live, conversational AI technical interviews with in-browser coding assessments**, then hands recruiters an evidence-grounded scorecard:

1. **Multi-source CV ingestion & semantic matching:** PDF ingestion (OCR fallback for scans), vectorized profiles, transparent 5-dimension weighted match scores against job rubrics.
2. **Autonomous interactive interviews:** real-time token-streamed chat (voice as a gated demo — see ADR-0006) with questions targeted at specific CV gaps.
3. **Isolated live coding sandbox:** Monaco IDE connected via mTLS gRPC to sandboxed Docker runtimes (Go, Python, TypeScript, Rust) — ADR-0002.
4. **Advisory proctoring telemetry:** tab switches, paste ratios, away time — displayed to recruiters only, never auto-failing (ADR-0005).
5. **Evidence-grounded scorecards:** every score cites verbatim candidate quotes; the final number is recomputed by domain code, never set by the LLM.
6. **Global Talent Passport (post-beta):** portable verified competency credentials.

### 1.3 Value Hypotheses (to be validated by the beta cohort — not claims)

| Hypothesis | Validation metric | Measured by |
|---|---|---|
| Cuts first-round screening time substantially | Hours from application → scored interview; recruiter self-reported hours saved | Beta cohort funnel data + interviews |
| Recruiters trust AI scorecards when evidence is shown | % of scorecards opened with transcript viewed; override rate | Product analytics |
| Live conversational + coding beats async take-homes for signal | Recruiter-rated interview usefulness vs previous process | Pilot feedback surveys |

Pilot economics model (engineering-hours-saved ROI) ships as the on-site
calculator; numbers shown there are illustrative until pilot data replaces them.

---

## 2. Target Personas & Stakeholder Needs

```
┌───────────────────────────────┬────────────────────────────────┬─────────────────────────────────┐
│ 1. Sarah — Head of TA (Admin) │ 2. Marcus — Technical Recruiter│ 3. Elena — Hiring Manager       │
│ - Requisition governance      │ - High-volume sourcing         │ - Candidate 360 scorecard review│
│ - Brand voice & AI rails      │ - Single/bulk resume ingest    │ - Verbatim quote validation     │
│ - Hiring velocity & ROI stats │ - Semantic match verification  │ - Coding sandbox inspection     │
│ - Tenant security & RBAC      │ - Dispatching AI invitations   │ - Hiring verdict & advancement  │
├───────────────────────────────┴────────────────────────────────┼─────────────────────────────────┤
│ 4. Alex — Software Candidate / Job Seeker                       │                                 │
│ - 1-click public careers portal application                     │ - Mobile-first interview flow   │
│ - Passwordless OTP & magic link candidate portal access         │ - Transparent AI disclosure     │
│ - Multi-stage application stepper tracking                      │ - Data access/erasure rights    │
└─────────────────────────────────────────────────────────────────┴─────────────────────────────────┘
```

---

## 3. Goals & Non-Goals

### Goals (beta)

- Close the loop: application → screening → live AI interview → evidence-grounded report → recruiter decision.
- Make every AI-produced number traceable (weights used, context version pinned, verbatim quotes) or honestly absent ("not evaluated").
- Zero-friction candidate entry (public board, passwordless portal, mobile-first chat).
- Tenant control where it matters (scoring weights, company context, prompt rails) with locked safety rails.

### Non-Goals (explicitly out of scope)

- **Not an ATS:** no sourcing, job-board posting, CRM, pipeline kanban, offer management, or onboarding. Intivai integrates into existing ATS workflows (webhooks — Epic 9); it does not replace them.
- No mobile native apps (responsive web only).
- No voice interviews on the beta critical path (gated demo only — ADR-0006).
- No SOC 2 certification claim during beta; prep tracked separately (`docs/compliance/soc2_readiness.md`).
- No billing/pricing enforcement in code during beta (manual invoicing; tiers defined in pricing.md).

---

## 4. Core Functional Requirements (Epics)

Priority key: **M** = must-have beta · **S** = should-have · **C** = could-have · **W** = won't-have (beta).

### Epic 1: Requisition & Competency Rubric Management — M
- Job creation with compensation bands, publish/unpublish visibility control (`is_published`), multi-stage pipeline config.
- Configurable scoring weights across 5 dimensions (defaults and resolution rules specified once in `docs/engineering/design-decisions.md` §1.4–1.5).
- Publish state controls public careers-board visibility.

### Epic 2: Multi-source CV Ingestion & Parsing — M
- PDF parsing with Poppler + Tesseract OCR fallback for scanned documents.
- Bulk upload (up to 50 concurrent), Asynq pipeline `cv:parse` → `cv:extract`, honest failure states (`failed_parse`/`failed_ocr`/`failed_extract`) with retry.
- DOCX support — S (post-beta unless a pilot blocks on it).

### Epic 3: AI Screening & Semantic Matching — M
- Embedding cosine similarity (384-dim local embeddings) plus keyword skills match; weighted composite score vs per-job threshold.
- Missing-requirement synthesis feeding the interview question generator.
- Candidate self-review portal for extracted profile corrections — S.

### Epic 4: Candidate Experience & Frictionless Authentication — M
- Public careers board with search/filter.
- Passwordless OTP + magic-link candidate portal.
- Application stepper with human-readable stages; honest empty/error states.

### Epic 5: AI Interview Execution (Chat + Sandbox) — M
- CV-gap-targeted questioning, token-streamed WS chat, interrupt, reconnect/resume, single-active-session enforcement.
- Embedded Monaco coding sandbox against mTLS gRPC sidecar runtimes — M for tech roles (S if no pilot needs Rust/TS at launch).
- Voice room — W for beta (gated demo exists; ADR-0006).

### Epic 6: Proctoring & Anti-cheat Telemetry — M (advisory-only posture)
- Client-side telemetry (tab switch, paste ratio, away time, focus loss), retention-capped raw events, recruiter-only audit trail.
- Never auto-fails; labeled "unverified" with a flag-for-human-review path. Rationale: ADR-0005.

### Epic 7: Candidate 360 Scorecards & Decisions — M
- Overall score + recommendation badge, dimension breakdown, per-question rationale with verbatim quotes, proctoring summary.
- Recruiter decision logging incl. override-with-reason of the AI recommendation; full transcript on the scorecard.

### Epic 8: Global Talent Passport — W (post-beta)
- Portable verified credentials; cut from MVP scope per four-perspective review consensus (unproven demand). Revisit after pilot feedback.

### Epic 9: Company Context & Custom AI Rails — M
- Versioned tenant company context + system prompt with hash dedup and version pinning on interviews; injection rails; safety rails hard-pinned last (non-overridable).

### Epic 10: Analytics, Funnel & ROI — S
- KPI dashboard (active roles, ingested CVs, pass rate, interviews run), pipeline funnel with drill-down.
- Full reporting suite — post-beta (competitive gap acknowledged; see review §2.5).

### Epic 11: Integrations — S (beta blocker for ATS-heavy pilots)
- Outbound generic webhooks (`interview.completed`, `candidate.screened`, `candidate.advanced`) with HMAC-SHA256 signing, retries, delivery logs.
- Native ATS integrations (Greenhouse/Lever) — post-beta.

---

## 5. Technical Architecture (summary)

Modular monolith (Go/Fiber) + React SPA + PostgreSQL/pgvector (RLS FORCE,
least-privilege app role) + Redis/asynq + MinIO + sandbox gRPC sidecars.
Details: `docs/engineering/architecture.md`. Decisions: ADR-0001 (status vs
stage), ADR-0002 (sandbox sidecar), ADR-0003 (speech stack), ADR-0004
(evaluation/proctoring data model), ADR-0005 (advisory proctoring), ADR-0006
(voice scope), ADR-0007 (backup offsite deferral).

## 6. Non-functional Requirements

| NFR | Requirement | Status |
|---|---|---|
| Latency | First LLM streaming token p95 < 1.5s; CV extraction < 30s; sandbox exec < 3s; public pages < 500ms | Targets — actuals recorded in `docs/engineering/slos.md` |
| Mobile | Candidate interview flow must be fully usable at 375px viewport; touch targets ≥44px | Required before beta |
| Security | RLS FORCE multi-tenancy; CSWSH origin allowlist; hashed OTPs; prompt-injection rails; TLS 1.3 in transit | Implemented (see threat model) |
| Encryption at rest | AES-256 for candidate data (pgcrypto column-level + MinIO SSE-S3) | **Target Q4 2026 — not yet implemented** |
| Reliability | Queue retry backoff with idempotency guards; nightly DB+object-store backups; restore drill monthly | Backup on-server now; offsite copy gated by ADR-0007 (hard date 2026-09-30) |
| Cost control | Explicit task retry budgets; per-org LLM token caps; cost ledger at the LLM port | Partial — see threat model + llm-outage runbook |
| Code quality | TDD; coverage floors (domain ≥70%, others ≥50%); strict types (no `any`) | Enforced by `make check` |

## 7. Customer Validation (beta cohort)

- Cohort: 5 pilots (target mix: tech startups hiring 5–20/mo; volume BPO as secondary segment).
- Entry criteria: signed pilot agreement, weekly feedback call slot, defined success metric per pilot (screening hours saved, candidate completion rate).
- Feedback channels: shared channel + structured weekly survey.
- Exit criteria: documented decision per pilot (convert / extend / churn reason) feeding pricing.md and roadmap re-prioritization.

## 8. Pricing

Single source: [`pricing.md`](pricing.md).

## 9. Testing & Quality Assurance

Current gate status is tracked ONLY in [`../engineering/beta-gate.md`](../engineering/beta-gate.md)
(counted on tagged releases). Test matrices:
`sit-matrix` (technical cases) and `uat-matrix` (HR acceptance scenarios) live
in `docs/engineering/test-matrices/`.
