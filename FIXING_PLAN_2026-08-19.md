# Intivai — Deep-Review Fixing Plan (2026-08-19)

Comprehensive remediation plan covering all findings from:

1. **Four-lens deep review** (Tech Lead · Product Owner · CEO · HR) — `REVIEW_2026-08-19.md`
2. **Code quality / readability / maintainability review** (3 craft agents: backend, frontend, infra)
3. **Coverage/test-honesty audit** (coverage gate, untested security surfaces)

Severity: 🔴 blocking · 🟡 important · 🟢 nit. Effort: S ≤0.5d · M ≤1d · L 2d+.

---

## Phase 0 — Release discipline (day 0, ~1h) 🔴

| # | Task | Files | Verify |
|---|---|---|---|
| 0.1 | Commit the ENTIRE working tree in logical commits (migrations 011–017, rubric worker, sidecar stack, voice demo, FE lifecycle, Makefiles, Caddyfile). Nothing else matters until a tag ships the reviewed code | whole tree | tag deploys migrations 001–017; `git status` clean |

---

## Phase 1 — Production blockers (days 1–3) 🔴

| # | Task | Files | Effort |
|---|---|---|---|
| 1.1 | **Sandbox on prod**: build `sandboxd` into the pushed image (multi-target or dedicated image); CI builds + pushes the 4 `intivai-sandbox-*` execution images; deploy provisions mTLS certs (`gen-sandbox-certs.sh`); add code.run round-trip to prod smoke | backend/Dockerfile, ci.yml, docker-compose.prod.yml, scripts/gen-sandbox-certs.sh | M |
| 1.2 | **`is_published` gate**: commit the migration-017 filter; FE publish/unpublish toggle on the job card; careers board excludes unpublished | migrations/017, job domain+handler+repo, Jobs.tsx, Careers.tsx | S |
| 1.3 | **Real SMTP**: prod overlay overrides `INTIVAI_SMTP_*`; remove mailpit from prod; wire `candidate_review` email type (currently dropped by unknown-default + hardcoded `localhost:5173` → `APP_PUBLIC_URL`); enqueue scorecard email; add offer/reject decision emails | docker-compose.prod.yml, email_worker.go, mailer.go, extract_worker.go, candidate_portal_handler.go | M |
| 1.4 | **Rubric worker**: `OrgID` in payload + `db.RunInTx`; column-scoped rubric update (no full-row clobber); LLM-retry idempotency guard; explicit `MaxRetry(5)` | rubric_worker.go, job repo, job_service.go | M |
| 1.5 | **CI honesty**: smoke `BASE=http://localhost:8080`; DeepSeek key as CI secret; `backend-smoke` in deploy `needs:`; `set -euo pipefail`; `permissions:` block; stop sed-rewriting tracked compose (interpolate `${IMAGE}`) | ci.yml, docker-compose.prod.yml | S |
| 1.6 | **JWT**: `WithExpirationRequired()` + `WithIssuer("intivai")` in Parse | jwt_provider.go | S |
| 1.7 | **Rate limits**: fail-closed on auth buckets; register `tenantRateLimit` AFTER `authMW` (dead today — keyFn always empty); per-IP limit on public jobs list + public apply | main.go, ratelimit.go | S |

---

## Phase 2 — Data honesty (days 3–4) 🔴

| # | Task | Files | Effort |
|---|---|---|---|
| 2.1 | Delete every fabricated value → honest "no data / not evaluated / telemetry unavailable" states: `interview_score ?? 85`; integrity "100/100 · Spotless" (UI **and** PDF); canned "AI Screening Recommendation"; hardcoded AI review QS=85; `?? 3 yrs` + fake skills; `quality_score \|\| 85` | Candidates.tsx, Candidate360Drawer.tsx, InterviewResult.tsx, pdf.go, sandbox_service.go, evaluation report | M |
| 2.2 | Backend enriches applications list with `interview_score`/interview fields (kills fake pill + dead filters) | screening DTO + repo | M |
| 2.3 | Careers: real error state instead of silent `DEMO_JOBS` fallback; delete dead `DEMO_JOBS` | Careers.tsx | S |
| 2.4 | Proctoring card renders "no telemetry recorded" when summary absent; FE empty `catch {}` blocks log with context | InterviewResult.tsx, InterviewVoice.tsx, CodingSandbox.tsx | S |

---

## Phase 3 — Layering + test honesty (days 4–6) 🟡

