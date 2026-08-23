# Intivai — Product Requirements Document (PRD) Summary

**Product Name:** Intivai  
**Category:** AI Technical Interview Platform  
**Document Status:** Production Baseline (M1–M4 Complete, UAT Aligned)  
**Target Quality Gate:** Revenue-Generating Enterprise Grade (Zero Mistakes, Strict Types, ADR-0001–ADR-0004)  
**Date:** 2026-08-20  

---

## 1. Executive Summary & Business Vision

### 1.1 Problem Statement
Traditional technical recruitment is fundamentally broken:
- **Lengthy Time-to-Hire:** Average technical hiring cycle spans 45+ days with up to 14 days lost in preliminary resume reviews and phone screens.
- **Engineering Bandwidth Drain:** Senior software engineers spend 15–20 hours per week conducting repetitive initial screening calls and live coding tests instead of shipping core product features.
- **Human Screening Inconsistency & Bias:** Resume skimming often introduces unconscious bias (school/brand bias) while missing strong non-traditional candidates.
- **Cheating & AI Ghostwriting in Take-Homes:** Take-home coding challenges suffer from pervasive LLM cheating without proctoring or authenticity visibility.

### 1.2 The Intivai Solution
Intivai is an AI-powered technical interview platform that unifies:
1. **Multi-Source CV Ingestion & Vector Semantic Matching:** Ingests PDF/DOCX resumes (including OCR scanned documents), vectorizes candidate profiles, and computes objective 5-dimension competency match scores against job rubrics.
2. **Autonomous Interactive AI Interviews:** Conducts real-time token-streamed chat and full-duplex WebRTC voice interviews tailored to specific CV gaps.
3. **Isolated Live Coding MicroVMs:** Embeds an in-browser Monaco IDE connected via mTLS gRPC to sandboxed Docker execution runtimes (Go, Python, TypeScript, Rust).
4. **Advisory Proctoring & Authenticity Telemetry:** Tracks tab switching, keystroke-to-paste ratios, away time, and window focus to generate tamper-evident audit trails.
5. **Hallucination-Free Candidate 360 Scorecards:** Delivers evidence-grounded scorecards where every score and hiring recommendation cites exact verbatim candidate quotes.
6. **Global Talent Passport:** Provides candidates with portable, cryptographically verified competency credentials, avoiding repetitive re-testing across hiring teams.

### 1.3 Key Value Proposition & ROI
- **80% Reduction in First-Round Screening Time** (< 48 hours from application to scored interview).
- **$38,000+ / Quarter in Engineering Hours Saved** for a 15-engineer hiring team.
- **100% Auditability & Compliance** (GDPR/EEOC compliant consent gates, SOC 2 Type II readiness, Row-Level Security).

---

## 2. Target Personas & Stakeholder Needs

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                     INTIVAI USER PERSONAS                                        │
├───────────────────────────────┬────────────────────────────────┬─────────────────────────────────┤
│ 1. Sarah — Head of TA (Admin) │ 2. Marcus — Technical Recruiter│ 3. Elena — Hiring Manager       │
│ - Requisition Governance      │ - High-Volume Sourcing         │ - Candidate 360 Scorecard Review│
│ - Brand Voice & AI Rails      │ - Single/Bulk Resume Ingest    │ - Verbatim Quote Validation     │
│ - Hiring Velocity & ROI Stats │ - Semantic Match Verification  │ - Coding Sandbox Inspection     │
│ - Tenant Security & RBAC      │ - Dispatching AI Invitations   │ - Hiring Verdict & Advancement  │
├───────────────────────────────┴────────────────────────────────┴─────────────────────────────────┤
│ 4. Alex — Software Candidate / Job Seeker                                                        │
│ - 1-Click Public Careers Portal Application                                                      │
│ - Passwordless OTP & Magic Link Candidate Portal Access                                          │
│ - Multi-Stage Application Stepper Tracking                                                       │
│ - Real-Time Chat & Voice Interview Assessment + Live Coding Sandbox                              │
│ - Portable Global Talent Passport Management                                                     │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Core Functional Requirements (Epics 1–10)

