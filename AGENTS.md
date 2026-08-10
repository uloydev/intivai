# Intivai — Engineering Workflow

## Project

- Monorepo: `backend/` (Go), `docker-compose.yml` (stack), `.github/workflows/ci.yml`
- Architecture reference: `AI_Interviewer_Phases.md` (per-phase deliverables + testing criteria), `AI_Interviewer_Research.md` (design decisions)
- Phases: M1 foundation, M2 job/CV/screening/context, M3 interviews (next)

## Commands (run from `backend/`)

| Command | Purpose | When |
|---|---|---|
| `make check` | gofmt + golangci-lint + vet + build + unit tests | Before EVERY commit. Must be green |
| `make lint` | golangci-lint (config: `.golangci.yml`) | Before commit |
| `make coverage` | per-package coverage floors (domain ≥70%, others ≥50%; needs stack up) | Before commit |
| `make test-integration-dev` | integration tests against local compose (needs stack up) | After any schema/repo/worker change |
| `make test` | unit tests | Before commit |
| `make dev` | boot full stack (docker compose, classic builder) | Start of a session |
| `make smoke` | end-to-end API scenario against running stack (`CV_PDF` env or `/tmp/kilo/cv.pdf`) | After any API change |
| `make migrate` | apply migrations (admin URL) | After fresh DB or new migration |
| `go run ./cmd/server -migrate-only` | same as `make migrate` | — |

Full check before commit:

```bash
make check && make coverage && make test-integration-dev
```

Plan for the current phase lives in `M3_Plan.md` (acceptance criteria from
`AI_Interviewer_Phases.md` — mark executed, not just coded).

## Workflow per change

0. **Never commit automatically.** Only commit when the user explicitly requests it (e.g. "commit", "create commits"). Leave changes staged/unstaged in the working tree otherwise
1. Small units. One context/worker/endpoint per change — never big-bang2. **TDD (red-green-refactor), layer-adapted:**
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

- [ ] Doc deliverables implemented per `AI_Interviewer_Phases.md` (full design doc, not just the deliverable table)
- [ ] Doc testing criteria executed with the required artifacts (fixtures/harnesses) — mark results in the phase doc
- [ ] `make check` green
- [ ] `make coverage` green (floors: domain ≥70%, others ≥50%)
- [ ] Integration tests green (`make test-integration-dev`) — env-gated tests run in CI
- [ ] Schema change = migration 00X + repo + domain in the SAME change; fresh-DB boot verified (`make dev` from clean volume)
- [ ] Worker pipeline: happy path + failure path verified (status machine states observable via API)
- [ ] New API surface smoke-tested (`make smoke`); OpenAPI (`api/openapi.yaml`) updated
- [ ] Carryover section updated: closed items marked, new deferrals recorded with dates

## Conventions (learned the hard way — follow, don't rediscover)

### Database
- All app DB access via GORM (`pkg/db`). App connects as `intivai_app` (least privilege); migrations via `INTIVAI_MIGRATE_URL` + `-migrate-only`
- Tenant tables: RLS FORCED (migration 002). Every tenant query must run inside a transaction with `app.org_id` set:
  - HTTP: tenant-tx middleware (already in place)
  - Workers/services: `db.RunInTx(ctx, pool, orgID, fn)`
- Never touch RLS tables outside a tenant transaction (`db.TxFrom` required)
- Postgres semantics — test, don't assume:
  - A failed statement aborts the whole transaction (25P02). Unique-violation → savepoint (`tx.SavePoint` / `RollbackTo`) before recovery logic
  - A handler that wrote >=400 must not commit (tenant-tx middleware rolls back)
  - NULL scans: use pointers (`*string`, `*int`, `*[]byte`) for nullable columns — `database/sql` cannot scan NULL into scalars or `json.RawMessage`
  - Unique constraint violations: map 23505 → domain sentinel in repo, sentinel → `DomainError` in use case, `httpapi.Error` maps `DomainError`/`NotFoundError` only
- Schema drift: migration + repo SQL + domain struct are three copies — changes must touch all three in one commit; add a column or NULL-scan is a migration, not a repo-only edit
- Migration naming: `NNN_<context>_<what>.up.sql` / `.down.sql` — zero-padded, one concern per migration, never renumber existing versions. Renames are safe (golang-migrate stores no checksums) but must be rename-only commits
- pgvector: embeddings `VECTOR(384)` bge-small; HNSW index exists

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
- JWT: only `type=auth` tokens accepted on API routes; `ws_ticket` only on `/candidate/interviews/:id/chat`
- Passwords: min 8 chars; bcrypt
- Role rank: a user may only create roles AT OR BELOW their own rank (admin > recruiter > interviewer/member) — no recruiter→admin escalation (`canCreateRole` in `create_user.go`)
- WS Origin allowlist: `INTIVAI_ALLOWED_ORIGINS` guards the chat socket (CSWSH); non-browser clients must send a matching Origin when the list is set

### Interviews (M3 realtime)
- WS framing: single writer goroutine (all frames through one channel); LLM streaming in its own goroutine with ctx cancel → `interrupt` stops the stream mid-response
- One active connection per interview (`sessionRegistry`); second socket rejected
- System prompt composed ONCE per connection (tenant prompt + company context version pinned at connect); LLM failure or interrupt still dispatches the next question (`turnState.sendQuestionOnce`)
- Session pinning: `resume` must echo the ticket's `session_id`, else `error: session mismatch`

## Tests

- Unit (pure domain): scoring engine, semantic, prompt validation, job/iam/cv domains, CVService compensation, PDF extraction, DTO json-tag contract — always run `make check`
- Integration (env-gated, skip without `TEST_DATABASE_URL`): RLS isolation, pg memory bank, re-score savepoint, repo round-trips (NULL/jsonb), score + extract worker pipelines, chat flow (ticket → start → answer → tokens → next; interrupt; second-connection rejection; session mismatch; LLM-error advance), interview service flow (ticket states, expiry/revoke, compose rails). Run with `make test-integration-dev`; executed in CI with real postgres+redis services
- New repo/worker logic → integration test (that is where this project's bugs lived)

## Environment

- `make dev` uses `docker-compose.yml` + `docker-compose.dev.yml` (non-conflicting host ports: postgres 5433, redis 6380, minio 9002/9003, app 8081)
- Docker buildx is broken on the dev machine: `DOCKER_BUILDKIT=0 docker build` (classic builder)
- Tesseract OCR in image: needs `tesseract-ocr-data-eng` + `poppler-utils` — do not remove
- DeepSeek key: `INTIVAI_DEEPSEEK_API_KEY`; absent → extract marks `failed_extract` honestly (use `POST /cvs/:id/extract` to retry after key is set)