| # | Task | Files | Effort |
|---|---|---|---|
| 3.1 | **OTP repo** (`Create`/`FindValidByToken`/`FindValidByCodeHash`/`IncrementAttempts`/`Consume`/`PurgeExpired`); portal handler thin; kill duplicated lookups + `?`/`$1` drift | new postgres_otp_repo.go, candidate_portal_handler.go | M |
| 3.2 | **Apply SQL → repo method** (`ApplyWithDedupe`: advisory lock + `ON CONFLICT … RETURNING`); handler keeps HTTP only | public_handler.go, job repo | S |
| 3.3 | **Coverage-gate parser fix** (bare `coverage: 0.0%` lines count as 0) — six invisible packages become visible | scripts/check-coverage.sh | S |
| 3.4 | **Tests for the two most serious gaps**: `UpdateDecision` transition table (service-level); OTP lockout (429-after-5, replay, expiry, cross-token-type 401); email/rubric/sync worker SkipRetry semantics | screening, portal, worker tests | M |
| 3.5 | Public-apply cleanup: never delete a re-applying candidate's resume; delete the application row on enqueue failure (no stuck `parsing` candidates) | public_handler.go | S |
| 3.6 | Memory layer: reuse `RunInTx`/`TxFrom` (no double-begin deadlock pattern); cache native SQLite handle + `busy_timeout` | postgres_memory.go, native_memory.go | S |
| 3.7 | Context upload outside the `FOR UPDATE` tx (DB work in tx, upload after commit) | context_service.go | S |
| 3.8 | Extract `CreateApplicationWithRecovery` (savepoint + 23505) shared by extract_worker + screening_service | extract_worker.go, screening_service.go | S |

---

## Phase 4 — Craft debt (days 6–9) 🟡

| # | Task | Files | Effort |
|---|---|---|---|
| 4.1 | `lib/stages.ts` shared stage→label/color map (kills the drifting duplicate) | new lib/stages.ts, Candidates.tsx, Candidate360Drawer.tsx | S |
| 4.2 | `useChatSession` hook + frame reducer out of the 110-line effect; stable bubble ids (`key={i}` on an edit stream) | Chat.tsx, ws.ts | M |
| 4.3 | `scanPublicJob` + `unmarshalJSONB`; `interviewColumns` const; kill 3× duplicate SELECT lists + marshal blocks | postgres_job_repo.go, postgres_interview_repo.go | M |
| 4.4 | `InterviewDeps`/worker-deps options structs (kill 12-param constructors); split `main()` → `buildWorkers()`/`registerRoutes()` | interview_service.go, main.go | M |
| 4.5 | `lib/sandbox.ts`, `lib/clipboard.ts`, `lib/invites.ts` (3 duplicated clusters); one `RecommendationBadge` | Chat.tsx, InterviewVoice.tsx, Interviews.tsx, InterviewResult.tsx, Candidate360Drawer.tsx | S |
| 4.6 | Un-swallow the 15 business-impacting `_ =` errors (log with context): telemetry, coding session, invitation enqueue, PDF cache upload, worker mark/fail, `SessionRemaining`/`TouchInterview` | interview_service, cv_service, evaluation service, context_service | S |
| 4.7 | Naming sweep: `q()`/`tq()` → `tx()`; one import-alias scheme; `errors.Is` everywhere; `map[string]interface{}` → typed structs (cv_handler, rubric_worker); `score_breakdown?: any` → typed | repos, services, cv_handler, api.ts | S |
| 4.8 | Landing/InterviewResult section splits; `ProctoringCard` extraction (kill the JSX IIFE); TimerGate named consts + one ticker; `cn()` uniformity; `useFilterList` + `useDeferredValue` | Landing.tsx, InterviewResult.tsx, TimerGate.tsx, 6 pages | M |
| 4.9 | Infra dedup: `TEST_ENV` + `COMPOSE` vars in backend/Makefile; CI calls `make check` + env hoist; `mktemp`+trap in check-coverage + fix non-matching exemption patterns; remove dead `COMPOSE` in backup.sh; resolve container names via `compose ps -q`; seed.sh fallback surfaces the first error; whisper `ARG VERSION`; align base-image pins | Makefiles, ci.yml, scripts, Dockerfiles | S |
| 4.10 | Dead code removal: backend `IsIdle`, `ResumeIdx`, `NewInterviewStart`, dup `queue.TaskSyncMnemosyne`, `SyncCandidate`/`SyncContext`, `TenantMiddleware`; FE `sendCodeRun`, `trackAudioAnomaly`, 4 dead api.ts types, `DEMO_JOBS` | backend + frontend | S |
| 4.11 | **Backend `Chat()` split** (222-line WS loop → `handleAnswer`/`handleInterrupt`/`handleTelemetry` + `wsWriter` type) — schedule with 4.2 | chat_handler.go | L |
| 4.12 | Canonical-type pass: `ResumeData` ×2, `AICodeReview` ×2, `ExecutionResult` ≈ `CodeResultMessage` → one type per concept, convert at boundaries | cv, screening, sandbox, interview domains | M |
| 4.13 | `config.Load()` → `getString(key, def)` helpers; single shared origin constant (kills `:3000` vs `:5173` split); budget literals 500/1500/800 → named consts | pkg/config/config.go, llm/client.go | S |
| 4.14 | Validation-flow dedup: one `validateInvitation` helper (3 flows); `mustUUID` ×2; task-literal consts (`"score_cv"`/`"send_email"`) + inline payload structs | interview_service.go, cv_service.go, extract_worker.go, job_service.go | S |
| 4.15 | N+1 batch lookup in evaluation + screening List endpoints (join or batched query, per `candidate_applications_lookup` pattern) | evaluation service, screening service | M |
| 4.16 | Comment hygiene sweep: chat_handler:31 misplaced readDeadline comment, `RunInTx` duplicated doc block, context_service stale "Removed lockOrg…" logs, stage.go misleading rule comment | 4 files | S |
| 4.17 | FE convention fixes: CandidatePortal → `useQuery`/`useMutation` + theme tokens + extracted submit button; PublicLayout `NAV` array (2× list); AppShell JWT decode via lib; Careers `formatSalary` once; CodeEditor identity ternary; FE naming sweep (`b`/`i`/`qc`); derived-state-through-effects → direct params; Prettier pass (Jobs.tsx indent); Dashboard badges driven by real health query | 8 files | M |
| 4.18 | Migrations craft: "what changed" version headers for 016/017 function rewrites; 002.down role-drop guard note; single id-convention (comment on 001); 009+ filename headers — drop or backfill | migrations | S |
| 4.19 | Mailer craft: parse templates once at package scope; drop or use dead `ctx`; "10 minutes" magic text; `"active"` sentinel in session_registry → constants | mailer.go, session_registry.go | S |

