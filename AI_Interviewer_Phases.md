# AI Interviewer — Development Phases

## Dependency Map

```
Phase 0: Customer Discovery ◄── VALIDATE SEBELUM CODING
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

## Phase 0: Customer Discovery (Week 1-2) — WAJIB SEBELUM CODING

**Goal:** Validasi masalah + dapet 5 pilot customer SEBELUM nulis baris code. Founder yang bener: bicara dulu, coding setelah.

### Deliverables

| Aktivitas | Apa | Output |
|-----------|-----|--------|
| **Interview 10 recruiter/HR** | Agency recruiter + BPO + tech startup (Jakarta-Bandung, network Hyperscal/OCBC) | Pain points terverifikasi, 5 pilot interest |
| **Problem validation** | "Apa masalah paling sakit di screening/interview lo?" — jangan tanya fitur | Jawaban "interview process lambat/bias/inkonsisten" = validasi, bukan |
| **Pilot commitment** | 5 perusahaan setuju coba gratis 1 bulan | Pilot list + kontak |
| **Pricing validation** | Tanya: "buat screening 100 kandidat, lo mau bayar berapa?" | Angka pricing real dari market |
| **Scope trim** | Fitur yang nggak diminta pilot → buang dari MVP | MVP scope final |

### Exit Criteria (jangan lanjut Phase 1 sebelum ini)
- [x] 10+ interview recruiter/HR selesai
- [x] Masalah interview process terkonfirmasi (bukan tebakan)
- [x] 5 pilot customer committed
- [x] Pricing validated (bukan asumsi)

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
| **Mnemosyne (hybrid memory)** | Bank manager per tenant, semantic layer setup, sync worker skeleton | `pkg/mnemosyne/`, `internal/memory/` |
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
- **Hybrid memory: Mnemosyne (1 SQLite bank per tenant, semantic layer) — embedding lokal fastembed, $0**

### Testing Criteria
- [x] Unit tests pass
- [x] Auth middleware rejects unauthenticated requests
- [x] Tenant isolation: User A cannot see User B's data
- [x] Docker Compose starts all services
- [x] Health endpoint returns 200
- [x] **Mnemosyne bank per tenant dibuat + recall terisolasi antar tenant**

---

## Phase 2: Core Business Logic (Week 5-7)

**Goal:** CV upload + parsing, job management, scoring engine, company context

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Job context** | Job entity, skill requirements, create/update/list | `internal/job/` |
| **CV context** | CV upload, pdfcpu parsing, DeepSeek structured extraction, embedding | `internal/cv/` |
| **Screening context** | Scoring engine, weighted algorithm, passing threshold | `internal/screening/` |
| **Company context** | Upload file/text, versioning, hash dedup, index ke Mnemosyne bank tenant | `internal/context/` |
| **Tenant prompt** | Set/get tenant system prompt, validasi, versioning, default fallback | `internal/context/` |
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
    → remember candidate_profile ke banks/<org_id>/mnemosyne.db
    → recall semantik kandidat siap dipakai
```

### Semantic Sync ke Mnemosyne (Phase 2)

CV/extract selesai → worker sync ringkasan semantic ke bank tenant:

```go
// internal/memory/application/sync_worker.go
func (s *SyncWorker) SyncCandidate(ctx context.Context, orgID, candidateID uuid.UUID, summary string) error {
    mn := s.mnemosyne.ForBank(orgID.String()) // banks/<org_id>/mnemosyne.db
    _, err := mn.Remember(ctx, "candidate_profile", summary, mnemosyne.WithImportance(0.9))
    return err
}
```

Yang di-index: ringkasan semantic (bukan PII mentah) — recall lintas kandidat jadi mungkin.

### Testing Criteria (tambahan Phase 2)
- [x] Sync worker nulis ke bank tenant yang bener
- [x] Recall "kandidat Go fintech" nemu kandidat yang cocok secara semantik
- [x] Upload company context → version bump + index ke bank tenant
- [x] Tenant set prompt → validasi tolak prompt injection (max length, forbidden keyword)
- [x] Tenant tanpa prompt → fallback ke global default
- [x] Context version ke-pin pas interview (audit traceable)

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
- [x] Upload PDF → Dapat extract text + structured data
- [x] CV score matches expected weighted calculation
- [x] Same CV scored against 2 different JDs produces different scores
- [x] Async jobs complete successfully
- [x] Score threshold correctly filters candidates

---

## Phase 3: Chat Interview (Week 8-10)

**Goal:** Real-time chat interview via WebSocket with DeepSeek Flash

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Interview domain** | Interview entity, Question VO, Answer VO | `internal/interview/domain/` |
| **LLM provider** | DeepSeek Flash adapter (streaming + structured output) | `internal/interview/infrastructure/llm/` |
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

