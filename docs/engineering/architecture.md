# Intivai — Project Structure (current implementation)

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM

DDD + hexagonal intent, applied pragmatically: every bounded context has
`domain` (entities + ports), `application` (use cases + workers), and
`infrastructure` (persistence/adapters), with HTTP handlers in `api/`.
The original design doc (pre-implementation) listed planned files that
landed differently — this file is the **synced source of truth**; the design
rationale lives in [`design-decisions.md`](design-decisions.md).

```
/
├── README.md                             # What + quickstart
├── AGENTS.md                             # Engineering workflow (commands, conventions)
├── CONTEXT.md                            # Ubiquitous language
├── CHANGELOG.md                          # Tag → notable changes
├── api/openapi.yaml                      # API contract (single source for DTOs)
├── docs/
│   ├── README.md                         # Docs index (status + last-reviewed per doc)
│   ├── FINDINGS.md                       # Single remediation ledger (all reviews)
│   ├── product/                          # PRD, pricing, demo seed guide
│   ├── engineering/                      # THIS FILE, design decisions, roadmap,
│   │                                     #   beta gate, schemas, threat model, SLOs
│   ├── adr/                              # Architecture decision records
│   ├── plans/active | plans/archive      # In-flight vs closed plans
│   ├── reviews/                          # Dated review snapshots (append-only)
│   ├── compliance/                       # Access control, retention, IR, SOC 2
│   └── runbooks/                         # deploy, rollback, restore drill, LLM outage
├── design-system/intivai/                # MASTER.md + page overrides (FE tokens)
│
├── backend/                            # Go monolith (modular)
│   ├── cmd/
│   │   ├── server/main.go              # Wiring: infra, contexts, workers, routes
│   │   └── loadcheck/                  # 100-conn WS load harness (make load-ws)
│   ├── pkg/
│   │   ├── config/                     # Viper env config (+ validation)
│   │   ├── db/                         # GORM pool (pgx stdlib), tenant/tx ctx,
│   │   │   └── migrations/             #   golang-migrate embedded (001–025 as of 2026-08-24)
│   │   ├── logger/                     # zerolog
│   │   ├── metrics/                    # Prometheus custom metrics (LLM tokens, active WS)
│   │   ├── queue/                      # asynq client/server, task-name consts
│   │   ├── storage/                    # MinIO (S3) client + FileStorage port
│   │   └── (removed)                   # no separate shared/kernel pkg — see internal/shared
│   ├── internal/
│   │   ├── shared/                     # domain kernel (Entity, events), errors
│   │   │   │                           #   (DomainError, NotFoundError), httpapi, httpmw
│   │   │   └──                        #   (auth/audit/cors/ratelimit/requestid)
│   │   ├── iam/                        # orgs, users, JWT auth, RBAC, roles
│   │   │   ├── domain/                 # Org, User, LoginIdentity
│   │   │   ├── application/            # RegisterOrg, Authenticate, CreateUser, TxManager
│   │   │   ├── infrastructure/auth/    # bcrypt + JWT provider (auth + ws_ticket types)
│   │   │   ├── infrastructure/persistence/  # postgres repo, tx manager
│   │   │   └── api/                    # auth handlers + auth/tenant-tx middlewares
│   │   ├── job/                        # job CRUD + PATCH status
│   │   ├── cv/                         # candidates, PDF upload
│   │   │   ├── domain/                 # Candidate entity, status machine
│   │   │   ├── application/            # CVService, ParseWorker, ExtractWorker,
│   │   │   │                           #   payload helpers
│   │   │   ├── infrastructure/ocr/     # pdftoppm + tesseract (scanned PDFs)
│   │   │   ├── infrastructure/persistence/
│   │   │   └── api/                    # upload/list/get/re-extract
│   │   ├── screening/                  # scoring engine + applications
│   │   │   ├── domain/                 # Score, SemanticScore (keyword + cosine),
│   │   │   │                           #   weights, Application
│   │   │   ├── application/            # ScreeningService, ScoreWorker
│   │   │   ├── infrastructure/persistence/
│   │   │   └── api/
│   │   ├── context/                    # company context + tenant prompt
│   │   │   ├── domain/                 # CompanyContext (versioned, dedup),
│   │   │   │                           #   TenantPrompt, ContainsInjection/ValidatePrompt
│   │   │   ├── application/            # ContextService, IndexWorker
│   │   │   ├── infrastructure/persistence/
│   │   │   └── api/
│   │   ├── interview/                  # chat + voice interview (WS / WebRTC)
│   │   │   ├── domain/                 # Interview aggregate (state machine,
│   │   │   │                           #   frozen clock), protocol frames, repos
│   │   │   ├── domain/service/         # question generator, bias, prompt composer,
│   │   │   │                           #   context window/budget, probe strategy
│   │   │   ├── application/            # InterviewService, VoiceSession, evaluation enqueuer
│   │   │   ├── infrastructure/persistence/  # interview/token/question-bank repos
│   │   │   ├── infrastructure/stt/     # Whisper STT adapter (whisper.cpp)
│   │   │   ├── infrastructure/tts/     # Edge TTS adapter (synthesized voice)
│   │   │   ├── infrastructure/webrtc/  # Pion signaling & VAD adapter
│   │   │   └── api/                    # chat handler (WS), voice handler (WS/WebRTC), session registry
│   │   ├── evaluation/                 # post-interview reports (P4a)
│   │   │   ├── domain/                 # Report schema + weighted aggregation
│   │   │   ├── application/            # EvaluationService (detail/list/report),
│   │   │   │                           #   EvaluationWorker (async retry)
│   │   │   ├── infrastructure/llm/     # evaluator (structured output, rails)
│   │   │   └── api/                    # GET /interviews, /interviews/:id, /candidates/:id/report
│   │   ├── notification/               # email notification subsystem
│   │   │   ├── application/            # EmailWorker (Mailpit SMTP dispatch)
│   │   │   └──                         #   interview invitation templates
│   │   ├── llm/                        # LLM provider (chat/stream/structured),
│   │   │                               #   Client with retry + fallback
│   │   ├── embedding/                  # local 384-dim embeddings (cybertron,
│   │   │                               #   CGO-free; multi-qa default, bge via env)
│   │   └── memory/                     # Mnemosyne memory banks
│   │       ├── domain/                 # MemoryBank port
│   │       ├── application/            # SyncWorker
│   │       └── infrastructure/
│   │           ├── native/             # SQLite bank per tenant (dev default)
│   │           └── postgres/           # pgvector bank + cosine recall (prod)
│   └── Makefile                        # check/lint/coverage/dev/smoke/seed/backup/restore/load-ws/load-k6
│
├── frontend/                           # React SPA (Vite + TS + Tailwind v4 + shadcn)
│   ├── src/
│   │   ├── lib/                        # api (typed errors), auth (JWT session),
│   │   │   │                           #   ws (chat frames), useProctoring (anti-cheat telemetry),
│   │   │   │                           #   theme (dark mode), utils
│   │   ├── components/                 # AppShell + shadcn ui primitives
│   │   ├── pages/                      # Landing, Careers, Login, Register, Dashboard, Jobs,
│   │   │                               #   CVs, Candidates, Interviews, InterviewResult,
│   │   │                               #   Invite, Chat, InterviewVoice
│   │   ├── types/api.ts                # DTO types (OpenAPI mirror)
│   │   └── index.css                   # design tokens (MASTER.md mapping, light+dark)
│   ├── e2e/                            # Playwright (happy path, auth, step-logged)
│   └── playwright.config.ts / vitest.config.ts
│
├── scripts/                            # smoke.sh, seed.sh, backup.sh, restore.sh,
│                                       #   check-coverage.sh, k6_load.js
├── docker-compose.yml                  # dev stack (migrate/app/postgres/redis/minio/whisper)
├── docker-compose.dev.yml              # dev port overlay (!override)
├── docker-compose.prod.yml             # prod overlay (Caddy-only ports, whisper, secrets)
├── Caddyfile                           # static FE + /api + WS proxy, auto-TLS
├── .env.prod.example                   # prod secrets checklist
└── .github/workflows/ci.yml            # backend, frontend, integration, smoke, deploy
```