---

## Phase 5 — HR fairness + ops hardening (days 9–12) 🟡

| # | Task | Files | Effort |
|---|---|---|---|
| 5.1 | Scoring fairness: neutral degree default (0.5, kills degree-ism); neutral-fill empty dimensions (never 0 for unasked); recommendation enum validation + consistency sampling; explicit 30-QA window policy | scoring.go, report.go, evaluator.go | M |
| 5.2 | Consent discloses full monitoring (tab/paste/audio); proctoring labeled "unverified · flagged events" with human-review step; monitoring-free alternative offered | Invite.tsx, consent flow, InterviewResult proctoring card | M |
| 5.3 | Candidate transparency: breakdown + plain-language rationale in portal; "screening benchmark not met" state; result/feedback view (GDPR access right) | CandidatePortal.tsx, portal DTO | M |
| 5.4 | Cost rails: explicit `MaxRetry(5)` + timeout per task; token/cost ledger at the LLM port; per-org daily cap; retry-storm alerting | queue config, llm provider, workers | M |
| 5.5 | DR: cron/systemd timer for backup; offsite copy (**hard gate 2026-09-30**); restore recreates `intivai_app` role/privileges/RLS; monthly restore test documented | scripts/backup.sh, restore.sh, docs | M |
| 5.6 | App container non-root + read-only rootfs + `no-new-privileges`; mem limits on ALL prod services; healthchecks on app/sidecar/caddy (`service_healthy`); cap concurrent sandbox runs | backend/Dockerfile, docker-compose.prod.yml, runner | S |
| 5.7 | GDPR: candidate data export + delete endpoints; written retention policy (14-day backup vs legal retention) | new endpoints, portal, docs | M |
| 5.8 | OTP spam hardening: per-email daily cap; surface enqueue failures | candidate_portal_handler.go | S |
| 5.9 | Ops posture: keep last N image tags + documented rollback; uptime check + queue-depth/worker-death alerts; drop whisper from prod until voice ships | ci.yml, docker-compose.prod.yml, docs | S |

---

## Phase 6 — Docs/OpenAPI sync (day 13) 🟢

| # | Task | Effort |
|---|---|---|
| 6.1 | OpenAPI: add the 6 missing endpoints (candidate-review, bulk, deletes, report/pdf); fix `sandbox/execute` `memory_kb` + timeout default (5 vs 10s); complete `/interviews/:id` + `/applications` schemas; document voice auth + `audio` frames | S |
| 6.2 | Research.md: downgrade §3 voice row to "gated demo"; migration counts 001–016; WS frame names (idx/response/session_budget_sec); "idle 5m" → 3-min read deadline; 10MB → 64KB upload; scoring-rails design-intent note; §4.9 stack table (Vite, not Next.js) | S |
| 6.3 | AGENTS.md: `gofmt -l` step in `check` (or reword); `interface{}` → `any`; document the OTP repo layer | S |
| 6.4 | Caddyfile: `{$DOMAIN}` env interpolation; `@index path / /index.html`; HSTS + security headers | S |

---

## Sequencing & gates

1. **Phase 0 → 1**: a tag from Phases 0–1 is the *minimum deployable* — sandbox works, jobs controlled, email real, CI truthful
2. **Phase 2 must ship before any real candidate** — fabricated decision data is the liability
3. **Phases 3–4** are the maintainability spine — complete before adding features
4. **Phase 5.4 (cost rails) and 5.5 (DR) are hard gates for the first paying customer** — the 2026-09-30 offsite-backup deadline is a customer gate, not a soft date
5. **Checkpoints**:
   - After Phases 0–2 → re-run the four-lens review (tech lead / PO / CEO / HR) before opening beta
   - After Phase 4 → re-run the craft audit (coverage gate must be honest; 0% packages visible)
   - Monthly → one documented restore test + one fresh-VPS deploy test

**Estimate**: ~13 focused sprint days, mostly mechanical; the only genuinely architectural work is 3.1–3.2 (OTP/Apply repos), 4.2 + 4.11 (chat refactor), and 5.1 (scoring fairness).
