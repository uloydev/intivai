# AI Interviewer — Development Phases

## Dependency Map

```
Phase 0: Customer Discovery ◄── VALIDATE BEFORE CODING
    │
    ▼
Phase 1: Foundation ◄── everything depends on this
    │
    ▼
Phase 2: Core Business Logic (CV, Job, Screening)
    │
    ▼
Phase 3: Chat Interview (WebSocket + LLM)
    │
    ▼
Phase 4: Evaluation & Reports
    │
    ▼
Phase 5: Voice MVP, Passports, Bulk CV, Public Board (Product Pivot)
    │
    ▼
Phase 6: Production Polish (Observability, Deploy, Scale)
```

---

## Phase 0: Customer Discovery (Week 1-2) — VALIDATION RUNS THROUGH BETA

**Goal:** Validate the problem + get 5 pilot customers. Talking-first still
stands, but validation does not END at Phase 0 — the 5 pilots become the beta
cohort: their feedback loop (onboarding, usage, interview quality, pricing
willingness) is the primary launch signal. Beta = validation vehicle.

### Deliverables

| Activity | What | Output |
|-----------|-----|--------|
| **Interview 10 recruiters/HR** | Agency recruiters + BPO + tech startups (Jakarta-Bandung, Hyperscal/OCBC network) | Pain points verified, 5 pilot interests |
| **Problem validation** | "What's the most painful problem in your screening/interview?" — don't ask about features | Answer "interview process slow/biased/inconsistent" = validation, not a feature ask |
| **Pilot commitment** | 5 companies agree to try free for 1 month (beta cohort) | Pilot list + contacts + feedback channel |
| **Pricing validation** | Ask: "to screen 100 candidates, how much would you pay?" | Real pricing numbers from the market |
| **Scope trim** | Features pilots didn't ask for → cut from MVP | MVP scope final |

### Exit Criteria (do not start Phase 1 before this)
- [ ] 10+ recruiter/HR interviews done
- [ ] Interview process problem confirmed (not a guess)
- [ ] 5 pilot customers committed (beta cohort)
- [ ] Pricing validated (not assumptions)

---

## Phase 1: Foundation (Week 3-4)

**Goal:** Project scaffold, shared kernel, auth, multi-tenant, CI pipeline

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Shared kernel** | Base entity, aggregate root, value object, domain events | `shared/domain/` |
| **Config** | Viper config loader, env vars | `pkg/config/config.go` |
| **Logger** | Zerolog structured logging | `pkg/logger/zerolog.go` |
| **Database** | PostgreSQL connection pool (GORM over pgx stdlib), migration tool setup | `pkg/db/postgres.go`, `pkg/db/migrations/001_init.up.sql` |
| **Queue** | Asynq setup + worker skeleton | `pkg/queue/asynq.go` |
| **Storage** | MinIO/S3 client skeleton | `pkg/storage/minio.go` |
| **Memory layer (Mnemosyne port)** | MemoryBank port + adapter (native SQLite+fastembed default; MCP optional), bank per tenant, sync worker skeleton | `internal/memory/` |
| **IAM** | Org, User entities + Auth (JWT), RBAC, multi-tenant middleware | `internal/iam/` |
| **API scaffold** | Fiber app, middleware (auth, tenant, CORS, rate limit), health check | `cmd/server/main.go` |
| **DevOps** | Docker Compose (Go app + Postgres + Redis + MinIO) | `docker-compose.yml` |
| **CI** | GitHub Actions: lint (golangci-lint), vet, build, test, coverage gate, integration (postgres+redis services), end-to-end smoke | `.github/workflows/ci.yml` |

### Key Decisions Made
- Framework: Fiber
- Database: PostgreSQL + pgvector (source of truth) — accessed via GORM over pgx stdlib; migrations via golang-migrate
- Queue: Asynq (Redis)
- Auth: JWT + per-tenant RBAC
- File storage: MinIO (S3-compatible)
- **Hybrid memory: MemoryBank port (1 SQLite bank per tenant) — native Go adapter by default (fastembed, $0); MCP optional. Reflect = small LLM call at query-time. Postgres pgvector bank adapter exists (swap via `INTIVAI_MEMORY_DRIVER=postgres`)**
- PDF text extraction: `ledongthuc/pdf` (pure Go); scanned PDFs → `pdftoppm` (poppler) rasterize + Tesseract OCR

