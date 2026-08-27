# Intivai Consolidated Remediation Plan

- **Date:** 2026-08-26
- **Status:** superseded by execution outcome — archived 2026-08-27
- **Owner:** EM
- **Execution outcome (2026-08-27):** all J1-J19 findings implemented with
  tests + ledger evidence; legacy A-D/F/G rows triaged — 129 closed, 8
  in-progress, 5 deferred (each with owner + revisit date in
  `docs/FINDINGS.md`). Gates: `make check` green, `make coverage` green
  (CD11 exemption register in `scripts/check-coverage.sh`, review date
  2026-08-28), integration + `-race` green, tracing gate + OpenAPI drift
  green. Residual tracked work lives in `docs/FINDINGS.md` (in-progress rows)
  and `docs/engineering/beta-gate.md` (ops/UAT closure) — NOT in this plan.
- **Scope:** all remaining work from R24, R25, R26, and R27; feature completion;
  OTel hardening; non-code cleanup
- **Canonical status source:** `docs/FINDINGS.md`
- **Superseded plans:**
  - `docs/plans/archive/review-fix-plan-2026-08-24.md`
  - `docs/plans/archive/feature-plan-2026-08-24-weights-questions-qa.md`
  - `docs/plans/archive/otel-tracing-plan-2026-08-26.md`

Historical execution plan. All decisions and acceptance criteria remain
retained here for context; status lives in `docs/FINDINGS.md`. Residual dated
follow-ups (A4/A5/A13, D24, B8, F17, G11, I17-watch) are ledger rows with
revisit dates, not open plan batches.

---

## 1. Consolidation Rules

1. Work is organized by risk and dependency, not by the historical plan that
   first described it.
2. One concern per commit. Migration, repository, and domain changes land in
   one migration-bearing commit.
3. TDD is mandatory: regression test first, expected red failure, implementation,
   targeted test, then batch gate.
4. Every handler, worker, repository, and external-provider path must inherit
   tracing through an approved seam or create a documented child span.
5. Security and data-integrity findings block release even when unit tests pass.
6. A plan batch is complete only when its code, tests, docs, ledger evidence, and
   operational verification all exist.
7. Deferrals require location, reason, owner, and revisit date in both this plan
   and the ledger.
8. No commit is created automatically. Changes remain reviewable in the working
   tree until explicitly authorized.

## 2. Current Baseline

### 2.1 Already delivered

- R24 truth-sync and docs-reorganization work is complete where ledger rows are
  closed.
- R25 B1-B4 backend work is implemented: strict weights, publish lock,
  generated question sets, job candidate context, and candidate Q&A protocol.
- R25 B5 is incomplete: recruiter-facing Q&A, HR-configurable Q&A limit, and
  candidate-context UI remain absent or incomplete.
- OTel batches A-E exist locally: HTTP, GORM, asynq, LLM, and interview spans.
- Jaeger v2 dev service works and live traces have been observed.
- Candidate-review migration 031 fixes the valid-token 500.

### 2.2 Gates currently known

- `make check`: green at last validation.
- `make test-integration-dev`: green at last validation.
- `go test -race ./internal/interview/...`: green at last validation.
- `make smoke`: green after smoke publication-state correction.
- `make coverage`: red with 15 packages below floor; baseline comparison was
  16 packages, so batch improved baseline but did not satisfy DoD.
- Production tracing is not enabled yet and OTel security hardening is required.

### 2.3 Status source

Existing open and needs-verification rows remain in `docs/FINDINGS.md`, including
A2-A5, A8-A19, B1-B8, C4-C10, D1-D14, D17-D27, E2-E3, F7-F8, F12, F14-F18,
G2-G11, and I17-I18. New review findings are recorded as J1-J19 in the
consolidated ledger update.

## 2.4 Legacy finding disposition map

Every existing non-closed finding has an explicit destination below. Status still
lives in `docs/FINDINGS.md`; this table prevents a finding from disappearing when
historical plans are archived.