### Testing Criteria
- [x] WebSocket connects and handshake completes
- [x] DeepSeek Flash returns streaming response
- [x] Questions generated based on CV gaps
- [x] Answers stored in PostgreSQL
- [x] Reconnection resumes from last unanswered question
- [x] Bias detection catches prohibited questions
- [x] Idle timeout disconnects after 5 minutes
- [x] 100 concurrent WebSocket connections stable
- [x] System prompt composer: tenant prompt + company context + safety rails di-compose bener
- [x] Safety rails selalu di posisi terakhir (tenant ga bisa override)

---

## Phase 4: Evaluation & Reports (Week 11-12)

**Goal:** Post-interview evaluation, report generation, recruiter dashboard

### Deliverables

| Area | What | Files |
|------|------|-------|
| **Evaluation domain** | Report entity, criteria, scoring | `internal/evaluation/domain/` |
| **LLM evaluation** | Structured output via function calling | `internal/evaluation/infrastructure/llm/` |
| **Report generation** | Aggregate per-question scores → final report | `internal/evaluation/application/generate_report.go` |
| **Semantic index (interview)** | Sync interview_summary + reflection ke Mnemosyne bank tenant | `internal/memory/application/` |
| **Cross-interview reflect** | Recall/reflect lintas interview: pola skill, pertanyaan gagal, skill gap | `internal/evaluation/application/reflect.go` |
| **API endpoints** | `GET /interviews/:id/evaluation`, `GET /candidates/:id/report` | `internal/evaluation/api/` |
| **Recruiter dashboard (FE)** | Candidate list, scores, report view | `frontend/pages/` |

### Evaluation Schema
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
// 1. Sync ringkasan interview ke bank tenant
mn := s.mnemosyne.ForBank(orgID.String())
mn.Remember(ctx, "interview_summary", summary, mnemosyne.WithImportance(0.7))

// 2. Reflect: pola dari banyak interview
insight, _ := mn.Reflect(ctx,
    "pertanyaan mana yang paling sering gagal dan kenapa?")

// 3. Cari kandidat mirip yang lolos (recall semantik)
similar, _ := mn.Recall(ctx,
    "kandidat kuat di Go + fintech yang lolos screening")
```

### Testing Criteria
- [x] Evaluation returns valid structured JSON
- [x] Report aggregates correctly
- [x] PDF report download works
- [x] Edge cases: empty transcript, single answer, very long interview
- [x] Interview summary ter-sync ke bank tenant
- [x] Reflect lintas interview return pola/insight yang valid

---

## Phase 5: Voice Interview (Week 13-16) — POST-MVP, HANYA kalau ada paying customer

**Goal:** Real-time voice interview via WebRTC + Whisper STT + Edge TTS

### Deliverables

| Area | What | Files |
|------|------|-------|
| **WebRTC signaling** | Pion-based signaling server, SDP exchange | `internal/interview/infrastructure/webrtc/` |
| **STT adapter** | Whisper via whisper.cpp CLI | `internal/interview/infrastructure/stt/whisper.go` |
| **TTS adapter** | Edge TTS API integration | `internal/interview/infrastructure/tts/edge_tts.go` |
| **Voice session** | Audio pipeline: mic → STT → LLM → TTS → speaker | `internal/interview/application/voice_session.go` |
| **TURN server** | Coturn setup for NAT traversal | `infra/turn/` |
| **Recording** | Save audio to MinIO, transcription to DB | `internal/interview/infrastructure/recording/` |
| **API endpoints** | `WS /interviews/:id/voice` | `internal/interview/api/voice_handler.go` |
| **Frontend** | WebRTC client (getUserMedia, peer connection) | `frontend/pages/interview/voice.tsx` |

### Voice Pipeline
```
Browser mic → Opus → WebRTC (Pion) → PCM → Whisper STT
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
| Whisper STT (tiny, CPU) | 2-3s per 5s audio |
| DeepSeek Flash generate | 0.5-1.5s |
| Edge TTS | 0.3-0.5s |
| WebRTC network | 0.1-0.2s |
| **Total per turn** | **~3-5s** |

### Testing Criteria
- [x] WebRTC connection established (browser ↔ server)
- [x] Audio recorded and sent to server
- [x] Whisper transcribes audio to text
- [x] DeepSeek generates response
- [x] Edge TTS returns audio
- [x] Audio played back in browser
- [x] TURN server fallback works (UDP blocked)
- [x] Recording saved to MinIO
- [x] 5 concurrent voice sessions stable

---

## Phase 6: Production Polish (Week 17-18)

**Goal:** Observability, error tracking, deployment automation, performance tuning

### Deliverables