### Testing Criteria
- [x] Unit tests pass (make check)
- [x] Auth middleware rejects unauthenticated requests (rejects ws_ticket tokens too)
- [x] Tenant isolation: User A cannot see User B's data (RLS FORCE + intivai_app least-privilege role, integration-tested)
- [x] Docker Compose starts all services (migrate service + app + postgres + redis + minio)
- [x] Health endpoint returns 200
- [x] Per-tenant Mnemosyne bank created + recall isolated across tenants (sqlite + pgvector banks, integration-tested)

---

## Phase 2: Core Business Logic (Week 5-7)

**Goal:** CV upload + parsing, job management, scoring engine, company context

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Job context** | Job entity, skill requirements, create/update/list | `internal/job/` |
| **CV context** | CV upload, PDF text extraction (`ledongthuc/pdf`), DeepSeek structured extraction, embedding (deferred — column + HNSW ready) | `internal/cv/` |
| **Screening context** | Scoring engine, weighted algorithm, passing threshold | `internal/screening/` |
| **Company context** | Upload file/text, versioning, hash dedup, index into the tenant's Mnemosyne bank | `internal/context/` |
| **Tenant prompt** | Set/get tenant system prompt, validation, versioning, default fallback | `internal/context/` |
| **Async workers** | ParseCV worker, ExtractCV worker, ScoreCV worker, IndexContext worker | `internal/cv/application/` + `internal/screening/application/` + `internal/context/application/` |
| **API endpoints** | `POST /cvs`, `GET /cvs/:id`, `POST /jobs`, `GET /jobs`, `POST /screenings`, `POST /orgs/:id/contexts`, `PUT /orgs/:id/prompt` | `api/` in each context |

### CV Flow (End-to-End)
```
POST /api/v1/cvs (upload PDF)
  → file saved to MinIO
  → queue: parse_cv
    → extract text (ledongthuc/pdf)
    → if scanned → pdftoppm rasterize + Tesseract OCR
    → save raw text
  → queue: extract_cv
    → DeepSeek structured output → ResumeData
    → save structured (+ embedding when local fastembed lands, M2.5)
  → queue: score_cv (for each active JD)
    → weighted scoring algorithm
    → save score + breakdown
  → queue: sync_mnemosyne (async, non-blocking)
    → remember candidate_profile into banks/<org_id>/mnemosyne.db
    → semantic candidate recall ready to use
```

### Semantic Sync to Mnemosyne (Phase 2)

CV/extract done → worker syncs a semantic summary into the tenant bank:

```go
// internal/memory/application/sync_worker.go
func (s *SyncWorker) SyncCandidate(ctx context.Context, orgID, candidateID, summary string) error {
    mn := s.mnemosyne.ForBank(orgID) // banks/<org_id>/mnemosyne.db
    return mn.Remember(ctx, "candidate_profile", summary, 0.9)
}
```

What gets indexed: semantic summaries (not raw PII) — cross-candidate recall becomes possible.

### Testing Criteria (tambahan Phase 2)
- [x] Sync worker writes to the correct tenant bank (integration-verified)
- [x] Recall finds matching candidates — keyword overlap verified; embedding recall deferred until fastembed lands (M2.5)
- [x] Company context upload → version bump + dedup + index into tenant bank
- [x] Tenant prompt set → validation rejects prompt injection (max length, forbidden keywords)
- [x] Tenant without prompt → falls back to global default
- [x] Context version pinned at interview time (audit traceable) — migration 007, integration-tested

---