| Ledger IDs | Destination | Completion evidence |
|---|---|---|
| A2-A5, A8 | Batch 6 runtime/CI/recovery | CI, sandbox, backup, restore, and publicURL verification artifacts |
| A9-A10, A17-A19 | Batch 6 CI/runtime; A19 also Batch 5/6 E2E | Required CI workflow gates and Playwright/smoke artifacts |
| A11-A16 | Batch 1 security or Batch 6 runtime/CI | Secret, Redis, Docker, smoke, and deploy verification |
| B1-B8 | Batch 1 fairness/truth, Batch 3 feature completion, Batch 5 frontend | Domain tests, recruiter/candidate UAT, and ledger evidence |
| C4-C10 | Batch 1 security/privacy | Auth, portal, rate-limit, storage-path, and candidate-journey tests |
| D1-D6, D10-D14 | Batch 2 transaction/worker integrity; D12 also Batch 6 error hardening | Integration tests, retry tests, migration/repository evidence |
| D7-D9, D17-D19, D24-D25 | Batch 2 concurrency/state integrity | Race tests, transaction tests, WS protocol tests |
| D20-D23, D26-D27 | Batch 2 worker integrity and Batch 6 hardening | Retry classification, safe errors, bulk outcome, and security tests |
| E2-E3 | Batch 2/7 test and coverage closure | Dedicated worker tests and coverage evidence |
| F7-F8, F12, F14-F15 | Batch 5 beta/UAT or explicit post-beta disposition | UAT artifacts or dated approved deferral |
| F17-F18 | Batch 5 frontend/UAT and Batch 7 beta closure | Mobile/a11y and response-frame-driven Playwright evidence |
| G2-G11 | Batch 5 frontend trust and candidate experience | FE tests plus 375px candidate journey evidence |
| I17 | Batch 2 extraction side-effect refactor | Idempotent after-commit enqueue and re-extract integration tests; scheduled 2026-08-27 |
| I18 | Batch 7 coverage closure | Package tests or explicit dated exemption register |

Rows marked `needs-verify` first receive evidence review. If evidence proves the
finding obsolete, the ledger records `obsolete` with the proof; otherwise it
becomes an implementation task in its mapped batch.

---

## 3. Decision Register

| ID | Decision | Resolution |
|---|---|---|
| CD1 | Canonical plan | This file is the only active plan; historical plans move to archive with status headers. |
| CD2 | Finding status ownership | `docs/FINDINGS.md` owns status; this plan owns sequencing, scope, and acceptance evidence. |
| CD3 | Non-code cleanup | Remove local-only credential/config clutter; retain project-required `.kilo/`, `.kilocode/`, CI, migrations, OpenAPI, Compose, scripts, and active engineering docs. |
| CD4 | Credential handling | `kilo.jsonc` must contain no token. Revoke exposed token before any commit. Use environment/local secret injection. |
| CD5 | OTel privacy | HTTP spans must not retain query, full URL, authorization, JWT, review token, or candidate-sensitive values. Redaction tests are release-blocking. |
| CD6 | OTel failure mode | Collector backpressure must drop bounded telemetry rather than block request, worker, or WS business paths. |
| CD7 | Async consistency | Worker enqueues that depend on committed state use after-commit dispatch or an outbox. Versioned tasks use conditional writes. |
| CD8 | Candidate Q&A visibility | Recruiters receive persisted Q&A through an authorized API/DTO/UI surface before B5 closes. |
| CD9 | Grounding failure mode | Candidate-question answering fails closed on unavailable grounding context; no unsupported LLM answer is sent. |
| CD10 | Production sandbox | Raw Docker socket must not be available to production sandboxd; socket proxy or equivalent containment is required. |
| CD11 | Coverage | Existing low-coverage baseline is not silently exempted. Either add tests or record explicit package exemptions with owner/date. |

---

## 4. Batch 0 — Consolidation and Safe Cleanup

### 4.1 Plan/archive normalization

- Add `Status: superseded by ...` headers to the three historical active plans.
- Move them with `git mv` into `docs/plans/archive/`.
- Update `docs/README.md` so it lists exactly one active plan and all relevant
  archived plans.
- Refresh index `Last-reviewed` date.
- Run docs link and markdown gates.

### 4.2 Non-code file disposition

