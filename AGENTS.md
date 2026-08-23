# Intivai — Engineering Workflow

## Strict Production Guidelines (Target: Revenue-Generating Production Grade)

1. **Mandatory TDD:** Use Test-Driven Development (TDD) for EVERY implementation. No exceptions.
2. **Clean & Efficient Code:** Code must be highly optimized, readable, and free of technical debt.
3. **Zero Mistakes:** Mistakes are unacceptable.
4. **No Hallucinations:** If unsure, ASK immediately. Do not guess.
5. **Idiomatic Code:** Strictly use Golang and TypeScript idioms.
6. **Production-First Mindset:** This is NOT a beta project. It is aimed at making money. Every decision must prioritize stability, scalability, and security.
7. **Strict Rule Adherence:** Strictly follow all rules defined in this document.
8. **Fail-Safe Error Handling:** Never swallow errors. All errors must be handled, logged with context, and never leak sensitive data.
9. **Strict Types:** TypeScript `any` is strictly forbidden. Go `any` (formerly `interface{}`) forbidden unless generic.
10. **HR-Centric Product Mindset:** Every feature must strictly align with HR needs. Before implementing, ask: "As an HR professional, does this genuinely make my hiring process more efficient?" Build for the user, not just for the tech.
11. **Depth Over Speed:** NEVER sacrifice quality for speed. Always perform deep, thorough, and systemic analysis (file-by-file, checking transaction boundaries, race conditions, and architecture constraints) before proposing solutions or declaring a review complete.
12. **Proactive Systemic Auditing (Corollary to Depth Over Speed):** When a bug (e.g., missing database column in an `INSERT`, or an unchecked error) is identified in one repository or handler, NEVER stop at fixing just that one instance. You MUST proactively search and audit the entire codebase for the same class of bug before declaring the issue resolved.
13. **Explicit State Inserts:** NEVER rely on database defaults (omitted columns) for fields that participate in state machines or domain logic (like `status`, `stage`). Always explicitly insert their starting values (e.g., `stage = 'applied'`) in repository `Create` methods and Raw SQL inserts to ensure the Go domain layer always loads a valid initial state.
14. **No Dynamic JSON Parsing:** Enforcing the "Strict Types" rule. When handling incoming JSON (like WebSocket telemetry or worker payloads), you must define and use strict Go structs for nested data. NEVER use `map[string]any` or `[]any` as an escape hatch, even for "dynamic" or "extra" payload fields.

## Project

- Monorepo: `backend/` (Go), `frontend/` (React SPA), `docker-compose*.yml` (dev/prod stacks), `.github/workflows/ci.yml`
- Architecture reference: `docs/engineering/roadmap.md` (phases + testing criteria), `docs/engineering/design-decisions.md` (design + impl-sync table), `docs/engineering/architecture.md` (current structure)
- Phases: P1–P4a complete; **beta gate** status in `docs/engineering/beta-gate.md`; roadmap in `docs/engineering/roadmap.md`

## Commands (run from repo root — the root `Makefile` forwards to `backend/`; everything also works directly from `backend/`)

| Command | Purpose | When |
|---|---|---|
| `make check` | FULL gate: backend gofmt + golangci-lint + vet + build + unit tests, then FE `npm run build` + vitest | Before EVERY commit. Must be green |
| `make lint` | golangci-lint (config: `.golangci.yml`) | Before commit |
| `make coverage` | per-package coverage floors (domain ≥70%, others ≥50%; needs stack up) | Before commit |
| `make test-integration-dev` | integration tests against local compose (needs stack up) | After any schema/repo/worker change |
| `make test` | unit tests | Before commit |
| `make proto` | regenerate sandbox gRPC stubs from `proto/sandbox.proto` (needs protoc + plugins) | After editing the proto |
| `make sandbox-images` | build the 4 per-language sandbox execution images (ADR-0002) | After Dockerfile changes / fresh machine |
| `make dev` | boot full stack on FRESH redis (stale asynq tasks wiped; postgres/minio volumes persist; builds sandbox images + mTLS certs) | Start of a session |
| `make redis-clear` | `FLUSHALL` queue while stack stays up | Mid-session queue cleanup |
| `make smoke` | end-to-end API scenario against running stack (`CV_PDF` env or `/tmp/kilo/cv.pdf`) | After any API change |
| `make seed` | seed local DB with demo org, jobs, prompt rails, context | For local testing & demo |
| `make seed-fresh` | wipe volumes, start fresh containers, run migrations & seed | For starting from a completely clean slate |
| `make backup` | trigger manual DB + MinIO backup archive | Backup ops |
| `make restore DUMP=...` | disaster recovery restore from dump | DR verification |
| `make migrate` | apply migrations (admin URL) | After fresh DB or new migration |
| `go run ./cmd/server -migrate-only` | same as `make migrate` | — |
| `make load-ws` | 100-concurrent WS load check (`CONNS` overrides) | After WS handler changes |
| `make load-k6` | k6 REST load test (100 concurrent users) | Load testing |

