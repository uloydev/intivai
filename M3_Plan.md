# M3 Plan — Chat Interview (Phase 3)

Scope from `AI_Interviewer_Phases.md` §Phase 3 (Week 8-10). Status tracking for
deliverables and doc testing criteria — mark boxes as executed, not just coded.

## Deliverables

| Area | Files | Status |
|------|-------|--------|
| Interview domain: Interview entity, Question VO, Answer VO | `internal/interview/domain/` | [ ] |
| Question generator: CV-gap strategy, bias detection | `internal/interview/domain/service/question_generator.go` | [ ] |
| Prompt composer: default + tenant prompt + company context + safety rails (pinned LAST) | `internal/interview/domain/service/prompt_composer.go` | [ ] |
| Context management: sliding window, tiktoken counting | `internal/interview/domain/service/` | [ ] |
| WebSocket hub: connections, heartbeat, per-interview room | `internal/interview/api/chat_handler.go` | [ ] |
| API: `POST /interviews`, `WS /interviews/:id/chat` | `internal/interview/api/` | [ ] |
| Reconnection: last unanswered question stored, resume | `internal/interview/application/` | [ ] |
| WS ticket auth: 10-min JWT bound to session+interview (Research §3); candidates never use internal JWT | `internal/iam/` + interview api | [ ] |

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

## TDD order (M3)

1. WS protocol test harness FIRST (in-process fiber + ws client) — nothing realtime proceeds without it
2. Question generator + bias detection: unit tests first (type-driven), then implementation
3. Prompt composer: unit tests — safety rails pinned LAST asserted first
4. LLM streaming: mock provider with deterministic token chunks; ChatStream contract pinned by test
5. Interview domain (entity, question/answer VOs): pure unit tests
6. Repos (interview/transcript persistence): integration spec first, batched
7. Handlers + WS handler: protocol + app.Test against the harness
8. Idle timeout: injectable clock; never test real 5-min waits

## Doc testing criteria (execute, then check)

- [ ] WS connects, handshake completes (candidate uses WS ticket, not JWT)
- [ ] WS upgrade without a valid ticket is rejected
- [ ] DeepSeek Flash returns streaming response
- [ ] Questions generated based on CV gaps
- [ ] Selected questions persisted to question bank (reuse + audit)
- [ ] Answers stored in PostgreSQL
- [ ] Reconnection resumes from last unanswered question
- [ ] Bias detection catches prohibited questions
- [ ] Idle timeout disconnects after 5 minutes
- [ ] 100 concurrent WebSocket connections stable
- [ ] Prompt composer: tenant prompt + company context + safety rails composed correctly
- [ ] Safety rails always last (tenant cannot override)

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