### Epic 1: Requisition & Competency Rubric Management
- **2-Step Creation Flow:** Step 1 captures Title, Department, Seniority, Location, Employment Type, and explicit Compensation Bands (min/max/currency). Step 2 configures the multi-stage pipeline and minimum passing score threshold.
- **Scoring Dimensions Configuration:** Support custom weightings across 5 key dimensions:
  - *Hard Skills Match* (Default: 35%)
  - *Experience Fit & Seniority* (Default: 20%)
  - *Vector Semantic Cosine Alignment* (Default: 25%)
  - *Education & Academic Foundation* (Default: 10%)
  - *Certifications & Domain Accreditations* (Default: 10%)
- **Lifecycle Status Control:** Instant publish/unpublish toggle. Unpublished roles are hidden from the public career board.

### Epic 2: Multi-Source CV Ingestion & Parsing
- **Format Ingestion:** Pure-Go PDF parser (`ledongthuc/pdf`) with fallback to Poppler (`pdftoppm`) + Tesseract OCR for image-only or scanned PDFs.
- **Bulk Upload Support:** Single-candidate modal and bulk batch upload supporting up to 50 concurrent PDF resumes.
- **Async Queue Pipeline:** Ingestion dispatches `cv:parse` -> `cv:extract` Asynq background workers. Structured profile extraction extracts Name, Email, Phone, Years of Experience, Skills array, Work Experience, and Education objects.

### Epic 3: AI Resume Screening & Semantic Matcher
- **Vector Cosine Similarity:** Embeds job descriptions and candidate CVs into dense 384-dimensional vector space (`multi-qa-MiniLM-L6-cos-v1`) via pure-Go fast inference.
- **Competency Gap Synthesis:** Computes match score (0–100) against requisition threshold. Identifies missing requirements (e.g. *Candidate lacks Rust and Kubernetes production experience*) to inform the interview question generator.
- **Candidate Self-Review Portal:** Direct magic link (`/candidate-review/:token`) allowing candidates in `pending_review` status to verify and correct extracted profile data.

### Epic 4: Candidate Experience & Frictionless Authentication
- **Public Career Board (`/careers`):** Search by keyword, department, location, or skill. Modal job details and 1-click resume submission.
- **Passwordless OTP Portal (`/candidate/portal`):** Candidates authenticate via 6-digit SHA256 hashed OTP code (`123456` in demo) or single-use magic link.
- **Application Stepper:** Real-time visibility into application stage: `Applied` -> `Screening Passed` -> `AI Interview Ready` -> `Completed`.

### Epic 5: AI Interview Execution (Voice & Chat + Sandbox)
- **Dynamic CV-Gap Questioning:** LLM synthesizes tailored technical challenges targeting specific resume gaps.
- **Token-Streamed Chat Room (`/chat/:id`):** Single-writer WebSocket architecture streaming LLM tokens in real-time. Candidate responses advance the state machine.
- **WebRTC Voice Room (`/voice/:id`):** Full-duplex voice streaming with Whisper STT and Edge Neural speech synthesis. Visual audio orb waveform.
- **Embedded Coding Sandbox (Monaco IDE):** Split-view code editor supporting Python, Go, TypeScript, and Rust. Communicates with sandboxed execution sidecars via mTLS gRPC.
- **Streaming Controls:** Interrupt button (`session.interrupt()`) allowing candidates to skip AI speech and formulate immediate answers.

### Epic 6: Proctoring, Authenticity & Anti-Cheat Telemetry
- **Advisory Telemetry Tracking:** Client-side proctoring engine (`useProctoring`) tracking:
  - Tab switching count & cumulative away duration.
  - Paste events with character counts and typing pacing ratios.
  - Window focus loss and inactivity timer gates.
