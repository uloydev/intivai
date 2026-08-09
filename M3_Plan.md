# M3 Plan — Chat Interview (Phase 3)

Scope from `AI_Interviewer_Phases.md` §Phase 3 (Week 8-10). Status tracking for
deliverables and doc testing criteria — mark boxes as executed, not just coded.

## Open items — carryover (from M2 + M3; review at every phase start)

| # | Item | Source | Scheduled | Status |
|---|------|--------|-----------|--------|
| 1 | Context management: sliding window (last 10 Q&A) + tiktoken budget | M3 deliverable | 2026-08-12 | [x] done 2026-08-10 (TrimContext + budget in domain/service/context.go; history seeded from transcript via RecentContext; handler window+budget enforced) |
| 2 | 100 concurrent WS connections load check (harness: Go load client) | M3 criterion | 2026-08-13 | [x] done 2026-08-10 (cmd/loadcheck + make load-ws; 100/100 in ~140ms; found+fixed tenant-tx pool deadlock) |
| 3 | Interview duration cap (30 min) + per-question timeout (3 min) | Research §2 config | 2026-08-12 | [x] done 2026-08-10 (MaxInterviewDuration in domain ExpireIfNeeded; PerQuestionTimeout as WS read deadline) |
| 4 | Dynamic follow-up probing (weakness→probe, strength→next) | Research §2 strategy | 2026-08-14 (post-load) | [x] done 2026-08-10 (deterministic probe on <8-word answers, inserted with renumbering, banked) |
| 5 | Smoke extended with interview endpoints | M3 plan | 2026-08-11 | [ ] |
| 6 | `go test -race` in CI (backend job) | cross-cutting | 2026-08-11 | [ ] |
| 7 | Local embeddings fastembed bge-small (columns+HNSW ready) | M2 deferral | 2026-08-15 | [x] done 2026-08-10 (pure-Go cybertron, 384-dim verified; bge-small gated on HF → multi-qa default, EMBED_MODEL_NAME to switch) |
| 8 | Embedding-based semantic recall (banks + SemanticMatch) | M2 deferral | 2026-08-15 (after 7) | [x] done 2026-08-10 (pg bank cosine Recall + cosine SemanticScoreWithEmbedder; keyword stays default) |
| 9 | Scanned-PDF OCR fixture + functional verification | M2 verification | 2026-08-12 | [x] done 2026-08-10 (in-process scanned PDF → pdftoppm → tesseract, verified in app container) |
| 10 | Context version pinned on interview row (audit) | Phases Phase 2 | 2026-08-13 | [x] done 2026-08-10 (migration 007 + pinned at CreateInterview, integration-tested) |
| 11 | Server heartbeat (PingInterval/PongWait) | Research §2 config | 2026-08-14 | [x] done 2026-08-10 (ping 30s / pong wait 10s, silent client dropped, unit-tested) |
| 12 | Multi-instance session registry (Redis) | cross-cutting | deferred — needs multi-instance deployment (DECISION) | [ ] |

## Deliverables

| Area | Files | Status |
|------|-------|--------|
| Interview domain: Interview entity, Question VO, Answer VO | `internal/interview/domain/` | [x] state machine + clock tested |
| Question generator: CV-gap strategy | `internal/interview/domain/service/question_generator.go` | [x] unit-tested |
| Bias detection: protected-class rules | `internal/interview/domain/service/bias.go` | [x] unit-tested |
| Prompt composer: default + tenant prompt + company context + safety rails (pinned LAST) | `internal/interview/domain/service/prompt_composer.go` | [x] unit-tested |
| Clock injection (idle timeout, expiry) | `internal/interview/domain/clock.go` | [x] |
| Context management: sliding window, tiktoken counting | `internal/interview/domain/service/context.go` | [x] unit-tested + chat flow asserts window |
| WebSocket hub: connections, heartbeat, per-interview room | `internal/interview/api/chat_handler.go` | [x] chat flow integration-tested (ticket → start → question → answer → tokens → next) |
| API: `POST /interviews`, `POST /candidate/interviews/:id/ticket`, `WS /candidate/interviews/:id/chat` | `internal/interview/api/` | [x] live-verified |
| Reconnection: last unanswered question stored, resume | `internal/interview/application/` | [x] resume sends start + current question (CurrentState) |
| WS ticket auth: 10-min JWT bound to session+interview (Research §3); candidates never use internal JWT | `internal/iam/` + interview api | [x] ws_ticket rejected on API routes, accepted on chat |
| Repos: interview, invitation token, question bank | `internal/interview/infrastructure/persistence/` | [x] round-trip + token lifecycle integration-tested |

LLM provider already exists (`internal/llm`, DeepSeek streaming + structured
output) — verify `ChatStream` against the interview flow.

## Migration 006 (expected)

- `interviews` + `interview_tokens` tables already exist (migration 001) — verify
  columns fit M3 (transcript JSONB, last_question_idx, expires_at, tokens
  used_at/revoked_at) before writing new migrations