## Frontend commands (run from `frontend/`; root aliases: `make fe-build`, `make fe-test`, `make fe-e2e`)

| Command | Purpose | When |
|---|---|---|
| `npm run build` | tsc + vite build | Before EVERY FE commit |
| `npx vitest run` | unit tests (api/ws libs) | Before commit |
| `npx playwright test` | E2E happy path (needs stack + LLM key) | After FE flow changes |

## Compose commands (run from repo root — compose files live at the root)

| Command | Purpose |
|---|---|
| `make up` / `make down` | start / stop the dev stack (base + dev overlay, `--env-file .env`) |
| `make logs` / `make ps` | follow logs / list stack status |
| `make restart` | `docker compose restart` |
| `make compose-build` | pre-up steps for a fresh machine: app image + sandbox images + mTLS certs |
| `make up-prod` / `make down-prod` | deploy / stop prod (base + prod overlay, `--env-file .env.prod`) |
| `make logs-prod` / `make ps-prod` | prod logs / status |

> `make dev` (backend) is the full fresh-session flow: down → build app image →
> sandbox images → certs → up. `make up` skips the rebuild steps.

## Production commands (VPS, `docker compose --env-file .env.prod`)

| Command | Purpose |
|---|---|
| `scripts/backup.sh` | nightly: pg_dump + MinIO mirror → backup bucket, 14-day retention |
| `scripts/restore.sh <dump>` | restore DB + object storage to `intivai_restore`, promote steps |
| `docker compose ... -f docker-compose.prod.yml up -d` | deploy (CI does this on main) |

> NEVER run prod compose without `--env-file .env.prod` — base compose
> `environment:` would win and boot with dev secrets. (`make up-prod` already
> enforces this via the `COMPOSE_PROD` invocation.)

Full check before commit (run from repo root):

```bash
make check && make coverage && make test-integration-dev
```

> The root `Makefile` is a thin delegation layer — never add logic to it;
> put logic in `backend/Makefile` or the owning script.

Plan for the current phase lives in `docs/engineering/roadmap.md` (acceptance
criteria — mark executed, not just coded). Beta-gate status is tracked in
`docs/engineering/beta-gate.md`.

## Workflow per change

0. **Never commit automatically.** Only commit when the user explicitly requests it (e.g. "commit", "create commits"). Leave changes staged/unstaged in the working tree otherwise
1. Small units. One context/worker/endpoint per change — never big-bang
2. **TDD (red-green-refactor), layer-adapted:**
   - Domain/use cases: unit test FIRST (type-driven — compile errors define the interface), then implement
   - Repos/workers: integration spec FIRST (round-trip, NULL columns, constraints, status transitions, idempotency), then implement; batch the slow DB cycles
   - Handlers: `app.Test` assertions on status + DTO shape; OpenAPI (`api/openapi.yaml`) is the contract
   - Bug fixes: failing regression test first — must fail for the expected reason, not a compile error
   - Exempt: migrations (DDL — verified via fresh-DB boot + round-trips), config/glue/main
3. Run `make check` — must be green
4. For schema/repo/worker changes: `make test-integration-dev` — must be green
5. Self-review the diff (`/review uncommitted`) before committing
6. Commit with Conventional Commits, imperative subject, body for migrations/security/why

## TDD layers (M3-specific)

| Layer | Test-first artifact | Cycle |
|---|---|---|
| Question generator, bias detection, prompt composer | Pure unit tests (safety rails last, CV-gap strategy) | Instant |
| WS protocol (handshake, ticket, resume, interrupt) | Protocol tests with in-process fiber + ws client — harness is the FIRST M3 task | Medium |
| LLM streaming | Mock provider with deterministic token chunks | Instant |
| Idle timeout | Inject clock; never test real 5-min waits | Instant |

## Phase planning (start of EVERY phase)

1. **Full-spec extraction**: copy ALL design requirements into the plan — deliverable tables AND Research.md config/behavior blocks (timeouts, budgets, heartbeat, probing). Reference blocks are requirements, not decoration
2. **Carryover review**: read the previous phase's "Open items (carryover)" section; schedule every item with a date or explicitly drop it as an out-of-scope DECISION (written down, not implied)
3. **Artifact-first verification**: for anything needing verification (OCR, load, concurrency), plan the fixture/harness as a task BEFORE the feature — no "implemented but unverifiable" states
4. **Deferral rule**: any deferral must record WHERE, WHY, and a DATE in the carryover section. "Deferred" without a date is an accident, not a decision
5. **Completeness review before marking phase done**: walk the full design doc line-by-line against a spec-coverage checklist (not just testing-criteria checkboxes). Happy path working ≠ spec covered