- **Anti-Cheat Philosophy (ADR-0004):** Proctoring data is strictly advisory. Events are displayed in the recruiter audit trail without auto-failing candidates, preserving human recruiter judgment.

### Epic 7: Comprehensive Candidate 360 Scorecards & Decisions
- **Executive Scorecard (`/interviews/:id`):**
  - Overall Performance Score (0–100) & AI Recommendation Badge (*Strong Hire*, *Hire*, *Consider*, *Reject*).
  - Competency radar & category breakdown.
  - Proctoring Authenticity Audit score.
- **Verbatim Quote Grounding:** Anti-hallucination guarantee. Every evaluation metric cites exact quotes from the interview transcript.
- **Recruiter Decision Logging:** Hiring managers can record hiring verdicts, add interview notes, and advance candidates to final offer stages.

### Epic 8: Global Talent Passport & Verified Credentials
- **Candidate Skill Portability:** Completed interviews issue a verifiable Global Talent Passport with percentile rankings and skill badges.
- **Cross-Organization Verification:** Enables hiring teams to fast-track pre-verified talent.

### Epic 9: Company Intelligence & Custom AI Interview Rails
- **Custom System Prompting (`/company-context`):** TA leaders define organization culture, technical interview philosophy, and tone of voice.
- **Safety Rails & Prompt Injection Defense:** `ContainsInjection` rails filter malicious candidate inputs before LLM execution, preserving evaluation objectivity.

### Epic 10: TA Analytics, Funnel Operations & ROI
- **Command Center Dashboard (`/dashboard`):** Real-time KPI summary (Active Roles, Ingested CVs, Screening Pass Rate, Interviews Run).
- **Pipeline Velocity Funnel:** Visual stage stepper displaying conversion rates across the hiring funnel with 1-click drill-down.
- **ROI Savings Calculator (`/#calculator`):** Interactive model demonstrating hours and budget saved based on monthly interview volume.

---