- Likely additions: messages table (or transcript reuse), question bank
  persistence (exists: `questions` table)

## Workflow gates (from AGENTS.md)

- [ ] `make check` green
- [ ] `make test-integration-dev` green (new repo/worker logic → integration test)
- [ ] `make coverage` green (floors: domain ≥70, others ≥50)
- [ ] Worker/WS happy path + failure path verified (status machine observable via API)
- [ ] Fresh-DB boot verified (`make dev` from clean volume)
- [ ] `make smoke` extended with interview endpoints
- [ ] Schema change = migration + repo + domain in SAME change
- [ ] Completeness review: full design doc (Research config blocks incl.) walked against spec-coverage checklist before phase is marked done

## TDD order (M3) — progress

1. [x] WS protocol test harness FIRST (in-process fiber + ws client) — upgrade/echo/ping proof + full chat flow test
2. [x] Question generator + bias detection: unit tests first, then implementation
3. [x] Prompt composer: unit tests — safety rails pinned LAST asserted first
4. [x] LLM streaming: mock provider with deterministic token chunks + httptest SSE contract; ChatStream fallback pinned
5. [x] Interview domain (entity, question/answer VOs): pure unit tests + frozen-clock idle/expiry
6. [x] Repos (interview/transcript, token lifecycle via definer, question bank): integration spec first
7. [x] Handlers + WS handler: chat flow integration test (ticket → start → question → answer → tokens → next; bad ticket rejected)
8. [x] Idle timeout: injectable clock; ws read deadline uses it

Remaining for M3 done-criteria: sliding-window context management (tiktoken), 100-conn load check, smoke extension with interview endpoints, live DeepSeek streaming verification (needs API key).

## Doc testing criteria (execute, then check)

- [x] WS connects, handshake completes (candidate uses WS ticket, not JWT)
- [x] WS upgrade without a valid ticket is rejected
- [x] DeepSeek Flash returns streaming response (live-verified with real key)
- [x] Questions generated based on CV gaps
- [x] Selected questions persisted to question bank (reuse + audit)
- [x] Answers stored in PostgreSQL
- [x] Reconnection resumes from last unanswered question (resume sends start + current question; session mismatch rejected)
- [x] Bias detection catches prohibited questions
- [x] Idle/per-question timeout: silent candidate disconnects (read deadline = PerQuestionTimeout 3m; 30-min duration cap expires via domain state machine, frozen-clock tested)
- [x] 100 concurrent WebSocket connections stable (cmd/loadcheck: 100/100 pass, ~140ms)
- [x] System prompt composer: tenant prompt + company context + safety rails composed correctly (live)
- [x] Safety rails always last (tenant cannot override)
- [x] Interrupt stops the AI mid-response (streaming goroutine + cancel; live-verified)
- [x] Single active connection per interview (second socket rejected)

## Design constraints (learned in M1/M2 — apply, don't rediscover)

- RLS: all interview queries inside tenant tx (`db.RunInTx`); candidate path via
  security-definer `validate_interview_token` (exists, migration 001)
- WS ticket: `TokenTypeWSTicket` + `Extra` claims already supported by
  `JWTProvider` (jwt_provider.go) — API routes reject non-auth tokens already
- asynq: interviews are realtime (not workers) — keep LLM streaming in the WS
  handler; workers only for post-interview evaluation (Phase 4)
- Idempotency: reconnection must not re-ask answered questions
- Injection: company context validated by `ContainsInjection` at upload (M2);
  composer pins safety rails last
- Realtime (landed): single writer goroutine per socket; LLM streaming in its
  own goroutine with ctx cancel (interrupt); single active connection per
  interview (`sessionRegistry`); prompt composed once per connection; LLM
  error / interrupt both dispatch the next question via `turnState`

---

## Beta Gate — definition of "beta started" (EM decision, 2026-08-10)

Scope change approved: P4a (evaluation core + recruiter dashboard-lite + invite
+ consent) and P6a (deploy + backup + alerting-lite) move INTO the MVP; the
candidate chat UI becomes a P3 deliverable. Beta = Phase 0 cohort (5 pilots).

| # | Gate item | Status |
|---|-----------|--------|
| 1 | P4a: evaluation LLM → report JSON; `evaluation` frame carries real scores | [x] |
| 2 | P4a: `GET /interviews` + `GET /interviews/:id` + `GET /candidates/:id/report` (JSON) | [x] |
| 3 | P3 FE: candidate chat UI (ticket → consent → interview → evaluation frame) | [x] |
| 4 | P4a FE: recruiter dashboard-lite (CV/job upload, interview list, result view) | [x] |
| 5 | Invite flow: shareable interview URL from invitation token | [x] |
| 6 | Consent capture: `consent_given` recorded at interview start | [x] |
| 7 | Live DeepSeek streaming verified with real key (smoke + Playwright E2E) | [x] |
| 8 | Fresh-volume boot 001–007 (`make dev` from clean volume) | [x] |
| 9 | Deploy: compose on VPS, domain + TLS, env management, push pipeline | [~] pipeline + overlay ready; needs VPS/domain/secrets |
| 10 | Backup & DR: postgres dump + MinIO mirror → backup bucket; restore test executed | [~] scripts ready; needs host cron + first restore test |
| 11 | Error alerting lite (Sentry Go, DSN-gated) | [~] wired; needs DSN |
| 12 | 5 pilot companies onboarded; feedback channel + retention criteria set | [ ] business |
| 13 | `make check` + `make coverage` + `make test-integration-dev` green | [x] |
| 14 | All commits pushed; deploy from tagged release | [ ] |