## Definition of Done (per phase)

- [ ] Doc deliverables implemented per `docs/engineering/roadmap.md` (full design doc, not just the deliverable table)
- [ ] Doc testing criteria executed with the required artifacts (fixtures/harnesses) — mark results in the phase doc
- [ ] `make check` green (includes OpenAPI drift guard + docs lint)
- [ ] `make coverage` green (floors: domain ≥70%, others ≥50%)
- [ ] Integration tests green (`make test-integration-dev`) — env-gated tests run in CI
- [ ] Schema change = migration 00X + repo + domain in the SAME change; fresh-DB boot verified (`make dev` from clean volume)
- [ ] Worker pipeline: happy path + failure path verified (status machine states observable via API)
- [ ] New API surface smoke-tested (`make smoke`); OpenAPI (`api/openapi.yaml`) updated
- [ ] Carryover section updated: closed items marked, new deferrals recorded with dates
- [ ] Docs affected by the change updated per `docs/README.md` index (status + last-reviewed refreshed)

## Documentation governance

- **One fact, one home.** Pricing → `docs/product/pricing.md`; beta status → `docs/engineering/beta-gate.md`; evaluation schema + WS frames + limits → `docs/engineering/schemas.md`; protocol contract → `api/openapi.yaml`. Everything else links.
- **Review findings append to `docs/FINDINGS.md`** — never create a new plan file for findings. Plans may group ledger rows but never own statuses.
- **Every scope flip or D-decision becomes an ADR** in `docs/adr/` the same day (template there); use `git mv`, one concern per commit, archive closed plans under `docs/plans/archive/`.
- **Compliance/policy docs describe reality** — unimplemented controls are marked Target-Qx, never present tense.
- **Language:** English-only for repo documentation.

## Conventions (learned the hard way — follow, don't rediscover)

### Database
- All app DB access via GORM (`pkg/db`). App connects as `intivai_app` (least privilege); migrations via `INTIVAI_MIGRATE_URL` + `-migrate-only`
- Tenant tables: RLS FORCED (migration 002). Every tenant query must run inside a transaction with `app.org_id` set:
  - HTTP: tenant-tx middleware (already in place)
  - Workers/services: `db.RunInTx(ctx, pool, orgID, fn)`
- Never touch RLS tables outside a tenant transaction (`db.TxFrom` required)
- Candidate-portal OTP access goes through the OTP repository layer (`Create`/`FindValidByToken`/`FindValidByCodeHash`/`IncrementAttempts`/`Consume`/`PurgeExpired`) — handlers never run OTP SQL directly, and never duplicate lookup logic
- Postgres semantics — test, don't assume:
  - A failed statement aborts the whole transaction (25P02). Unique-violation → savepoint (`tx.SavePoint` / `RollbackTo`) before recovery logic
  - A handler that wrote >=400 must not commit (tenant-tx middleware rolls back)
  - NULL scans: use pointers (`*string`, `*int`, `*[]byte`) for nullable columns — `database/sql` cannot scan NULL into scalars or `json.RawMessage`
  - Unique constraint violations: map 23505 → domain sentinel in repo, sentinel → `DomainError` in use case, `httpapi.Error` maps `DomainError`/`NotFoundError` only
- Schema drift: migration + repo SQL + domain struct are three copies — changes must touch all three in one commit; add a column or NULL-scan is a migration, not a repo-only edit
- Migration naming: `NNN_<context>_<what>.up.sql` / `.down.sql` — zero-padded, one concern per migration, never renumber existing versions. Renames are safe (golang-migrate stores no checksums) but must be rename-only commits
- pgvector: embeddings `VECTOR(384)` (default model multi-qa-MiniLM-L6-cos-v1 — bge-small gated on HF, switch via `EMBED_MODEL_NAME`); HNSW index exists

### API
- DTOs: json tags on every result (snake_case); no field-name leaks
- `PATCH` semantics: pointer fields in update commands; nil = keep
- Errors: `httpapi.Error` handles `DomainError` (400/401/403) + `NotFoundError` (404). Sentinels must be converted in the use case, never returned raw
- Never return raw `err.Error()` from handlers

### Async workers (asynq)
- Handlers register on the shared mux via `Register(mux)`
- `asynq.SkipRetry` for permanent failures (not found, invalid payload, LLM unrecoverable); return error for transient
- Idempotency: retries after commit must not re-run LLM or duplicate side effects — guard on status/state before work
- Terminal failure states set `error_message` on the entity (visible via API)
- Typed payloads + task name constants, never string literals/maps