| Path/category | Disposition | Reason / acceptance |
|---|---|---|
| `kilo.jsonc` | Remove from repository scope; revoke token | Contains credential-like `userToken`; no secret may exist in tracked or proposed commit files. |
| `.agents/rules/` | Remove unless explicitly adopted by project workflow | Duplicate editor-agent guidance; no runtime/build dependency. |
| `.clinerules/` | Remove unless explicitly adopted | Duplicate caveman/tool instructions; no product/build dependency. |
| `.cursor/rules/` | Remove unless explicitly adopted | Local editor instructions, not application source. |
| `.windsurf/rules/` | Remove unless explicitly adopted | Local editor instructions, not application source. |
| `.github/copilot-instructions.md` | Keep only if GitHub Copilot is an approved project tool | Otherwise remove as duplicate instruction source. |
| `.github/hooks/rtk-rewrite.json` | Keep only if CI/developer workflow consumes it | Otherwise remove; prove no references first. |
| `.kilo/command/` | Keep | Project Kilo commands are part of repository configuration. |
| `.kilocode/rules/` | Keep if loaded by repository configuration | Current RTK rule is active project tooling. |
| `docs/plans/active/*` superseded plans | Archive | Historical decisions retained, no duplicate active execution plans. |
| `backend/pkg/db/migrations/*` | Keep | Reversible production schema artifacts. |
| `api/openapi.yaml` | Keep | API contract and drift-gate input. |
| `docker-compose*.yml` | Keep | Runtime/development deployment configuration. |
| `scripts/*.sh` | Keep when referenced by Makefile/CI; remove dead helpers | Check references before deletion. |
| `make_commits.sh` | Keep deleted | No references found; helper encouraged unsafe bulk commits and is not project workflow. |

Verification:

- `git grep` confirms no source or CI reference to removed local-only files.
- Secret scan over tracked and staged candidate files returns no token.
- `make check` and docs gates pass after disposition.

### 4.3 Ledger and plan hygiene

- Append J1-J19 to `docs/FINDINGS.md`.
- Add this plan as the R27/R28 execution home in `docs/README.md`.
- Do not duplicate full finding descriptions in old active plans after archive.

Gate: only one file remains in `docs/plans/active/`; docs links pass; no credential
file remains in proposed scope.

---

## 5. Batch 1 — Critical Security and Privacy

### 5.1 Portal authentication and public apply

Close existing C9/C10 and related A/B rows:

- Remove portal tokens from public apply responses.
- Require mailbox proof before issuing candidate portal authentication.
- Remove `invitation_token` from cross-org application lookup results.
- Add per-email daily and per-IP abuse limits.
- Preserve generic response semantics to avoid account enumeration.
- Integration-test apply, verify, cross-org access, replay, and rate-limit paths.

Acceptance:

- No credential appears in public apply response, logs, traces, or DTOs.
- Cross-org candidate data and interview invitation tokens are inaccessible.
- Smoke includes the protected apply flow.

### 5.2 Credential and telemetry privacy

Fix J1, J2, J5, and J9:

- Revoke exposed `kilo.jsonc` token and remove file from repository scope.
- Configure `otelfiber` to omit/sanitize `url.query`, `url.full`, and sensitive
  path values. Never export `ticket`, review token, JWT, authorization header,
  candidate name/email, CV text, prompt, completion, or provider response body.
- Validate WS action values before adding any action attribute.
- Replace raw provider error text in `RecordError`/status with safe error class,
  HTTP status, retryability, and attempt number.
- Add exporter assertions containing synthetic secrets; tests must prove absence.

Acceptance:

- Synthetic `?ticket=secret` does not appear in Jaeger export.
- Synthetic provider response containing candidate text does not appear in span
  events or status descriptions.
- Security review confirms candidate data remains outside traces and logs.

### 5.3 Existing security backlog

Schedule C5, C6, C8, C9, C10, G4, G6, G9, F12, A11, A12, A15, and relevant
needs-verification rows. No beta release while C9 remains open.

Gate: security integration tests, secret scan, `make smoke`, and ledger evidence.

---

## 6. Batch 2 — Transaction and Concurrency Integrity

### 6.1 Public candidate Q&A atomicity

Fix J6:

