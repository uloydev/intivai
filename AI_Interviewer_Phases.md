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
Phase 5: Voice Interview (WebRTC + STT + TTS) — POST-MVP, skip sampai paying customer
    │
    ▼
Phase 6: Production Polish (Observability, Deploy, Scale)
```

---

## Phase 0: Customer Discovery (Week 1-2) — MANDATORY BEFORE CODING

**Goal:** Validate the problem + get 5 pilot customers BEFORE writing a line of code. A real founder: talks first, codes after.

### Deliverables

| Activity | What | Output |
|-----------|-----|--------|
| **Interview 10 recruiters/HR** | Agency recruiters + BPO + tech startups (Jakarta-Bandung, Hyperscal/OCBC network) | Pain points verified, 5 pilot interests |
| **Problem validation** | "What's the most painful problem in your screening/interview?" — don't ask about features | Answer "interview process slow/biased/inconsistent" = validation, not a feature ask |
| **Pilot commitment** | 5 companies agree to try free for 1 month | Pilot list + contacts |
| **Pricing validation** | Ask: "to screen 100 candidates, how much would you pay?" | Real pricing numbers from the market |
| **Scope trim** | Features pilots didn't ask for → cut from MVP | MVP scope final |

### Exit Criteria (do not start Phase 1 before this)
- [ ] 10+ recruiter/HR interviews done
- [ ] Interview process problem confirmed (not a guess)
- [ ] 5 pilot customers committed
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
| **Database** | PostgreSQL connection pool, migration tool setup | `pkg/db/postgres.go`, `pkg/db/migrations/001_init.up.sql` |
| **Queue** | Asynq setup + worker skeleton | `pkg/queue/asynq.go` |
| **Storage** | MinIO/S3 client skeleton | `pkg/storage/s3.go` |
| **Memory layer (Mnemosyne port)** | MemoryBank port + adapter (native SQLite+fastembed default; MCP optional), bank per tenant, sync worker skeleton | `internal/memory/` |
| **IAM** | Org, User entities + Auth (JWT), RBAC, multi-tenant middleware | `internal/iam/` |
| **API scaffold** | Fiber app, middleware (auth, tenant, CORS, rate limit), health check | `cmd/server/main.go` |
| **DevOps** | Docker Compose (Go app + Postgres + Redis + MinIO) | `docker-compose.yml` |
| **CI** | GitHub Actions: lint, test, build | `.github/workflows/ci.yml` |

### Key Decisions Made
- Framework: Fiber
- Database: PostgreSQL + pgvector (source of truth)
- Queue: Asynq (Redis)
- Auth: JWT + per-tenant RBAC
- File storage: MinIO (S3-compatible)
- **Hybrid memory: MemoryBank port (1 SQLite bank per tenant) — native Go adapter by default (fastembed, $0); MCP optional. Reflect = small LLM call at query-time**

### Testing Criteria
- [ ] Unit tests pass
- [ ] Auth middleware rejects unauthenticated requests
- [ ] Tenant isolation: User A cannot see User B's data
- [ ] Docker Compose starts all services
- [ ] Health endpoint returns 200
- [ ] **Per-tenant Mnemosyne bank created + recall isolated across tenants**

---

## Phase 2: Core Business Logic (Week 5-7)

**Goal:** CV upload + parsing, job management, scoring engine, company context

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Job context** | Job entity, skill requirements, create/update/list | `internal/job/` |
| **CV context** | CV upload, pdfcpu parsing, DeepSeek structured extraction, embedding | `internal/cv/` |
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
    → pdfcpu extract text
    → if scanned → Tesseract OCR
    → save raw text
  → queue: extract_cv
    → DeepSeek structured output → ResumeData
    → DeepSeek embedding → vector
    → save structured + vector
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
func (s *SyncWorker) SyncCandidate(ctx context.Context, orgID, candidateID uuid.UUID, summary string) error {
    mn := s.mnemosyne.ForBank(orgID.String()) // banks/<org_id>/mnemosyne.db
    _, err := mn.Remember(ctx, "candidate_profile", summary, mnemosyne.WithImportance(0.9))
    return err
}
```

What gets indexed: semantic summaries (not raw PII) — cross-candidate recall becomes possible.

### Testing Criteria (tambahan Phase 2)
- [ ] Sync worker writes to the correct tenant bank
- [ ] Recall "Go fintech candidate" finds semantically matching candidates
- [ ] Company context upload → version bump + index into tenant bank
- [ ] Tenant prompt set → validation rejects prompt injection (max length, forbidden keywords)
- [ ] Tenant without prompt → falls back to global default
- [ ] Context version pinned at interview time (audit traceable)

---