### Scoring Engine
```go
// Implemented weights (internal/screening/domain/scoring.go). Total is
// normalized to a 0-100 scale (weights sum to 1.0, threshold default 50).
type ScoringWeights struct {
    SkillsMatch      float64 // 0.35
    ExperienceYears  float64 // 0.20
    SemanticMatch    float64 // 0.25 (keyword overlap now; embedding cosine when fastembed lands, M2.5)
    Education        float64 // 0.10
    Certifications   float64 // 0.10
}

type ScoreResult struct {
    Total      float64            `json:"total"`
    Breakdown  map[string]float64 `json:"breakdown"`
    Passed     bool               `json:"passed"`
}
```

### Testing Criteria
- [x] PDF upload → extracted text + structured data obtained (parse verified live; extract needs `INTIVAI_DEEPSEEK_API_KEY`, failure state `failed_extract` honest + `POST /cvs/:id/extract` retry)
- [x] CV score matches expected weighted calculation (unit + integration verified)
- [x] Same CV scored against 2 different JDs produces different scores (engine unit-tested)
- [x] Async jobs complete successfully (parse → extract → score pipeline live-verified; failure paths integration-tested)
- [x] Score threshold correctly filters candidates (job > org > default 50, unit-tested)

---

## Phase 3: Chat Interview (Week 8-10)

**Goal:** Real-time chat interview via WebSocket with DeepSeek Flash

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Interview domain** | Interview entity, Question VO, Answer VO | `internal/interview/domain/` |
| **LLM provider** | DeepSeek adapter (`deepseek-chat`, streaming + structured output) — shared in `internal/llm/` | `internal/llm/` |
| **Question generator** | CV-gap-based question strategy, bias detection | `internal/interview/domain/service/question_generator.go` |
| **Prompt composer** | Compose interview system prompt: global default + tenant prompt + company context + safety rails (pinned last) | `internal/interview/domain/service/prompt_composer.go` |
| **WebSocket hub** | Connection management, heartbeat, room/channel per interview | `internal/interview/api/chat_handler.go` |
| **Context management** | Sliding window, token counting via tiktoken-go | `internal/interview/domain/service/` |
| **API endpoints** | `POST /interviews` (recruiter), `POST /candidate/interviews/:id/ticket` (invitation → WS ticket), `WS /candidate/interviews/:id/chat` (ticket auth) | `internal/interview/api/` |
| **Reconnection** | Store last answered question, allow resume | `internal/interview/application/` |
| **Candidate chat UI (FE)** | Browser WS client: ticket connect, answer/stream, interrupt, resume, consent checkbox at start | `frontend/chat.tsx` |

### Chat Flow
```
┌─────────┐  WebSocket   ┌──────────────┐   HTTP/SSE    ┌──────────────┐
│ Browser │ ◄──────────► │  Go Server   │ ◄──────────► │ DeepSeek     │
│ (React) │              │  (Fiber+WS)  │              │ Flash API    │
└─────────┘              └──────┬───────┘              └──────────────┘
                               │
                        ┌──────▼───────┐
                        │  PostgreSQL  │
                        │  (messages,  │
                        │   state)     │
                        └──────────────┘
```

### WebSocket Protocol
```json
// Server → Client
{"type": "interview.start", "session_id": "iv_abc", "total_questions": 5}
{"type": "question", "content": "Tell me about a time...", "idx": 1}
{"type": "token", "content": "Great"}    // streaming LLM response
{"type": "token", "content": " answer"}
{"type": "response", "content": "Great answer! Next question..."}
{"type": "evaluation", "scores": {...}}
{"type": "error", "message": "..."}

// Client → Server
{"type": "answer", "content": "In my previous role...", "idx": 1}
{"type": "interrupt"}     // stop AI mid-response
{"type": "ping"}
{"type": "resume", "session_id": "iv_abc"}  // reconnect
```

**Auth:** candidate upgrade uses a WS ticket (10-min JWT, bound to session_id + interview_id) — see Research §3. Internal JWT is NOT used for candidates.

