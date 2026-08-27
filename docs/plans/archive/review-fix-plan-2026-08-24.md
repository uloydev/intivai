# Review Fix Plan — Deep Audit 2026-08-24

> Status: superseded by `docs/plans/archive/intivai-remediation-plan-2026-08-26.md` · Owner: EM
> Source: four-agent deep review executed 2026-08-24 (docs-sync, ops/CI/security,
> Go backend, React frontend). Every claim below was verified against code by
> the auditing agents; file:line references preserved from their reports.
>
> Governance: **statuses live only in `docs/FINDINGS.md`** (source tag `R24`).
> This plan groups ledger rows into execution batches; it never owns a status.
> New findings are appended to the ledger when their batch starts, with
> evidence quotes from this plan.

## How work proceeds

- One concern per commit; Conventional Commits; never commit without explicit instruction.
- TDD layer per repo AGENTS.md: domain/use case → unit test first; repos/workers → integration spec first; handlers → `app.Test` assertions; bug fixes → failing regression test that fails for the right reason.
- Gates after every batch: `make check` green; schema/repo/worker changes also `make test-integration-dev`; API surface changes also `make smoke` + OpenAPI update.
- Schema change = migration + repo SQL + domain struct in one commit; fresh-DB boot verified once per migration-bearing batch (`make dev` from clean volume).

---

## Decision register (CONFIRMED 2026-08-24)