### Security
- Prompt/context content: `ContainsInjection` rails before persistence
- JWT: only `type=auth` tokens accepted on API routes; `ws_ticket` only on `/candidate/interviews/:id/chat` (browsers pass it as `?ticket=`)
- Passwords: min 8 chars; bcrypt
- Role rank: a user may only create roles AT OR BELOW their own rank (admin > recruiter > interviewer/member) — no recruiter→admin escalation (`canCreateRole` in `create_user.go`)
- WS Origin allowlist: `INTIVAI_ALLOWED_ORIGINS` guards the chat socket (CSWSH); non-browser clients must send a matching Origin when the list is set

### Frontend
- **No TDZ in Components:** All static lookup tables (e.g. currency maps, badge styles, static option arrays) and pure formatting utilities MUST be declared at the top-level module scope outside React component render bodies to prevent Temporal Dead Zone (`ReferenceError`) runtime crashes.
- **Strict Types:** `any` strictly forbidden in TypeScript. Use domain-specific interface definitions from `types/api.ts`.

### Interviews (M3 realtime)
- WS framing: single writer goroutine (all frames through one channel); LLM streaming in its own goroutine with ctx cancel → `interrupt` stops the stream mid-response
- One active connection per interview (`sessionRegistry`); second socket rejected
- System prompt composed ONCE per connection (tenant prompt + company context version pinned at connect); LLM failure or interrupt still dispatches the next question (`turnState.sendQuestionOnce`)
- Session pinning: `resume` must echo the ticket's `session_id`, else `error: session mismatch`

## Tests

- Unit (pure domain): scoring engine, semantic, prompt validation, job/iam/cv domains, CVService compensation, PDF extraction, DTO json-tag contract — always run `make check`
- Integration (env-gated, skip without `TEST_DATABASE_URL`): RLS isolation, pg memory bank, re-score savepoint, repo round-trips (NULL/jsonb), score + extract worker pipelines, chat flow (ticket → start → answer → tokens → next; interrupt; second-connection rejection; session mismatch; LLM-error advance), interview service flow (ticket states, expiry/revoke, compose rails). Run with `make test-integration-dev`; executed in CI with real postgres+redis services
- New repo/worker logic → integration test (that is where this project's bugs lived)
- E2E & Browser Testing (Playwright):
  - **Single-Worker Execution:** Set `workers: 1` in `playwright.config.ts` when running against a local shared backend/DB instance to prevent session token collisions and state mutation conflicts.
  - **Form State Settling:** Always wait for navigation or state assertions (e.g. `await expect(page).toHaveURL(/.*\/dashboard/)`) after login/form submits before subsequent `page.goto()` calls.
  - **Consumable Tokens:** For single-use tokens (OTP, magic links), generate fresh tokens via the API or inject signed JWTs directly into `localStorage`.
  - **Valid PDF Fixtures:** Never use arbitrary truncated byte strings; always use `buildMinimalPDF` to generate valid ISO 32000-1 cross-reference tables.

## Environment

- `make dev` uses `docker-compose.yml` + `docker-compose.dev.yml` (non-conflicting host ports: postgres 5433, redis 6380, minio 9002/9003, app 8081); prod = `docker-compose.prod.yml` (Caddy-only ports) + `--env-file .env.prod`
- FE dev: `frontend/` Vite on :5173 (proxies `/api` + WS to :8081); `INTIVAI_ALLOWED_ORIGINS` must include the FE origin (CSWSH + CORS)
- Docker: BuildKit is enabled (no `DOCKER_BUILDKIT=0`) — verified working for the app, sandboxd, and sandbox execution images
- Tesseract OCR in image: needs `tesseract-ocr-data-eng` + `poppler-utils` — do not remove
- LLM API key: `INTIVAI_LLM_API_KEY`; absent → extract marks `failed_extract` honestly (use `POST /cvs/:id/extract` to retry after key is set)

<!-- caveman-begin -->
Respond terse like smart caveman. All technical substance stay. Only fluff die.

Rules:
- Drop: articles (a/an/the), filler (just/really/basically), pleasantries, hedging
- Fragments OK. Short synonyms. Technical terms exact. Code unchanged.
- Pattern: [thing] [action] [reason]. [next step].
- Not: "Sure! I'd be happy to help you with that."
- Yes: "Bug in auth middleware. Fix:"

Switch level: /caveman lite|full|ultra|wenyan-lite|wenyan-full|wenyan-ultra
Stop: "stop caveman" or "normal mode"

Auto-Clarity: drop caveman for security warnings, irreversible actions, user confused. Resume after.

Boundaries: code/commits/PRs written normal.
<!-- caveman-end -->