## 4. Technical Architecture & Architecture Decision Records (ADRs)

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                   INTIVAI MODULAR ARCHITECTURE                                   │
├────────────────────────────────┬────────────────────────────────┬────────────────────────────────┤
│ FRONTEND (React 19 SPA)        │ BACKEND (Go 1.24 Modular Mono) │ INFRASTRUCTURE & SIDE-CARS     │
│ - Vite 8 + Tailwind CSS + Radix│ - Fiber HTTP API (/api/v1)     │ - PostgreSQL 16 + pgvector RLS │
│ - Monaco Code Editor           │ - Realtime WS & WebRTC Engine  │ - Redis 7 (Asynq Queue & RL)   │
│ - TanStack Query + Sonner      │ - Asynq Background Workers     │ - MinIO S3 Object Storage      │
│ - Phosphor & Lucide Icons      │ - LLM Provider Port   │ - Sandboxd gRPC MicroVMs       │
└────────────────────────────────┴────────────────────────────────┴────────────────────────────────┘
```

### 4.1 Key Architecture Decisions (ADRs)

1. **ADR-0001: Two-Dimensional Candidate State Machine**
   - Disentangles internal processing lifecycle (`status`: `new` -> `parsing` -> `extracting` -> `extracted` / `pending_review` -> `failed_*`) from the business recruiting workflow (`stage`: `applied` -> `screening` -> `interview_invited` -> `interview_completed` -> `offered` / `rejected`).
   - Prevents UI state deadlocks and race conditions.

2. **ADR-0002: Sandboxed Code Execution Runtime**
   - Language-specific runner images (`intivai-sandbox-go`, `python`, `typescript`, `rust`) isolated with non-root execution, 256MB RAM caps, 5s execution timeouts, and disabled network access.
   - Core server communicates with `sandboxd` daemon over mutual TLS (mTLS) gRPC.

3. **ADR-0003: Speech Architecture (Whisper STT + Kokoro/Edge TTS)**
   - WebRTC audio stream decoded via low-latency Whisper STT engine. Responses synthesized via Edge Neural TTS with real-time waveform visualization.

4. **ADR-0004: Comprehensive Evaluation & Proctoring Data Model**
   - Canonical `reports` table storing `overall_score`, `recommendation`, `competencies` JSONB, `quotes` JSONB, and `proctoring_summary` JSONB.
   - Advisory-only proctoring posture guarantees fair, human-in-the-loop hiring decisions.

---

## 5. Non-Functional Requirements (NFRs)

| NFR Category | Requirement Specification |
|---|---|
| **Performance & Latency** | - Public pages load in < 500ms.<br>- First LLM streaming token delivered in < 1.5s.<br>- CV extraction completed in < 30s.<br>- Code sandbox execution returns in < 3s. |
| **Security & Privacy** | - Multi-tenant isolation enforced via PostgreSQL Row-Level Security (`FORCE RLS`) with `intivai_app` least-privilege DB role.<br>- Cross-Site WebSocket Hijacking (CSWSH) protection with `INTIVAI_ALLOWED_ORIGINS` guard.<br>- Passwordless candidate authentication with SHA256 OTP hashing and 7-day TTL tokens.<br>- Prompt injection validation on all user/tenant inputs.<br>- **Roadmap:** AES-256 encryption at rest for all candidate data (CVs, transcripts, scores) via PostgreSQL `pgcrypto` and MinIO SSE-S3 (planned, not yet implemented).<br>- TLS 1.3 in transit for all API and WebSocket connections. |
| **Reliability & Availability** | - Asynq background queue with automatic retry backoff and idempotency guards.<br>- Fail-safe error handling; zero raw error leakage in API responses.<br>- Nightly automated DB & MinIO disaster recovery backup scripts. |
| **Scalability & Concurrency** | - Single Go binary handles 100+ concurrent real-time WebSocket interview connections.<br>- Redis sliding-window rate limiting on all public and authenticated endpoints. |
| **Code Quality & Testing** | - Strict Test-Driven Development (TDD) for all backend and frontend changes.<br>- Per-package test coverage floors: Domain ≥ 70%, others ≥ 50%.<br>- Strict types enforced: `any` strictly forbidden in TypeScript and Go. |

---

## 6. Pricing Model

| Tier | Price | Limits | Target |
|------|-------|--------|--------|
| **Free** | $0 | 10 interviews/mo, chat only | Trial evaluation |
| **Starter** | $29/mo | 100 interviews | Small teams |
| **Pro** | $199/mo | 1,000 interviews | Growing companies |
| **Interview Credits** | $99 | 500 credits (1 credit = 1 interview) | Volume BPO |
| **Volume** | $0.50/interview | Usage-based | High-volume users |
| **Enterprise** | Custom | Self-hosted | Data-sensitive orgs |

Credits expire after 12 months. Volume pricing: $0.20/credit for 1,000+ credits.

---

## 7. Testing & Quality Assurance Summary

The Intivai platform is governed by two complementary testing matrices:
1. **System Integration Testing (SIT) Matrix (`sit_test_matrix.md`):** 58 technical test cases validating API contracts, RLS isolation, Asynq pipeline transitions, and real-time WebSocket protocol framing.
2. **HR User Acceptance Testing (UAT) Matrix (`uat_test_matrix.md`):** 60+ acceptance scenarios structured across 10 HR Business Epics validating candidate experience, recruiter efficiency, anti-cheat audit trails, and scorecard accuracy.

### Current Test Suite Status (100% Green Gate Check)
- **Automated Playwright SIT Suite (`sit-matrix.spec.ts`):** **7/7 Passed (100%)**
- **Automated Pages & Flow Validation:** **11/11 Passed (100%)**
- **Frontend Unit & Component Tests (`vitest run`):** **18/18 Passed (100%)**
- **Backend Quality Gate (`make check`):** **gofmt + golangci-lint + go vet + go build + go test: PASSED**