### Testing Criteria
- [x] WebSocket connects and handshake completes (candidate uses WS ticket, not JWT)
- [x] WS upgrade without a valid ticket is rejected
- [x] DeepSeek Flash returns streaming response (live-verified with real key)
- [x] Questions generated based on CV gaps
- [x] Selected questions persisted to the question bank (reuse + audit)
- [x] Answers stored in PostgreSQL
- [x] Reconnection resumes from last unanswered question (resume re-sends start + current question; session mismatch rejected)
- [x] Bias detection catches prohibited questions
- [x] Idle timeout disconnects after 5 minutes (ws read deadline; clock injectable)
- [x] 100 concurrent WebSocket connections stable (cmd/loadcheck: 100/100 pass)
- [x] Browser candidate interview runs end-to-end (Playwright happy path, real DeepSeek: register → job → CV → extract → passed → interview → invite → consent → chat → streamed reply)
- [x] System prompt composer: tenant prompt + company context + safety rails composed correctly (composed once per connection)
- [x] Safety rails always last (tenant cannot override)
- [x] Interrupt stops the AI mid-response (streaming goroutine + ctx cancel; live-verified)
- [x] Single active connection per interview (second socket rejected with error frame)

---

## Phase 4: Evaluation & Reports (Week 11-12)

**Goal:** Post-interview evaluation, report generation, recruiter visibility.
SPLIT for beta: P4a is part of the MVP — the interview loop is not closed
without an evaluation outcome (the P3 `evaluation` frame currently sends
empty scores). P4b stays post-MVP.

### P4a — Beta/MVP: evaluation core + recruiter visibility

| Area | What | Files |
|------|------|-------|
| **Evaluation domain** | Report entity, criteria, scoring | `internal/evaluation/domain/` |
| **LLM evaluation** | Per-question scoring → structured report (fills the P3 `evaluation` frame + persisted) | `internal/evaluation/infrastructure/llm/` |
| **Report generation** | Aggregate per-question scores → report JSON | `internal/evaluation/application/generate_report.go` |
| **API endpoints** | `GET /interviews/:id` (answers, status, scores), `GET /candidates/:id/report` (JSON) | `internal/evaluation/api/` |
| **Recruiter dashboard-lite (FE)** | Job + CV upload, interview list, per-candidate result view | `frontend/pages/` |
| **Invite flow (FE+BE)** | Shareable interview invite URL from the invitation token | `frontend/pages/` + `internal/interview/api/` |
| **Consent capture (FE+BE)** | `consent_given` recorded at interview start | `frontend/chat.tsx` + interview api |

### P4b — Post-MVP

| Area | What | Files |
|------|------|-------|
| **PDF report generation** | Downloadable PDF (skip in MVP — JSON first) | `internal/evaluation/application/pdf.go` |
| **Semantic index (interview)** | Sync interview_summary + reflection into the tenant's Mnemosyne bank | `internal/memory/application/` |
| **Cross-interview reflect** | Cross-interview recall/reflect: skill patterns, failing questions, skill gaps | `internal/evaluation/application/reflect.go` |
| **Dashboard polish** | Full candidate list, filters, comparisons | `frontend/pages/` |

### Evaluation Schema

**Canonical — Research §2 & §5 must follow this schema (single source of truth).**
```json
{
  "overall_score": 78,
  "dimensions": {
    "technical": { "score": 82, "weight": 0.4 },
    "communication": { "score": 75, "weight": 0.2 },
    "problem_solving": { "score": 80, "weight": 0.25 },
    "culture_fit": { "score": 70, "weight": 0.15 }
  },
  "per_question": [
    {
      "question_idx": 1,
      "score": 85,
      "rationale": "Strong understanding of distributed systems",
      "strengths": ["Clear explanation", "Used real examples"],
      "weaknesses": []
    }
  ],
  "strengths": ["Go expertise", "System design", "Communication"],
  "weaknesses": ["Limited cloud experience"],
  "recommendation": "proceed"
}
```

### Cross-Interview Reflect (Phase 4)

Interview selesai → sync summary ke Mnemosyne → reflect lintas interview jadi mungkin:

```go
// internal/evaluation/application/reflect.go
// 1. Sync interview summary into the tenant bank
mn := s.mnemosyne.ForBank(orgID.String())
mn.Remember(ctx, "interview_summary", summary, mnemosyne.WithImportance(0.7))

// 2. Reflect: patterns across many interviews
insight, _ := mn.Reflect(ctx,
    "which question fails most often and why?")

// 3. Find similar candidates who passed (semantic recall)
similar, _ := mn.Recall(ctx,
    "strong Go + fintech candidates who passed screening")
```

### Testing Criteria (P4a — beta gate)
- [x] Evaluation returns valid structured JSON (per-question scores + overall)
- [x] Report aggregates correctly (domain recomputes; LLM never sets the final number)
- [x] `evaluation` frame carries real scores + overall + recommendation (complete/pending)
- [x] `GET /interviews/:id` + `GET /interviews` list + `GET /candidates/:id/report` (org-checked)
- [x] Recruiter dashboard-lite: upload CV → create job → create interview → see result (browser, Playwright-verified)
- [x] Invite URL flow: share link → candidate opens → consent → ticket exchange → chat (E2E-verified)
- [x] Edge cases: empty transcript/single answer/long interview covered by domain tests + evaluator validation

### Testing Criteria (P4b — post-MVP)
- [ ] PDF report download works
- [ ] Interview summary synced to the tenant bank
- [ ] Cross-interview reflect returns valid patterns/insights

---

## Phase 5: Voice Interview (Week 13-16) — POST-MVP, ONLY with a paying customer

> **DEFERRED (2026-08-18, design decision via grilling session):** the working
> tree ships a gated demo of this phase — WS route (ticket + org-checked),
> Pion signaling (SDP offer/answer + ICE aligned with the FE), VAD/STT/TTS
> adapters, and an FE voice page at `/voice/:id` (URL-only, not in nav). The
> MVP demo path delivers TTS audio to the client as base64 `audio` WS frames;
> **real Opus decoding (mic → STT) and Opus-over-RTP (TTS → speaker) remain
> unimplemented.** Full Phase 5 lands only when a paying customer requires it
> (WHY: per original phase gate — no paying customer; WHAT: voice pipeline,
> TURN, recording; WHEN: on first paying customer request). The gated demo
> stays in the tree as a sales demo until then.

**Goal:** Real-time voice interview via WebRTC + Whisper STT + Edge TTS

### Deliverables

| Area | What | Files |
|------|------|-------|
| **WebRTC signaling** | Pion-based signaling server, SDP exchange | `internal/interview/infrastructure/webrtc/` |
| **VAD** | Voice activity detection (silero-vad) — segmentasi utterance → trigger STT | `internal/interview/infrastructure/webrtc/vad.go` |
| **STT adapter** | Whisper via whisper.cpp CLI (tiny=dev, small/large-v3=production) | `internal/interview/infrastructure/stt/whisper.go` |
| **TTS adapter** | Edge TTS API integration | `internal/interview/infrastructure/tts/edge_tts.go` |
| **Voice session** | Audio pipeline: mic → STT → LLM → TTS → speaker | `internal/interview/application/voice_session.go` |
| **TURN server** | Coturn setup for NAT traversal | `infra/turn/` |
| **Recording** | Save audio to MinIO, transcription to DB | `internal/interview/infrastructure/recording/` |
| **API endpoints** | `WS /interviews/:id/voice` | `internal/interview/api/voice_handler.go` |
| **Frontend** | WebRTC client (getUserMedia, peer connection) | `frontend/pages/interview/voice.tsx` |

### Voice Pipeline
```
Browser mic → Opus → WebRTC (Pion) → PCM → VAD (silero-vad) → segment → Whisper STT
                                                    │
                                                    ▼
                                            DeepSeek Flash (LLM)
                                                    │
                                                    ▼
                                            Edge TTS API
                                                    │
                                                    ▼
                              WebRTC (Pion) → Opus → Browser speaker
```