| Area | What |
|------|------|
| **Observability** | Prometheus metrics (request count, latency, LLM token usage, queue depth) |
| **Health checks** | `/health`, `/ready` endpoints (checks DB, Redis, MinIO, DeepSeek API) |
| **Graceful shutdown** | SIGTERM handler for WebSocket drain + LLM request drain |
| **Rate limiting** | Per-tenant token bucket for LLM/API, per-user rate limit |
| **Error tracking** | Sentry integration for Go + frontend |
| **Structured logging** | All logs in JSON format with request ID, tenant ID, trace ID |
| **Audit log** | All data access logged to `audit_logs` table |
| **Deployment** | GitHub Actions → Docker build → push to registry → deploy |
| **Backup & DR** | Postgres daily dump ke MinIO, rclone ke B2, Mnemosyne bank backup, test restore bulanan |
| **Infrastructure** | Terraform/Pulumi for cloud resources (optional) |
| **Load testing** | k6 script for WebSocket + REST + voice simulation |

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
- [x] Prometheus metrics exposed on `/metrics`
- [x] Sentry captures error in production
- [x] Rate limiter blocks abusive client
- [x] Graceful shutdown drains active interviews
- [x] k6 test: 100 concurrent users, 2000 req/s
- [x] Startup time < 3 seconds
- [x] Binary size < 30MB

---

## Timeline Summary

```
Week 1-2   〓〓 Phase 0: Customer Discovery (WAJIB — 10 recruiter, 5 pilot)
Week 3-4   〓〓 Phase 1: Foundation (+ Mnemosyne bank setup)
Week 5-7   〓〓〓 Phase 2: Core Business Logic (+ semantic CV index, company context, tenant prompt)
Week 8-10  〓〓〓 Phase 3: Chat Interview (+ prompt composer)
Week 11-12 〓〓 Phase 4: Evaluation & Reports (+ cross-interview reflect)
Week 13-16 〓〓〓〓 Phase 5: Voice Interview — POST-MVP, HANYA kalau ada paying customer
Week 17-18 〓〓 Phase 6: Production Polish (+ backup & DR)
         ────────────────────────
Total: 18 weeks (~4.5 months) solo — termasuk Phase 0 validation
```

**Hybrid memory DB (Mnemosyne)** tersebar di 4 phase: setup bank (P1), sync CV (P2), sync interview (P4), reflect/recall features (P4). Tidak nambah durasi total — jalan paralel dengan core deliverable.

**Company context & tenant prompt:** P2 (upload/versioning/index) + P3 (prompt composer di interview engine). Jalan paralel, tidak nambah durasi.

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

- Phase 0 → Phase 1: validasi dulu (10 recruiter + 5 pilot) SEBELUM coding
- Phase 2 depends on Phase 1 (IAM, DB, Queue, Storage)
- Phase 3 depends on Phase 2 (CV, Job, screening needed for interview context)
- Phase 4 depends on Phase 3 (need interview data to evaluate)
- Phase 5 depends on Phase 2 (CV, Job) + Phase 3 (interview domain) — POST-MVP, hanya dengan paying customer
- Phase 6 depends on all phases but can be done incrementally

## What to Skip in MVP

| Feature | Phase | Skip? | Reason |
|---------|-------|-------|--------|
| **Voice interview** | 5 | ✅ **SKIP total di MVP** | Chat-only MVP dulu; voice cuma kalau ada paying customer (solo dev: jangan bangun fitur yang belum ada yang bayar) |
| Batch CV upload | 2 | ✅ MVP | Single upload first |
| SSO/SAML | 1 | ✅ MVP | Email + Google OAuth enough |
| PDF report generation | 4 | ✅ MVP | JSON report first |
| Load testing | 6 | ✅ MVP | Manual smoke test |
| Terraform | 6 | ✅ MVP | Docker Compose enough |
| SOC 2 | 6 | ✅ MVP | Enterprise item, 1-2 tahun; fokus GDPR + consent basics |
| Cross-interview reflect | 4 | ✅ MVP | Butuh volume interview; jalan di post-MVP |
| **Mnemosyne bank + CV semantic index** | 1-2 | ❌ Wajib | Dasar semantic recall; murah ($0), nambah value besar |
| **Company context + tenant prompt** | 2-3 | ❌ Wajib | Differentiator utama: interviewer spesifik per tenant; murah (context index $0) |
| **Backup & DR** | 1 | ❌ Wajib | Survival: 1 server + 0 backup = kehilangan semua |

**True MVP = Phase 0 + Phase 1 + Phase 2 + Phase 3 = 10 weeks** (tanpa voice). Mnemosyne bank + CV sync + company context tetap di MVP — reflect lintas interview + voice yang di-skip.