| # | Decision | Options | Decision |
|---|---|---|---|
| DA | Voice route mounted vs docs claiming unmounted (ADR-0006, openapi.yaml:789, design-decisions.md:640, roadmap.md:313 all say unmounted; `main.go:430` mounts it) | (a) Unmount route, align code to ADR-0006; (b) Keep mounted, rewrite docs + ADR amendment | **(b) KEEP MOUNTED.** Docs rewritten to match reality; ADR-0007-style amendment records the flip (new ADR superseding ADR-0006's mount clause). OpenAPI spec gains the `/interviews/{id}/voice` path. FE voice-page defects (G4 token leak, G5 mic leak) become mandatory fixes in Batches 3/6 |
| DB | Portal magic token delivery | (a) Interim strips + caps; (b) Proper email delivery | **BOTH STAGED** — (a) Batch 1a immediately, (b) Batch 1b after |
| DC | Backend Sentry | (a) Wire it; (b) Demote docs claim | **WIRE IT** (Batch 5.6). Beta-gate #11 + roadmap P6 Sentry criterion stay untouched in Batch 0 — they flip to truthful-closed only when wiring lands (never both half-done) |
| DD | Consent auto-tick on invite autostart (`Invite.tsx:51-56`) | (a) Keep implied consent; (b) Require explicit click | **EXPLICIT CLICK** (Batch 6 minors sweep) |
| DE | Playwright E2E in CI | (a) PR-blocking; (b) Nightly first | **NIGHTLY FIRST** (Batch 7.3); promote to required after F18 stall fixed |
| DF | Docker socket containment | (a) Socket proxy; (b) Defer documented | **ADD PROXY** (Batch 7.11) with ADR-0002 amendment note |

---

## Batch 0 — Truth sync & ledger reconciliation (no runtime behavior change)

Purpose: make every doc tell the truth before fixing code, so later batches flip honest checkboxes instead of dishonest ones. All small; single day total.

0.1 **Append R24 findings to `docs/FINDINGS.md`** — new source tag R24 registered in Sources line; rows per mapping table at bottom of this plan. Close two stale rows with evidence:
- D2 → closed: OTP repo layer exists (`postgres_portal_repo.go:83,98,128`; handler repo-only `candidate_portal_handler.go:89,150,152`). AGENTS.md claim correct today.
- F10 → closed: DOCX parsing shipped (`parse_worker.go:19`, dispatch :73–76, extraction :206–228).
Reconcile needs-verify rows answered by this audit (append Notes, do not rewrite history): A2/A3 (smoke job reality), A6 (prod overlay now has healthchecks/non-root app — base compose still bare), A7 (prod `:?` fail-fast verified), C3 (SSRF guards verified), C6 (rate limit present but telemetry exempt — see D20), B7 (voice-page token leak CONFIRMED real — see G4; other sub-items still need walk), D11 (overlap confirmed real — D16), F17 (TDZ clean, static tables module-scope).

0.2 **Fix falsely-checked claims** (docs commits):
- beta-gate.md #11 + roadmap.md:418 Sentry criterion → DEFERRED to Batch 5 completion: flip to closed only when wiring lands (DC decision).
- roadmap.md P5 criteria `[x]` STT/audio-playback rows → uncheck to match mocked pipeline (design-decisions.md:49); roadmap.md:313 header "route NOT mounted" → rewritten: mounted, gated demo per DA.
- design-decisions.md:55 "migrations 001–009" → 001–025; :640 "NOT mounted in production routing" → "mounted, demo-gated" per DA; beta-gate.md:22 "001–007" → 001–025.
- schemas.md line anchors: CV_MAX_UPLOAD_MB → config.go:153; ctxMaxBytes → context_service.go:29.
- ADR for the voice-mount flip: new `docs/adr/0008-voice-demo-route-mounted.md` superseding ADR-0006's unmount clause (same-day ADR rule), with rationale: route is auth-guarded, demo-deferred feature kept reachable for staged demos; openapi.yaml:789 comment updated accordingly.

0.3 **Archive executed plan**: `git mv docs/plans/active/docs-reorganization-plan.md docs/plans/archive/`; tick its §8 DoD boxes; fix docs/README.md index entry + add missing entries (design-system/*.md, frontend/README.md).

0.4 **OpenAPI hygiene**:
- Deduplicate top-level keys `/health`, `/live`, `/ready` (openapi.yaml:43/:1177, :52/:1185, :61/:1193).
- Fix drift-guard blind spot (`scripts/check-openapi-drift.sh:14` greps only direct route literals): extend to scan helper `Register*Routes` methods too (e.g., grep `router.(Get|Post|...)` inside `internal/**/api/*.go` Register functions), or switch to a route-list fixture asserted against spec. Test-first: add a guard self-test proving a helper-mounted route IS detected.
- Per DA (keep mounted): spec GAINS `GET /interviews/{id}/voice` path entry matching voice_handler.go registration; the "intentionally NOT mounted" comment at openapi.yaml:789–790 replaced with demo-gated note referencing ADR-0008.

0.5 **PDF honesty fix** (B1-class, customer-facing): `evaluation/application/pdf.go:449–451` defaults IntegrityScore 100 on zero events; :474 prints static "Verified Session Authenticity". Change: zero proctoring events → render "No telemetry recorded", omit score line entirely, drop authenticity string unless events exist AND no flags. Unit test first on PDF builder output.

Acceptance: docs lint green; drift guard catches injected fake route in test; ledger reflects R24; `make check` green.

---

## Batch 1 — Critical security & data-integrity

### 1a. Portal auth bypass (ledger C9) — highest priority

Attack chain (verified): public apply returns `portal_token` for any supplied email → verify → 7-day candidate JWT for victim email → cross-org application listing leaks scores + live `invitation_token` → attacker joins victim interview.

Fixes, three layers, one commit each:

1. **Lookup view strip (migration)** — `candidate_applications_lookup` must not expose `invitation_token` (postgres_portal_repo.go:143–146 SELECTs it). Migration drops/recreates view without the column; repo SQL + any domain struct updated same commit. Integration spec FIRST: assert lookup columns exclude invitation_token; assert existing portal flows unaffected.
2. **Apply response strip** — regression test first: POST `/public/jobs/:id/apply` response MUST NOT contain `portal_token` field (job/api/public_handler.go:199–215). Handler creates token row, returns 201 without credential.
3. **Abuse caps** — per-email daily apply cap + per-IP bucket on apply route (rate-limit infra exists; extend keyFn). Unit-test cap logic pure; integration-test 429 shape.

### 1b. Magic-link email delivery (completes DB decision)

Email worker already sends magic links (ledger F1 verified). Apply flow switches to enqueueing the same email type; response contains only generic confirmation. Integration test: apply → email-worker task enqueued with recipient email; no token anywhere in HTTP response. Verify SMTP env contract in `.env.prod.example`.

### 1c. DELETE 204 crash (ledger G1)

Regression tests first (vitest): `request()` returns undefined for 204 empty body — currently throws `TypeError` (api.ts:34,58). Fix: early-return on `res.status === 204`. Then per-call-site assertions: GDPR erase logs out + clears storage (CandidatePortal.tsx:110); CV delete invalidates cache (CVs.tsx:178); webhook + context deletes toast success. Add 404-on-second-delete assertion where cheap.

Acceptance: failing-tests-first demonstrated; `make check && make test-integration-dev` green; `make smoke` extended with an apply→portal round-trip asserting no token leak (add scenario step).

---

## Batch 2 — Tenant isolation & data exposure

2.1 **Scorecard notify dead query (D15)** — `evaluation_worker.go:174–175` queries RLS `users` outside tenant tx → always zero rows, silent. Fix: wrap in `db.RunInTx(ctx, pool, orgID, ...)` mirroring main.go:242–268 webhook pattern. Integration spec FIRST: seed org users → run notify path → assert recipients resolved (use injectable mailer capturing sends).

2.2 **PDF cache-hit authz bypass (D16)** — `evaluation/service.go:289–302` streams cached PDF before ownership check. Fix: resolve interview + actor-org check BEFORE cache existence check; cache key stays interview-scoped. Integration test: recruiter of org B requesting org A interview UUID → 404 on both cold and warm cache paths (warm-cache denial is the regression).

2.3 **Proctoring lost-update (D17)** — `postgres_interview_repo.go:100–136` read-modify-write on JSONB. Fix: atomic `events = events || $1::jsonb` append + recompute summary from returned set (mirror RecordCodingSession :161 pattern). Integration race test: two concurrent RecordProctoringEvent → both events present after commit.

Acceptance: `make test-integration-dev` green; coverage floors hold; smoke passes.

---

## Batch 3 — Chat subsystem integrity

Backend first (protocol additions ripple to FE types):

3.1 **wsWriter deadlock (D18)** — defer order chat_handler.go:572–586: `cancel()` runs after `close()`; sender blocked on `ch <-` holds mutex whose release waits on connCtx fired only post-close. Fix: explicit ordered teardown (cancel connCtx BEFORE close in a single cleanup func; close() uses try-lock-with-timeout; writer death drains channel). Unit test with ws harness FIRST: kill transport mid-stream → handler goroutine exits, sessionRegistry Release called, Redis lock releasable within seconds (assert via fake clock, no real TTL wait).

3.2 **Overlapping turns (D16-related, ledger D19)** — read loop chat_handler.go:702–705 lacks busy guard; handleAnswer overwrites turn state mid-stream. Fix: reject second AnswerMessage while turn active with protocol `error` frame (code `turn_in_progress`); ignore-interrupt semantics documented. Protocol test FIRST: two answers back-to-back → exactly one next_question dispatched, second answer rejected; interrupt-then-answer still works. Update `ChatFrame` union in ws.ts:13–48 same commit family; FE renders rejection as retryable notice.

3.3 **Ticket re-mint for reconnect (G2)** — ticket TTL 10 min (interview_service.go:30) vs 30-min interviews; reconnects reuse expired `?t=` (useChatSession.ts:88–92). Design: persist `invitation_token` in sessionStorage keyed by interview id for session duration; on handshake rejection classified as ticket-expired, FE calls existing mint endpoint with invitation_token, retries connect once. Security note: invitation_token already lives in invite URL; sessionStorage exposure ≈ status quo, expires with tab close. Tests: ws lib unit test for re-mint-once-then-fail logic (mock fetch + fake ws); integration test hitting real mint endpoint twice (idempotent success).

3.4 **Reconnect budget reset (G3)** — reconnectCountRef never reset on successful open (useChatSession.ts:33,84–99). Fix: reset to 0 on `interview.start`/resume accepted. Unit test: six drop-connect cycles spread over session never enter permanent disconnected.

3.5 **URL token strip (G6)** — CandidatePortal.tsx:137–143 leaves `?token=` after verify; refresh replays spent token and handleLogout() wipes valid session (:131–134). Fix: after mutate onSuccess, `setSearchParams({}, {replace:true})`. Component test: verify success → URL clean → remount does not logout.

3.6 **Voice page fixes (DA = keep mounted, so mandatory)** — InterviewVoice.tsx:81–82 sends recruiter auth JWT as `?ticket=` on public route (G4): backend RequireTicket validates ws_ticket type only, so FE must mint a proper ws_ticket via the interview ticket endpoint and pass that; auth JWT never goes into URLs. onclose cleanup parity (:132–152): server-initiated close stops MediaStream tracks + closes RTCPeerConnection exactly like onerror (G5). Typed frame union replaces implicit-any JSON.parse. Component tests: token source assertion (no auth JWT in URL), cleanup assertions on close.

Acceptance: `make check` + integration green; `make load-ws` rerun after 3.1/3.2 (writer-path changes); FE vitest green; manual happy-path chat pass locally.

---

## Batch 4 — Worker reliability

4.1 **Transient-error classification (D20)** — webhook_worker.go:99–102 returns SkipRetry for ALL errors incl. transient DB failures; same pattern context_service.go:248–250 (IndexWorker). Fix: classify — permanent (not-found/wrong-org/invalid payload) → SkipRetry + terminal status + error_message; transient → return err (asynq retries) + warn log with delivery id. Unit-test classifier pure; integration test: forced transient failure → delivery retried (asynq test server or manual redrive), permanent failure → status failed exactly once.

4.2 **Bulk upload reporting (m-5)** — cv_service.go:141–158 four silent `continue`s. Fix: collect per-file outcome {filename, ok, reason}; return counts in DTO; warn-log each skip with reason. Handler test asserts DTO shape (OpenAPI updated if response extended).

4.3 **Extract worker error messages (m-2 partial)** — extract_worker.go:276 persists raw cause.Error(). Map provider/internal errors to safe categories (network/quota/format/unknown) with detail only in logs. Unit test mapping table.

4.4 **Rubric worker coverage (E-row)** — integration spec FIRST: happy path, tenant tx usage, retry-does-not-double-invoke-LLM (status-gated), clobber-safety column-scoped write — then fix whatever fails (expected: D1 verification resolves here).

4.5 **Swallowed transition errors (m-1)** — interview_service.go:398–400 `_ = iv.Complete()` + probe insert skip :391–395. Log with context at minimum; decide: Complete failure during advance should surface via error frame, not vanish. Unit test: Complete-error path emits error frame, question still advances per rails.

Acceptance: integration suite green; asynq retry semantics covered by tests; no `_ =` on meaningful errors remains in workers (grep clean).

---

## Batch 5 — Error taxonomy & hardening sweep

5.1 **db.WrapError (m-3)** — pkg/db/errors.go:38 maps everything unknown → DomainError → renders 400 + leaks internal text. Fix: PG-mapped codes keep sentinel mapping; unknown → internal error type rendered as generic 500 by httpapi.Error. Unit tests: cancellation inside tx → 500 generic; unique violation → mapped 400 sentinel unchanged.

5.2 **WS/sandbox error sanitization (m-2/m-7)** — chat_handler.go:34–41 (errorFrame raw err.Error()), :752 (sidecar errors to candidate), sidecar/runner.go:238 (daemon errors into res.Error), runner.go:216 kill failure unlogged. Fix: sanitize to category codes candidate-side, details to structured logs; log kill failures. Unit tests per surface.

5.3 **Demo-token env gate (m-4)** — postgres_portal_repo.go:84,117 demo-prefix bypass compiled in. Gate behind `INTIVAI_DEMO_MODE=1` (default off; prod compose never sets). Unit test: off → demo tokens rejected like garbage; on → bypass active.

5.4 **Telemetry rate limit (m-6)** — main.go:428 userRateLimit keyFn returns "" sans Actor → limiter skipped. Key by ticket subject/interview id instead; cap generous but finite. Unit test keyFn.

5.5 **Sandbox concurrency bound (m-7 rest)** — semaphore around per-frame `code.run` goroutines (chat_handler.go:302); cap HTTP execute test-cases (chat WS caps at 20; HTTP path uncapped). Config-driven, default modest. Unit test semaphore saturation rejects cleanly.

5.6 **Sentry backend wiring (DC)** — sentry.Init gated on DSN; fiber recovery middleware reporting panics with request id; asynq panic/error capture in worker base; correlation tag = release sha. Smoke: forced panic route behind env in dev only → event captured (manual verify, not CI).

5.7 **Misc minors** — parse_worker.go:65–68 mislabels transient store failure as `failed_ocr`: distinguish `failed_fetch` retryable vs true OCR failure. CORS footgun note in config comments (`ALLOWED_ORIGINS=*` + credentials). Bulk-op DTO logging.

Acceptance: `make check`; error-message leak grep (candidate-visible surfaces contain no provider/docker/internal strings — assert via existing handler tests extended).

---

## Batch 6 — Frontend trust & UX

Majors:

6.1 **Sentry replay PII masking (G7)** — main.tsx:10–19 replayIntegration with no masking, 1.0 error sampling. Fix: `maskAllText:true, maskAllInputs:true, block:['[data-sentry-block]']` baseline; disable replay entirely on candidate portal/chat routes (route-based init or beforeSend drop); document residual risk in compliance doc Target marker. Vitest: config object assertions (cheap guard against regressions).

6.2 **Fabricated strengths removal (G8, B1-class)** — CandidatePortal.tsx:672–700 static strings under "Automated feedback". Fix: render only fields backed by real evaluation payload; empty → neutral "No automated feedback available" (matches RecommendationBadge honesty pattern). Component test: no evaluation → no fabricated copy.

6.3 **Silent mutations (G9/G10)** — Jobs.tsx:106–122 patchStatus/patchPublished missing onError → add ApiError toast; Candidates.tsx:220–257 Promise.allSettled masks total failure, catch dead → report succeeded/failed counts, failure toast when succeeded==0, per-item error surfacing. Component tests: rejected promise → visible error state.

Minors sweep (single commit family, each with tiny test where testable):
- Debounce `sendCodeChange` (Chat.tsx:606–608, CodingSandbox.tsx:48–51) ~300ms trailing; flush on submit.
- Timer auto-submit honest toast (Chat.tsx:155–157 respects send result).
- Toast-spam guard deep-linked candidate (Candidates.tsx:147–163 dedupe ref).
- TestCaseManager stable IDs (TestCaseManager.tsx:16 counter/uuid, not length+1).
- Remove dead `questionIdx` prop or implement reset-between-questions (CodingSandbox.tsx:14,21–28) — implement reset, matches product intent.
- Type-lie casts removed (Candidates.tsx:491,592 `as unknown as`); InterviewVoice JSON.parse typed discriminated union (moot if DA=unmount).
- getBlob parity (api.ts:71–78): timeout, ApiError, 401 handling; clipboard .catch (InterviewResult.tsx:157).
- Interviews count clamp 1..10 client-side (Interviews.tsx:107).
- PublicLayout CTA expiry-aware (PublicLayout.tsx:56 use getSession validity).
- SOC 2 footer claim removed pending certification (PublicLayout.tsx:288) — compliance-truth rule.
- emptyAutoRefetchRef reset on identity change (CandidatePortal.tsx:79,213–221).
- Formatting helpers hoisted to module scope (TimerGate.tsx:95, CandidatePortal.tsx:250) — repo convention.
- SortIcon out of render body (Candidates.tsx:324–327).
- RequestHuman param label fix (Chat.tsx:170 posts invitation_token field with ticket value) — send correct field, backend contract tightened + tested.
- OTP resend cooldown UI (CandidatePortal.tsx:376–383) 60s disabled state.
- Consent auto-tick per DD decision.
- StrictMode-safe autostart guard (Invite.tsx:51–56 busy-ref before effect side effects).
- Scroll-FAB positioning container relative (Chat.tsx:421–428) — visual verify mobile matrix (ties F8 UAT).

Token-stream visibility (minor #1, but UX-significant): expose streamBufferRef incrementally to transcript (throttled render) — include if time allows in 6.x, else defer with date below.

Acceptance: `make fe-build` + vitest green; manual pass of candidate journey on 375px viewport (feeds F8/F17 UAT evidence).

---

## Batch 7 — CI / ops hardening

7.1 Deploy job gates: `needs: [backend, frontend, backend-integration, backend-smoke]` (ci.yml:175).
7.2 Root-gate job: run root `make check` (FE build+vitest+lint-docs) as its own job or fold into frontend job; Makefile:65 comment becomes true.
7.3 Playwright nightly workflow (DE=b): cron schedule, stack up + seeded + LLM secret, `workers:1`, allow-fail initially → promote to required after F18 stall fixed. F18 fix task included here: response-frame-driven wait replacing banner regex (frontend/e2e/happy-path.spec.ts).
7.4 Smoke without LLM key: scripts/smoke.sh:59–61 degrade to skip-with-warning (CI sets SKIP_REASON), never silently green — distinct exit code surfaced in job summary.
7.5 Pin golangci-lint via official action with version input (ci.yml:23).
7.6 gosec enabled in .golangci.yml (start with default severity config; triage findings, fix or nolint-with-comment).
7.7 Digest-pin runtime images (backend/Dockerfile alpine:3.20 → digest, align builder/runtime alpine minor; sandbox images digest-pinned in ci.yml:201–210 pull step); Dependabot or renovate for image bumps.
7.8 Redis prod auth: requirepass from .env.prod `${REDIS_PASSWORD:?}` (docker-compose.prod.yml:125–131) + app DSN/addr updates.
7.9 restore.sh role bootstrap passwords from env `${...:?}` (restore.sh:38,41); runbook note: rotate post-restore mandatory step printed by script.
7.10 GitHub Environment `production` protection: required reviewer on deploy; post-deploy healthcheck step polls `/ready` and auto-runs documented rollback TAG on failure (compose rollback prose docker-compose.prod.yml:11–20 becomes a script).
7.11 Socket proxy (DF): add docker-socket-proxy service; sidecar DOCKER_HOST points at proxy with CONTAINERS=1, IMAGES=1 only; ADR-0002 amendment appended same day.
7.12 Backup × GDPR erase interplay: compliance doc gets explicit policy row — erased candidates persist in 14-day backup window (legal accepted statement, Target marker for crypto-erase future). Links migration 018.
7.13 mTLS rotation procedure one-pager in docs/engineering (CA 10y gen-sandbox-certs.sh:19; leaf 365d regen steps).

Acceptance: CI green end-to-end incl. new nightly trigger dry-run (workflow_dispatch manual run); prod deploy rehearsed on staging-like local compose with env-protection simulation where possible.

---

## Batch 8 — Beta-gate closure (ops-heavy, mostly outside repo)

- #9 VPS/domain/secrets provisioning checklist execution.
- #10 host cron install (scripts/backup.cron) + FIRST restore drill (restore.sh into intivai_restore, promote rehearsal) — record artifact.
- #12 pilot recruiting partner outreach (business).
- #14 tagged release flow: tag → deploy-from-tag verification (ci.yml deploy consumes TAG var already).
- SLO actuals recording (docs/engineering/slos.md:10–17) from k6 + load-ws runs post-Batch-3 (WS changes may shift numbers; measure after).
- Sentry DSN production value + alert routing (finishes DC).

---

## Explicit deferrals (WHERE/WHY/DATE per governance)

| Item | Where deferred | Why | Revisit date |
|---|---|---|---|
| Token streaming visibility in chat transcript | Batch 6 note | polish vs trust-critical work; needs design pass | 2026-09-15 |
| Encryption-at-rest (ledger F12) | stays Target Q4 2026 in PRD/compliance | infra procurement | Q4 planning |
| ATS integrations (F14), reporting depth (F15) | post-beta by design | competitive scope | post-beta |
| Voice real STT/Opus (P5 unchecked criteria) | P5 demo-only per ADR-0006 | depends on DA outcome + product priority | P5 kickoff |
| localStorage JWT → httpOnly cookie (C8) | accepted beta risk, documented | cross-cutting auth refactor | post-beta hardening |

## Finding → ledger row mapping (append at Batch 0.1)

New IDs (next free per section):

| Plan ref | Ledger ID | Section |
|---|---|---|
| Portal auth bypass | C9 | C Security |
| Apply abuse caps | C10 | C Security |
| Scorecard notify dead query | D15 | D Architecture/correctness |
| PDF cache-hit authz | D16 | D |
| Proctoring lost-update | D17 | D |
| wsWriter deadlock | D18 | D |
| Overlapping answer turns | D19 | D |
| Webhook/index transient swallow | D20 | D |
| db.WrapError 400-for-internal | D21 | D |
| Demo-token ungated | D22 | D |
| Telemetry limiter exempt | D23 | D |
| Sandbox concurrency/error-leak | D24 | D |
| Swallowed Complete()/probe errors | D25 | D |
| Extract ErrorMessage leak / failed_ocr mislabel | D26 | D |
| Bulk upload silent skips | D27 | D |
| Rubric worker untested | E3 | E Coverage |
| Drift guard blind spot | E4 | E |
| DELETE 204 crash | G1 | G Frontend (new) |
| Ticket TTL vs interview duration | G2 | G |
| Reconnect budget never resets | G3 | G |
| Voice page auth-JWT-as-ticket | G4 | G |
| Voice onclose mic leak | G5→note | G (merge with G4 if DA=unmount) |
| Magic-link URL replay | G6 | G |
| Silent job mutations | G7 | G |
| Bulk ops masked failures | G8 | G |
| Sentry replay PII | G9 | G |
| Fabricated strengths | G10 | G |
| FE minors bundle | G11 | G |
| Voice route mounted vs 4 docs | H1 | H Docs truth (new) — resolved by doc rewrite + ADR-0008 per DA, not unmount |
| Sentry claimed wired | H2 | H |
| P5 checked-but-mocked criteria | H3 | H |
| Stale ranges/anchors (001–009, 001–007, schemas anchors) | H4 | H |
| Duplicate YAML keys | H5 | H |
| Executed plan unarchived | H6 | H |
| PDF integrity default 100 | B10 | B Data honesty |
| Deploy misses frontend gate | A9 | A Release/deploy |
| Root check/docs-lint absent in CI | A10 | A |
| restore.sh known-password roles | A11 | A |
| Socket uncontained (DF) | A12 | A |
| No deploy approval/auto-rollback | A13 | A |
| Image pinning/alpine mismatch | A14 | A |
| Redis prod unauth | A15 | A |
| Smoke hard-fails sans key | A16 | A |
| golangci unpinned install | A17 | A |
| gosec absent | A18 | A |
| E2E not in CI (+F18 stall) | A19 | A |

## Suggested execution order

Batch 0 → 1 → 2 → 3 (backend half) → 4 → 5 → 6 → 7 → 8, with 3.3–3.5 (FE reconnect) able to parallel-track 4 behind 3.1/3.2 landing. Batches 0–1 are days, not weeks; 2–3 the technical core; 7 mostly config. Each batch ends with gates + ledger status flips with evidence links.