### Latency Budget
| Stage | Target |
|-------|--------|
| VAD (silero-vad, CPU) | <0.1s per segment |
| Whisper STT (tiny=dev; small/large-v3=production) | 2-3s per 5s audio |
| DeepSeek Flash generate | 0.5-1.5s |
| Edge TTS | 0.3-0.5s |
| WebRTC network | 0.1-0.2s |
| **Total per turn** | **~3-5s** |

### Testing Criteria
- [x] WebRTC connection established (browser ↔ server)
- [x] VAD segmentation: energy-based VAD / speech detector
- [x] Audio streaming & handling via WebRTC / WebSocket signaling
- [x] Whisper STT adapter (whisper.cpp docker sidecar)
- [x] DeepSeek generates response
- [x] Edge TTS returns audio
- [x] Audio played back in browser
- [ ] TURN server fallback works (UDP blocked)
- [ ] Recording saved to MinIO
- [ ] 5 concurrent voice sessions stable

---

## Phase 6: Production Polish (Week 17-18)

**Goal:** Observability, error tracking, deployment automation, performance
tuning. SPLIT for beta: P6a ships with the beta gate, P6b later.

### P6a — Beta essentials

| Area | What |
|------|------|
| **Deployment** | Docker Compose on VPS, domain + TLS, env management; push pipeline (GitHub Actions → build → deploy) |
| **Backup & DR** | Postgres daily dump to MinIO + rclone to B2; Mnemosyne bank backup; restore test run monthly (`scripts/backup.sh`, `scripts/restore.sh`, `make backup`, `make restore`) |
| **Health checks** | `/health`, `/ready` (DB, Redis, MinIO, DeepSeek reachability), `/live` |
| **Graceful shutdown** | SIGTERM: WS drain + LLM request drain (implemented — verify live) |
| **Rate limiting** | Per-tenant sliding window + auth limits (implemented — tune limits for beta) |
| **Structured logging** | JSON logs with request ID + tenant ID (implemented — verify retention/rotation) |
| **Error alerting (lite)** | Error visibility on beta failures (Sentry Go + Sentry React frontend) |

### P6b — Post-MVP

| Area | What |
|------|------|
| **Observability** | Prometheus metrics (request count, latency, LLM token usage, queue depth, active WS connections) + Grafana dashboard |
| **Error tracking** | Sentry for Go + frontend (full) |
| **Load testing** | k6: REST load test (`scripts/k6_load.js`, `make load-k6`) + WS loadcheck (`make load-ws`) |
| **Audit persistence** | `audit_logs` table writes (currently console-only) |
| **Infrastructure** | Terraform/Pulumi (optional) |
| **Compliance** | SOC 2 prep, GDPR consent docs (consent capture ships in P4a) |

### Monitoring Dashboard (Grafana)
```
┌─────────────────────────────────────────────────────┐
│  Active Interviews: 12    │  Avg Latency: 1.2s       │
│  LLM Tokens/min: 4,521    │  Queue Depth: 3          │
│  CVs Parsed: 1,234        │  Error Rate: 0.12%       │
├─────────────────────────────────────────────────────┤
│  Response Time p50: 450ms  p95: 1.2s  p99: 2.8s     │
│  WebSocket Connections: 47  Active Voice: 3          │
│  DeepSeek API Cost: $2.34  Storage: 1.2GB            │
└─────────────────────────────────────────────────────┘
```

### Testing Criteria
- [x] Prometheus metrics exposed on `/metrics` (with custom token & ws metrics)
- [x] Sentry captures error in production (Go backend + React frontend)
- [x] Rate limiter blocks abusive client
- [x] Graceful shutdown drains active interviews
- [x] k6 test: 100 concurrent users (`scripts/k6_load.js` / `make load-k6`)
- [x] Startup time < 3 seconds
- [x] Binary size < 30MB

---

## Timeline Summary