- Make the atomic append predicate require active status and unexpired interview.
- Preserve cap enforcement in the same statement.
- Return distinct outcomes for inactive, expired, cap-exhausted, and missing rows.
- Add concurrent completion/expiry/append integration tests.

### 6.2 Job publication and worker consistency

Fix J7, J8, and existing I8/I17/D1/D20/D25:

- Move publish-triggered enqueues after transaction commit or implement an
  outbox with retry/visibility guarantees.
- Include job version in question/rubric task payloads.
- Lock or conditionally update job rows so stale workers cannot overwrite newer
  question sets or write after unpublish.
- Define republish behavior: invalidate old set, regenerate, and preserve the
  candidate consistency rule for interviews already created.
- Move extraction side-effect enqueues out of worker execution before fixing
  I17 task IDs; preserve email/memory idempotency.
- Classify transient worker failures as retryable and permanent failures as
  `SkipRetry` with terminal safe error state.

### 6.3 State and data races

Fix D7, D8, D10, D17, D18, D19, D24, D25, J14, and J16:

- Make probe insertion and interview cursor advancement atomic or recoverable.
- Prevent context-upload rows from committing without durable object storage or
  compensate by deleting orphan rows/objects.
- Replace proctoring JSONB read-modify-write with atomic append and recomputed
  summary.
- Preserve WS writer teardown ordering and test transport death.
- Reject overlapping answer turns and preserve single next-question dispatch.
- Bound sandbox execution concurrency and sanitize daemon errors.
- Make telemetry shutdown `sync.Once`-safe and retry semantics explicit.
- Correct the GORM trace test to execute through `TxFrom(ctx)` and assert the
  DB span is a child of `tenant.tx`.

Acceptance:

- `go test -race` passes for interview, queue, DB, worker, and affected packages.
- Integration tests prove no lost updates, stale writes, post-expiry writes, or
  pre-commit worker reads.
- No meaningful `_ =` error swallowing remains in affected workers/services.

Gate: targeted integration suite, race suite, full `make check`.

---

## 7. Batch 3 — Feature Completion: Weights, Questions, Candidate Q&A

### 7.1 B1/B2 backend completion

- Close remaining weight validation and publish-lock verification.
- Add dedicated rubric-worker tests: tenant tx, retries, idempotency, and
  column-scoped writes.
- Complete question-set stale-version and republish semantics from Batch 2.
- Expose safe `question_set_error` state to recruiter UI with retry action.
- Validate all generated framing fields for bias and length before persistence.

### 7.2 B3 candidate context

Fix J12 and existing context findings:

- Return the incremented version and updated timestamp from PUT using `RETURNING`
  or a read-after-write.
- Enforce injection rails before persistence.
- Distinguish job-not-found, unauthorized, malformed, and storage failures.
- Add recruiter-facing context editor and AI-suggest draft flow; suggestions
  are never auto-persisted.

### 7.3 B4/B5 candidate Q&A

Fix J10, J11, J13 and complete B5:

- Fail closed when grounding context cannot be loaded; no LLM call on missing
  or unavailable required grounding material.
- Add recruiter-authorized Q&A retrieval to `InterviewDetail` or a dedicated
  endpoint. Include OpenAPI, frontend types, UI, and authorization tests.
- Add org-admin configuration endpoint/UI for `candidate_qa_limit`, with bounds
  validation and tenant authorization.
- Add candidate chat affordance, cap indicator, refusal state, and distinct
  candidate-question/answer transcript rendering.
- Preserve safety rails last for interviewer prompt composition.

Acceptance:

- Recruiter can create/edit candidate context, request a draft, and inspect all
  persisted candidate Q&A after interview.
- Candidate cannot advance normal interview state with a candidate-question
  frame.
- Missing grounding produces refusal without provider invocation.
- OpenAPI and DTO contract tests pass.

Gate: B3/B4 integration tests, FE unit tests, WS `-race`, smoke extension,
fresh-DB boot.

---

## 8. Batch 4 — OTel Tracing Production Hardening

### 8.1 HTTP and WebSocket context

Fix J4:

- Preserve the Fiber `UserContext()` through WebSocket upgrade and initialize
  chat connection context from the handshake span.