### Scoring Engine
```go
type ScoringWeights struct {
    SkillsMatch      float64 // 0.35
    ExperienceYears  float64 // 0.20
    SemanticMatch    float64 // 0.25 (embedding cosine similarity)
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
- [ ] PDF upload → extracted text + structured data obtained
- [ ] CV score matches expected weighted calculation
- [ ] Same CV scored against 2 different JDs produces different scores
- [ ] Async jobs complete successfully
- [ ] Score threshold correctly filters candidates

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
| **API endpoints** | `POST /interviews`, `WS /interviews/:id/chat` | `internal/interview/api/` |
| **Reconnection** | Store last answered question, allow resume | `internal/interview/application/` |

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
- [ ] WebSocket connects and handshake completes (candidate uses WS ticket, not JWT)
- [ ] WS upgrade without a valid ticket is rejected
- [ ] DeepSeek Flash returns streaming response
- [ ] Questions generated based on CV gaps
- [ ] Selected questions persisted to the question bank (reuse + audit)
- [ ] Answers stored in PostgreSQL
- [ ] Reconnection resumes from last unanswered question
- [ ] Bias detection catches prohibited questions
- [ ] Idle timeout disconnects after 5 minutes
- [ ] 100 concurrent WebSocket connections stable
- [ ] System prompt composer: tenant prompt + company context + safety rails composed correctly
- [ ] Safety rails always last (tenant cannot override)

---

## Phase 4: Evaluation & Reports (Week 11-12)

**Goal:** Post-interview evaluation, report generation, recruiter dashboard

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Evaluation domain** | Report entity, criteria, scoring | `internal/evaluation/domain/` |
| **LLM evaluation** | Structured output via function calling | `internal/evaluation/infrastructure/llm/` |
| **Report generation** | Aggregate per-question scores → final report | `internal/evaluation/application/generate_report.go` |
| **Semantic index (interview)** | Sync interview_summary + reflection into the tenant's Mnemosyne bank | `internal/memory/application/` |
| **Cross-interview reflect** | Cross-interview recall/reflect: skill patterns, failing questions, skill gaps | `internal/evaluation/application/reflect.go` |
| **API endpoints** | `GET /interviews/:id/evaluation`, `GET /candidates/:id/report` | `internal/evaluation/api/` |
| **Recruiter dashboard (FE)** | Candidate list, scores, report view | `frontend/pages/` |

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

### Testing Criteria
- [ ] Evaluation returns valid structured JSON
- [ ] Report aggregates correctly
- [ ] PDF report download works
- [ ] Edge cases: empty transcript, single answer, very long interview
- [ ] Interview summary synced to the tenant bank
- [ ] Cross-interview reflect returns valid patterns/insights

---

## Phase 5: Voice Interview (Week 13-16) — POST-MVP, ONLY with a paying customer

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
- [ ] WebRTC connection established (browser ↔ server)
- [ ] VAD segmentation correct: answers split at pauses, not merged across sentences
- [ ] Audio recorded and sent to server
- [ ] Whisper transcribes audio to text
- [ ] DeepSeek generates response
- [ ] Edge TTS returns audio
- [ ] Audio played back in browser
- [ ] TURN server fallback works (UDP blocked)
- [ ] Recording saved to MinIO
- [ ] 5 concurrent voice sessions stable

---

## Phase 6: Production Polish (Week 17-18)

**Goal:** Observability, error tracking, deployment automation, performance tuning

### Deliverables

| Area | What |
|------|------|
| **Observability** | Prometheus metrics (request count, latency, LLM token usage, queue depth) |
| **Health checks** | `/health`, `/ready` endpoints (checks DB, Redis, MinIO, DeepSeek API) |
| **Graceful shutdown** | SIGTERM handler for WebSocket drain + LLM request drain |
| **Rate limiting** | Per-tenant sliding window (Redis) for API, fixed 1-min window for LLM tokens, per-user rate limit |
| **Error tracking** | Sentry integration for Go + frontend |
| **Structured logging** | All logs in JSON format with request ID, tenant ID, trace ID |
| **Audit log** | All data access logged to `audit_logs` table |
| **Deployment** | GitHub Actions → Docker build → push to registry → deploy |
| **Backup & DR** | Postgres daily dump ke MinIO, rclone ke B2, Mnemosyne bank backup, test restore bulanan |
| **Infrastructure** | Terraform/Pulumi for cloud resources (optional) |
| **Load testing** | k6 for WebSocket + REST; voice = manual smoke test (k6 doesn't support WebRTC) |

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
- [ ] Prometheus metrics exposed on `/metrics`
- [ ] Sentry captures error in production
- [ ] Rate limiter blocks abusive client
- [ ] Graceful shutdown drains active interviews
- [ ] k6 test: 100 concurrent users, 2000 req/s
- [ ] Startup time < 3 seconds
- [ ] Binary size < 30MB

---

## Timeline Summary

```
Week 1-2   〓〓 Phase 0: Customer Discovery (MANDATORY — 10 recruiters, 5 pilots)
Week 3-4   〓〓 Phase 1: Foundation (+ Mnemosyne bank setup)
Week 5-7   〓〓〓 Phase 2: Core Business Logic (+ semantic CV index, company context, tenant prompt)
Week 8-10  〓〓〓 Phase 3: Chat Interview (+ prompt composer)
Week 11-12 〓〓 Phase 4: Evaluation & Reports (+ cross-interview reflect)
Week 13-16 〓〓〓〓 Phase 5: Voice Interview — POST-MVP, ONLY with a paying customer
Week 17-18 〓〓 Phase 6: Production Polish (+ backup & DR)
         ────────────────────────
Total: 18 weeks (~4.5 months) solo — including Phase 0 validation
```

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
| PDF report generation | 4 | ✅ MVP | JSON report first |
| Load testing | 6 | ✅ MVP | Manual smoke test |
| Terraform | 6 | ✅ MVP | Docker Compose enough |
| SOC 2 | 6 | ✅ MVP | Enterprise item, 1-2 years; focus on GDPR + consent basics |
| Cross-interview reflect | 4 | ✅ MVP | Needs interview volume; runs post-MVP |
| **Mnemosyne bank + CV semantic index** | 1-2 | ❌ Mandatory | Foundation for semantic recall; index-time $0 (reflect = small LLM call at query-time), big value add |
| **Company context + tenant prompt** | 2-3 | ❌ Mandatory | Main differentiator: tenant-specific interviewer; cheap (context index $0) |
| **Backup & DR** | 1 | ❌ Mandatory | Survival: 1 server + 0 backups = losing everything |

**True MVP = Phase 0 + Phase 1 + Phase 2 + Phase 3 = 10 weeks** (no voice). Mnemosyne bank + CV sync + company context stay in MVP — cross-interview reflect + voice are skipped.
