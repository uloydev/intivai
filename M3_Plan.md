# M3 Plan — Chat Interview (Phase 3)

Scope from `AI_Interviewer_Phases.md` §Phase 3 (Week 8-10). Status tracking for
deliverables and doc testing criteria — mark boxes as executed, not just coded.

## Open items — carryover (from M2 + M3; review at every phase start)

| # | Item | Source | Scheduled | Status |
|---|------|--------|-----------|--------|
| 1 | Context management: sliding window (last 10 Q&A) + tiktoken budget | M3 deliverable | 2026-08-12 | [ ] |
| 2 | 100 concurrent WS connections load check (harness: Go load client) | M3 criterion | 2026-08-13 | [ ] |
| 3 | Interview duration cap (30 min) + per-question timeout (3 min) | Research §2 config | 2026-08-12 | [ ] |
| 4 | Dynamic follow-up probing (weakness→probe, strength→next) | Research §2 strategy | 2026-08-14 (post-load) | [ ] |
| 5 | Smoke extended with interview endpoints | M3 plan | 2026-08-11 | [ ] |
| 6 | `go test -race` in CI (backend job) | cross-cutting | 2026-08-11 | [ ] |
| 7 | Local embeddings fastembed bge-small (columns+HNSW ready) | M2 deferral | 2026-08-15 | [ ] |
| 8 | Embedding-based semantic recall (banks + SemanticMatch) | M2 deferral | 2026-08-15 (after 7) | [ ] |
| 9 | Scanned-PDF OCR fixture + functional verification | M2 verification | 2026-08-12 | [ ] |
| 10 | Context version pinned on interview row (audit) | Phases Phase 2 | 2026-08-13 | [ ] |
| 11 | Server heartbeat (PingInterval/PongWait) | Research §2 config | 2026-08-14 | [ ] |
| 12 | Multi-instance session registry (Redis) | cross-cutting | deferred — needs multi-instance deployment (DECISION) | [ ] |

## Deliverables

| Area | Files | Status |
|------|-------|--------|
| Interview domain: Interview entity, Question VO, Answer VO | `internal/interview/domain/` | [x] state machine + clock tested |
| Question generator: CV-gap strategy | `internal/interview/domain/service/question_generator.go` | [x] unit-tested |
| Bias detection: protected-class rules | `internal/interview/domain/service/bias.go` | [x] unit-tested |
| Prompt composer: default + tenant prompt + company context + safety rails (pinned LAST) | `internal/interview/domain/service/prompt_composer.go` | [x] unit-tested |
| Clock injection (idle timeout, expiry) | `internal/interview/domain/clock.go` | [x] |
| Context management: sliding window, tiktoken counting | `internal/interview/domain/service/` | [ ] |
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
- [x] Idle timeout disconnects after 5 minutes (read deadline + injectable clock)
- [ ] 100 concurrent WebSocket connections stable (load check pending)
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