## Observability

- **Metrics:** Prometheus (`pkg/metrics`, `/metrics`), LLM token ledger, active WS gauge
- **Errors:** Sentry (`pkg/observability`), fiber + worker panic capture
- **Tracing:** OpenTelemetry → Jaeger v2 all-in-one
  (`docs/plans/archive/intivai-remediation-plan-2026-08-26.md`)
  - Bootstrap: `pkg/telemetry.Init` in `cmd/server` + `cmd/sandboxd`;
    `INTIVAI_OTEL_ENABLE=false` is a hard noop and the runtime kill switch
  - Spans: HTTP server (`httpmw.Tracing`, route-template names), `tenant.tx` +
    gorm SQL (`db.RunInTx` / otelgorm via `db.WithPlugin`),
    `queue.process <task>` linked to the producing trace via task headers
    (`queue.TracingMiddleware`, wire FIRST on the worker mux),
    `llm.chat|chat_stream|structured_output` with per-attempt children,
    `interview.answer.process` / `interview.turn.stream`
  - Logs: `trace_id` sits next to `request_id` in every audit line — paste it into
    the Jaeger UI search to open the waterfall; UI at `localhost:16686` (dev overlay only)
  - PII rule: candidate names/emails/CV text/prompts/tokens never enter span attributes
  - **Tracing gate** (`scripts/check-tracing.sh`, part of `make check`): new backend code
    inherits tracing only through the instrumented seams, so the gate fails on any bypass —
    `c.Context()` in handlers, raw asynq enqueues outside `pkg/queue`, direct OpenAI usage
    outside `internal/llm`, package-level `otel.Tracer` vars, and missing bootstrap wiring
    in the two binaries

## Layer rules (enforced by review, not tooling)

- **Ports in `domain/`, adapters in `infrastructure/`** — e.g. `MemoryBank`,
  `InterviewRepository`, `ApplicationRepository`; one provider port for LLM
  (`internal/llm`) that every context uses (never per-context providers)
- **RLS discipline**: every tenant-scoped query runs in a tenant transaction
  (`db.RunInTx`, reused request tx) — see AGENTS.md
- **Workers** live in `application/` as `*Worker` with `Register(mux)`; task
  names single-sourced in `pkg/queue`
- **DTOs** mirror `api/openapi.yaml` (snake_case); FE types in
  `frontend/src/types/api.ts` are the same contract
- **Design deviations vs the original plan**: no `shared/domain` events bus
  (Event struct exists, unused), storage adapter in `pkg/storage` (not
  `infrastructure/`), PDF parsing via `ledongthuc/pdf` + tesseract OCR in
  `cv/infrastructure/ocr` (no `parser/` pkg), embeddings via cybertron
  (fastembed deferred — bge gated on HF), evaluation as its own context (P4a).