```
Week 1-2   〓〓 Phase 0: Customer Discovery (10 recruiters, 5 pilots → beta cohort)
Week 3-4   〓〓 Phase 1: Foundation (+ Mnemosyne bank setup)
Week 5-7   〓〓〓 Phase 2: Core Business Logic (+ semantic CV index, company context, tenant prompt)
Week 8-10  〓〓〓 Phase 3: Chat Interview (+ prompt composer, candidate chat UI)
Week 11-12 〓〓 Phase 4a: Evaluation core + recruiter dashboard-lite + invite/consent (MVP/BETA GATE)
Week 13-16 〓〓〓〓 Phase 5: Voice Interview — POST-MVP, ONLY with a paying customer
Week 17-18 〓〓 Phase 6a: Beta ops — deploy, backup & DR, alerting (parallel with beta)
         ────────────────────────
Total: 18 weeks (~4.5 months) solo — including Phase 0 validation + beta cohort
```

**P4b + P6b** (PDF report, full dashboard, cross-interview reflect, Prometheus,
Sentry full, k6, Terraform, SOC 2) run post-MVP — scheduled after beta
feedback decides what pilots actually use.

**Hybrid memory DB (Mnemosyne)** spans 4 phases: bank setup (P1), CV sync (P2), interview sync (P4), reflect/recall features (P4). Doesn't add to total duration — runs in parallel with core deliverables.

**Company context & tenant prompt:** P2 (upload/versioning/index) + P3 (prompt composer in the interview engine). Runs in parallel, doesn't add duration.

## Phase Dependencies

```
Phase 0 ──► Phase 1 ──► Phase 2 ──► Phase 3 ──► Phase 4
                  │                       │
                  │                       │
                  └──────────────────► Phase 5
                                        │
                                        ▼
                                    Phase 6 ◄── all phases
```

- Phase 0 → Phase 1: validate first (10 recruiters + 5 pilots) BEFORE coding
- Phase 2 depends on Phase 1 (IAM, DB, Queue, Storage)
- Phase 3 depends on Phase 2 (CV, Job, screening needed for interview context)
- Phase 4 depends on Phase 3 (need interview data to evaluate)
- Phase 5 depends on Phase 2 (CV, Job) + Phase 3 (interview domain) — POST-MVP, only with a paying customer
- Phase 6 depends on all phases but can be done incrementally

## What to Skip in MVP

| Feature | Phase | Skip? | Reason |
|---------|-------|-------|--------|
| **Voice interview** | 5 | ✅ **SKIP entirely in MVP** | Chat-only MVP first; voice only if there's a paying customer (solo dev: don't build features nobody pays for) |
| Batch CV upload | 2 | ✅ MVP | Single upload first |
| SSO/SAML | 1 | ✅ MVP | Email + Google OAuth enough |
| PDF report generation | 4b | ✅ MVP | JSON report first |
| Full recruiter dashboard | 4b | ✅ MVP | Dashboard-lite required (P4a) |
| Cross-interview reflect | 4b | ✅ MVP | Needs interview volume; runs post-MVP |
| Load testing | 6b | ✅ MVP | Manual smoke test + loadcheck harness |
| Terraform | 6b | ✅ MVP | Docker Compose enough |
| SOC 2 | 6b | ✅ MVP | Enterprise item, 1-2 years; focus on GDPR + consent basics (consent capture ships in P4a) |
| **Mnemosyne bank + CV semantic index** | 1-2 | ❌ Mandatory | Foundation for semantic recall; index-time $0 (reflect = small LLM call at query-time), big value add |
| **Company context + tenant prompt** | 2-3 | ❌ Mandatory | Main differentiator: tenant-specific interviewer; cheap (context index $0) |
| **Backup & DR** | 1 | ❌ Mandatory | Survival: 1 server + 0 backups = losing everything |

**True MVP = Phase 0 + Phase 1 + Phase 2 + Phase 3 + Phase 4a = ~12 weeks** (no voice). Mnemosyne bank + CV sync + company context stay in MVP; the evaluation core (P4a) closes the interview loop — empty scores = no product. Cross-interview reflect + voice + P4b/P6b extras are skipped. Beta gate sits at the end of P4a: deploy live, backup restore tested, 5 pilot interviews completed end-to-end, feedback channel open.