- Add an integration test proving handshake, `interview.connect.compose`, DB,
  LLM, and stream spans share the expected trace.
- Keep route-template names and omit raw URL/query attributes.

### 8.2 Fail-soft exporter

Fix J3:

- Remove `sdktrace.WithBlocking()` or replace with a bounded non-blocking policy.
- Bound exporter queue, batch size, retry duration, and shutdown timeout.
- Add collector-down and queue-full tests proving API, worker, and WS progress
  is unaffected.
- Emit one rate-limited internal warning and expose dropped-span metrics.

### 8.3 Production deployment

Fix J17 and complete OTel DoD:

- Decide production enablement explicitly; if enabled, set sample ratio `0.1`
  rather than inheriting development `1.0`.
- Keep Jaeger internal-only in production, add restart/resource policy, and
  document in the runbook.
- Keep `OTEL_ENABLE=false` as emergency kill switch.
- Add Sentry trace correlation or update docs to state it is deferred.
- Add sandboxd gRPC propagation and embedding spans only under a dated Batch F
  scope; do not claim complete tracing before implementation.

### 8.4 Tracing gate redesign

Fix J15:

- Replace grep-only enforcement with AST/static checks or compile-time test
  fixtures.
- Scan `cmd`, `internal`, and `pkg` while excluding only generated files and
  tests where appropriate.
- Validate middleware order, not merely string presence.
- Detect grouped/multiline package-level tracer declarations.
- Block direct asynq/OpenAI bypasses while allowing generated or test fixtures.
- Add negative tests for every rule and a bypass matrix.

Acceptance:

- No sensitive URL values in live Jaeger traces.
- Collector outage does not block business paths.
- HTTP-to-WS and HTTP-to-worker trace continuity is proven.
- Gate catches all fixture bypasses and produces no unacceptable false positives.

Gate: OTel unit/integration tests, Jaeger live smoke, `-race`, tracing gate.

---

## 9. Batch 5 — Frontend Trust and Candidate Experience

Close G2-G11, B7-B8, F7-F8, and remaining frontend needs-verification rows:

- Protect/re-mint expired WS tickets once; reset reconnect budget after success.
- Strip single-use tokens from URLs after verification.
- Fix voice-page ticket type, media cleanup, and typed frames.
- Remove fabricated candidate feedback and false proctoring/integrity claims.
- Mask/disable Sentry replay on candidate surfaces.
- Report partial bulk-operation failures.
- Debounce code changes, fix timer-submit result handling, stable testcase IDs,
  clipboard/getBlob errors, input clamps, stale-auth CTA, OTP cooldown, and
  StrictMode guards.
- Complete mobile 375px journey and response-frame-driven Playwright waits.

Acceptance:

- `make fe-build`, Vitest, and Playwright targeted scenarios pass.
- Candidate UI never presents unsupported hiring/evaluation claims.
- No candidate credential remains in browser URLs after consumption.

---

## 10. Batch 6 — CI, Runtime, and Recovery Hardening

Close A2-A5, A8-A19, D20-D27, and related operational findings:

- Gate deploy on backend, frontend, integration, smoke, and root checks.
- Keep golangci-lint version pinned and add gosec after triage.
- Pin runtime and sandbox images by digest; align Alpine versions.
- Add Redis production authentication and update all app DSNs.
- Move restore role passwords to required environment variables; print mandatory
  post-restore rotation instruction.
- Add production environment approval and post-deploy health/rollback automation.
- Contain sandbox Docker access through socket proxy with minimal permissions;
  rendered production Compose must contain no raw socket mount.
- Add sandbox image/cert/exec-image deployment verification.
- Make smoke behavior explicit when LLM key is absent; no silent green or
  permanently red policy.
- Add backup schedule, offsite policy, restore drill artifact, mTLS rotation
  runbook, and GDPR backup/erase policy.

Gate: CI workflow dry run, staging-like production Compose render, smoke,
restore drill, sandbox execution, and security lint.

---

## 11. Batch 7 — Coverage and Beta Closure

### 11.1 Coverage

Resolve I18 and J18:

- Remove `queue_debug_test.go` or turn it into assertions.
- Add focused tests for new telemetry, sandboxd bootstrap, LLM, memory, sandbox,
  context, and screening persistence paths where they are production-critical.
- For intentionally low-value adapters, add explicit package exemptions with
  rationale, owner, and review date rather than weakening the global floor.

### 11.2 Beta gate

- Reconcile every beta-gate checkbox with actual implementation.
- Execute UAT matrices for mobile candidate flow, voice/demo, scorecard,
  recruiter Q&A, and error recovery.
- Record actual SLO/load results after WS changes.
- Close only evidence-backed findings; leave accepted post-beta risks explicitly
  dated in the ledger.

Gate:

```text
make check && make coverage && make test-integration-dev
make smoke
make load-ws
go test -race ./...
```

Plus fresh-DB boot, production Compose render, security scan, and documented
manual artifacts.

---

## 12. New R27 Findings Included

| ID | Finding | Planned batch |
|---|---|---|
| J1 | Untracked `kilo.jsonc` contains credential-like `userToken` | 0, 1 |
| J2 | otelfiber exports query/full URL values containing JWT/review tokens | 1, 4 |
| J3 | OTel `WithBlocking` can propagate collector backpressure into business paths | 4 |
| J4 | WebSocket upgrade discards Fiber handshake trace context | 4 |
| J5 | Unvalidated WS action is copied into telemetry attributes | 1, 4 |
| J6 | Candidate Q&A append lacks atomic active/expiry predicate | 2 |
| J7 | Question worker permits stale/duplicate generation and overwrites | 2, 3 |
| J8 | Job update enqueues work before transaction commit | 2 |
| J9 | Raw LLM provider response can enter span error data | 1, 4 |
| J10 | Persisted candidate Q&A is not exposed through recruiter API/UI | 3 |
| J11 | Candidate-question grounding fails open on context errors | 3 |
| J12 | Candidate-context PUT returns stale version after upsert | 3 |
| J13 | Org Q&A limit has no HR configuration API/UI | 3 |
| J14 | Telemetry shutdown wrapper is not concurrency-safe | 2, 4 |
| J15 | Tracing gate relies on bypassable grep checks and misses ordering | 4 |
| J16 | GORM trace test bypasses `TxFrom(ctx)` and cannot prove tenant nesting | 2, 4 |
| J17 | Production OTel is disabled; default enabled ratio is development 100% | 4, 6 |
| J18 | Debug queue test logs only and has no assertions | 7 |
| J19 | Active plan/index status is stale and duplicates execution sources | 0 |

Existing overlapping findings remain linked rather than duplicated: A12 covers
raw production Docker socket exposure; C9 covers portal auth bypass; D18/D19
cover WS teardown/overlapping turns; I17/I18 remain open until their stated
dates and evidence are resolved.

---

## 13. Ownership and File Safety

| Area | Primary ownership | Protected files |
|---|---|---|
| Plan/docs/ledger | docs owner | this plan, `docs/FINDINGS.md`, `docs/README.md`, architecture/runbooks |
| Portal/security | backend security owner | portal repo, public handler, auth/rate-limit tests |
| DB/concurrency | backend persistence owner | migrations, repositories, transaction tests |
| Feature B5 | interview/job + FE owners | Q&A API, context UI, interview result UI, OpenAPI |
| OTel | platform/backend owner | telemetry, HTTP middleware, queue, DB, LLM, Compose |
| Runtime/CI | ops owner | Compose, Dockerfile, workflows, restore/backup/smoke scripts |
| Non-code cleanup | repo owner | local tool configs, plan paths, docs index |

No two agents edit the same file in parallel. Shared test harness changes land
before the feature batch that consumes them.

## 14. Archive Criteria

This consolidated plan moves to `docs/plans/archive/` only when:

- Every in-scope ledger row is `closed`, or explicitly `obsolete`/`deferred` with
  owner and revisit date.
- `make check`, `make coverage`, integration, race, smoke, and required UAT gates
  are green or have documented approved exemptions.
- Production Compose/security render checks pass.
- OpenAPI, schemas, architecture, runbooks, and docs index match reality.
- Evidence artifacts are linked from the ledger and this plan.

Until then, this file remains the sole active plan.