Beta starts when gates 1–13 are checked and 14 is done — no silent partial beta.

## Fix plan — M1–M3 gap closure (2026-08-10 → 08-15)

Order = date + dependency. Each item: TDD per AGENTS, `make check` +
`make test-integration-dev` green, mark carryover row [x] with date.

### F0. Commit pending work (08-10, first)
Uncommitted across working tree — split logical commits:
1. `feat(interview): sliding-window context + token budget` — `domain/service/context.go` + tests, `RecentContext`, handler windowing
2. `feat(interview): 30-min duration cap + 3-min per-question timeout` — domain consts + `ExpireIfNeeded`, read deadline
3. `feat: extend smoke with interview flow` — scripts/smoke.sh (ws, ticket, chat)
4. `ci: add go test -race to backend job` — .github/workflows/ci.yml
5. my session leftovers: Makefile dev/redis-clear, docker-compose.dev.yml `!override`, AGENTS.md rows

### F1. Scanned-PDF OCR fixture (08-12) — item 9
- Fixture: scanned PDF (image-only, 1-2 pages) committed under `testdata/`; generation script documented
- Test: parse worker → `pdftoppm` + tesseract → extracted text contains expected strings; failure path `failed_ocr` honest
- Files: `internal/cv/` test + fixture; verify tesseract-ocr-data-eng + poppler in Dockerfile still installed
- Gate: `make test-integration-dev` (OCR runs in containerized CI)

### F2. Context version pinned on interview row (08-13) — item 10
- Migration 007: `interviews.context_version INT` (nullable) + `contexts` version FK semantics
- `CreateInterview` writes latest company-context version at creation (audit trace)
- Repo round-trip test + API asserts version on interview detail
- Files: migration + `internal/interview/` repo/domain/application in SAME change

### F3. 100-conn WS load harness (08-13) — item 2
- `cmd/loadcheck` (or `scripts/load_ws.go`): Go client, N=100 concurrent sockets, ticket-per-conn (or shared interview), answer+interrupt mix, assert all streams complete, errors = 0, duration budget
- Harness FIRST, then run against dev stack; record numbers in M3_Plan criterion
- Note: single-connection-per-interview rule → harness must use 100 distinct interviews (or accept 99 rejections as expected behavior — decide in harness)

### F4. Server heartbeat (08-14) — item 11
- Handler: server ping on `PingInterval=30s` (writer goroutine ticker), `PongWait=10s` read deadline margin; pong frames from client honored
- Protocol: keep client `ping`→`pong` (already exists); add server-initiated ping
- Tests: ws harness — stale client (no pong) dropped; active client survives
- Also verify `HandshakeTimeout: 10s` in upgrader config (fiberws)

### F5. Dynamic follow-up probing (08-14, post-load) — item 4
- Domain: probe strategy (weakness→probe question, strength→next) in `question_generator.go` or new `probe.go`; unit tests: weak answer → probe generated from CV-gap skill; strong → advance
- Handler: after answer, if probe applies → send probe as next question (persisted like normal questions)
- No LLM for probing (deterministic, cheap) — only LLM for responses

### F6. Local embeddings fastembed bge-small (08-15) — item 7
- `internal/llm/embed.go` or `internal/embedding/`: fastembed Go adapter (pure-Go ONNX runtime or fastembed-go), model bge-small-en-v1.5, 384d — no network at inference
- `Embed()` impl replaces "not implemented"; unit test: known sentence → dims=384, stable
- Column `VECTOR(384)` + HNSW already in migration 002

### F7. Embedding-based semantic recall (08-15) — item 8
- Postgres bank: `Recall` cosine path (`embedding <=> $1` ORDER BY, LIMIT), fallback LIKE when embedding NULL
- Sync worker: compute embedding at `Remember` (async, non-blocking; index-time cost accepted)
- SemanticMatch in scoring swaps keyword overlap → cosine (weights unchanged)
- Tests: bank integration — same-tenant semantic recall ranks exact match first; cross-tenant isolation still holds

### F8. Close-out (08-15)
- Carryover table: mark all closed, remove DECISION deferral item 12 (keep)
- `make check` + `make test-integration-dev` + `make coverage` green; fresh-DB boot; update phases doc boxes M2/M3
- Self-review diff before commit; conventional commits per item

