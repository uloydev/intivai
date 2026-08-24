# Findings Ledger

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Single remediation ledger. Every review/audit appends rows here — never
> create a new plan file for findings. Statuses verified against code where
> evidence exists; `needs-verify` = claimed done somewhere but not confirmed
> in code during the 2026-08-24 docs reorganization.

Status values: `open` · `in-progress` · `closed` (evidence noted) · `needs-verify` · `obsolete`.

Sources: **R19** `docs/reviews/2026-08-19-four-lens.md` · **F19** FIXING_PLAN_2026-08-19 (deleted, absorbed here) · **UX19** FIXING_PLAN_UIUX_2026-08-19 (deleted, absorbed here) · **FP** FIX_PLAN 36 findings (untracked, deleted, absorbed here) · **CR22** `docs/reviews/2026-08-22-code.md` obs. 9–15 · **MAP** MERGED_ACTION_PLAN (archived at `docs/plans/archive/merged-action-plan.md`) · **COMP22** `docs/reviews/2026-08-22-competitive.md` · **R24** deep review 2026-08-24, four-agent audit (docs-sync, ops/CI/security, Go backend, React frontend), grouped into `docs/plans/active/review-fix-plan-2026-08-24.md`.

---

## A. Release / deploy integrity

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| A1 | R19#1, F19-0.1 | Review round left uncommitted; tags ship pre-fix code | closed | 2026-08-24 reorg committed everything tracked; working tree still has unrelated modified files — keep committing discipline |
| A2 | F19-1.5, R19#6 | CI smoke port mismatch, no LLM key, smoke not a deploy gate, sed-rewrites compose | needs-verify | ci.yml was modified post-review; re-audit `.github/workflows/ci.yml`. R24 note: smoke IS now a deploy gate (ci.yml:175) and no-LLM-key hard-fail confirmed real → A16; port-mismatch/sed items still need eyes |
| A3 | FP#22 | CI smoke runs dev config (no --env-file) | needs-verify | same re-audit as A2. R24 note: not yet re-examined |
| A4 | R19 TL/PO | Sandbox dead on prod (image lacks sandboxd, certs, exec images unpublished) | open | `make sandbox-images` exists locally; prod pipeline verification pending (beta gate #9/#14) |
| A5 | R19 Ops | Backup has no cron/timer; restore lacks role bootstrap; offsite copy missing | open | ADR-0007 hard date 2026-09-30; beta gate #10. R24 adds: restore.sh:38,41 recreates roles with repo-published passwords on cluster-loss DR → A11 |
| A6 | R19 Ops | App container runs as root; mem limits only on app; no healthchecks on sidecar/caddy | closed | R24 2026-08-24: prod overlay resolved — app non-root `appuser` (backend/Dockerfile:13,21), mem_limits on all services, healthchecks app/sidecar/caddy (docker-compose.prod.yml:74–79,111–116,146–151). Dev-base compose stays bare by design |
| A7 | FP#17 | Prod compose allows empty INTIVAI_LLM_API_KEY default | closed | R24 2026-08-24: `${VAR:?}` fail-fast verified on all dev-secret keys (docker-compose.prod.yml:34,37–38,41,44–48,53) |
| A8 | CR22#13 | Empty publicURL → invite URL malformed; validate at startup | needs-verify | |
| A9 | R24 | Deploy job not gated on frontend job — broken FE tests/lint ship to prod | open | ci.yml:175 `needs` omits `frontend`; deploy re-builds FE so only compile errors block |
| A10 | R24 | Root `make check` + docs lint never run in CI despite Makefile:65 comment claiming it | open | ci.yml has no root-check job; FE gates run only in frontend job, lint-docs nowhere |
| A11 | R24 | restore.sh bootstrap recreates roles with repo-published passwords on cluster-loss DR | open | restore.sh:38,41 hardcoded `'intivai_app'`/`'intivai_rls_bypass'`; rotation is manual post-step |
| A12 | R24 | sandbox-sidecar mounts host docker.sock as root, uncontained | open | docker-compose.prod.yml:104; accepted ADR-0002 risk; DF decision 2026-08-24 = add socket proxy (fix plan 7.11) |
| A13 | R24 | No manual approval/environment protection on prod deploy; no auto-rollback on failed health | open | ci.yml:174–176 auto-deploys on main merge; rollback is comment prose (docker-compose.prod.yml:11–20) |
| A14 | R24 | Runtime images tag-pinned not digest-pinned; alpine mismatch builder 3.23 vs runtime 3.20 | open | backend/Dockerfile:2,:11; sandbox images rebuilt from mutable tags each deploy (ci.yml:201–210) |
| A15 | R24 | Prod Redis unauthenticated (internal-network only) | open | docker-compose.prod.yml:125–131, no requirepass |
| A16 | R24 | Smoke CI job hard-fails without INTIVAI_LLM_API_KEY secret | open | scripts/smoke.sh:59–61 exits 1 on failed_extract; repo without secret = permanently red |
| A17 | R24 | golangci-lint installed via per-run `go install` — slow, toolchain drift | open | ci.yml:23 (@v2.10.1 but unpinned binary) |
| A18 | R24 | No security linters (gosec etc.) in .golangci.yml for PII-handling prod codebase | open | backend/.golangci.yml standard set only (:8–13) |
| A19 | R24 | Playwright E2E exists (6 specs) but runs nowhere automated; F18 stall blocks required status | open | DE decision 2026-08-24 = nightly scheduled workflow first (fix plan 7.3) |

## B. Data honesty / fairness

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| B1 | R19#2, F19-2.1, UX19-P1 | Fabricated decision data (`?? 85`, "Spotless" integrity, canned recommendations, fake resume facts) | needs-verify | Highest-priority re-audit: grep `?? 85\|\|\| 85\|Spotless` in FE + pdf.go before any customer. R24 note: pdf.go integrity-default-100 CONFIRMED real → B10; FE fabricated strengths CONFIRMED real → G10; remaining sub-items grep pending |
| B10 | R24 | PDF scorecard invents IntegrityScore 100 from zero data; prints static "Verified Session Authenticity" | closed | 2026-08-24: zero-telemetry branch renders "No telemetry recorded", omits score + authenticity line (pdf.go); TDD — 3 new PDF text-extraction tests, red phase showed fabricated "Integrity Score: 100/100 · Verified Session Authenticity" output. Discovery: summary.Flags never rendered anywhere — separate gap, not fabricated data |
| B2 | R19 HR | Degree-ism at auto-gate (missing degree scores 0) | open | scoring.go neutral default 0.5 |
| B3 | R19 HR | Empty dimensions score 0 (never-probed = penalized) | open | neutral-fill policy needed |
| B4 | R19 HR | LLM recommendation enum unvalidated; one hallucination = verdict | needs-verify | evaluator validation may have landed with P4a work |
| B5 | R19 HR | Consent doesn't disclose monitoring scope; coerced consent | open | consent copy + monitoring-free alternative |
| B6 | R19 HR | Ghosted candidates: unknown email type dropped, no outcome emails, portal "in progress" forever | needs-verify | email worker types were extended; verify `candidate_review` path |
| B7 | UX19-1.1..1.8 | Missing/null recommendation renders green PROCEED; invented "Strong Hire" KPI; voice page reads recruiter token; dead stage toggles; reconnect input stays enabled; third-person transcript entries; ProctoringCard 0/100 on no data; invite link missing token | needs-verify | several files rewritten since 08-19 (useChatSession, stages.ts exist); walk each before beta |
| B8 | UX19-2.1..2.11 | Candidate journey dead ends (post-interview dead end, timer 00:00 idle, expired invite toast-only, sandbox squeezes chat, over-promised copy) | needs-verify | same sweep as B7 |
| B9 | COMP22 §9.3 | Recruiters need override control + full transcript on scorecard | closed | PUT `/interviews/{id}/decision` + migration 022 recruiter_decisions shipped; transcript viewer per UX19-3.7 needs-verify |

## C. Security

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| C1 | R19 TL | JWT Parse lacked expiration/issuer requirements | closed | jwt_provider.go:57 `WithExpirationRequired()` + `WithIssuer("intivai")` verified 2026-08-24 |
| C2 | R19 TL | Rate limiter fail-open on auth buckets; tenantRateLimit registered before authMW (dead); public endpoints unlimited | closed | ratelimit.go returns 503 on `/auth/` Redis errors; main.go:432 registers limits after authMW; public routes carry authRateLimit — verified 2026-08-24 |
| C3 | FP#1, FP#5 | Webhook SSRF: redirect bypass + DNS rebinding | closed / needs-verify | CheckRedirect guard verified webhook_worker.go:46; net.LookupIP rebinding check needs spot-check in domain/webhook.go. R24 2026-08-24: rebinding defense CONFIRMED — scheme allowlist, localhost/.local rejection, IP-literal checks, per-address validation during resolution (domain/webhook.go:63–113) + redirect re-validation (webhook_worker.go:46–51); HMAC signing + 1024 B response cap also verified |
| C4 | FP#3, FP#27 | Unsafe JWT claims type assertions (panic risk) in chat handler | needs-verify | comma-ok refactor claimed; unit-test grep pending |
| C5 | CR22#15 | Context download without org re-verification at storage layer | open | defense-in-depth: validate object path org prefix |
| C6 | R19 CEO | Public apply = unauthenticated LLM-spend/spam vector | open | per-IP limit added on apply route (authRateLimit present); per-email daily cap (F19-5.8) unverified. R24 note: still no per-email cap → C10; telemetry limiter exempt found → D23 |
| C7 | COMP22/MAP P0-3 | Hide proctoring UI from candidates | needs-verify | Chat.tsx rewritten since; confirm no indicators render |
| C8 | R19 HR | Candidate JWT 7d in localStorage | open | accepted risk for beta; revisit token storage |
| C9 | R24 | CRITICAL portal auth bypass: public apply returns portal_token for any email without mailbox proof → candidate JWT for victim → cross-org application listing leaks scores + live invitation_token → attacker joins victim interview | open | job/api/public_handler.go:199–215 (token in response :214); postgres_portal_repo.go:143–146 (lookup SELECTs invitation_token across orgs). Fix plan Batch 1a/1b (DB decision: both staged) |
| C10 | R24 | Public apply lacks per-email daily cap (per-IP bucket exists via authRateLimit) | open | fix plan Batch 1a.3 |

## D. Architecture / correctness

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| D1 | F19-1.4 | Rubric worker: missing tenant tx → dead-letter; full-row clobber; retry re-invokes LLM | needs-verify | MaxRetry(5) + OrgID payload present (job_service.go:277); tx/clobber fixes need code read |
| D2 | F19-3.1 | OTP access must go through repo layer (no handler SQL) | closed | R24 2026-08-24: OTP repo layer EXISTS — postgres_portal_repo.go:83,98,128 implements FindValidByToken/FindValidByCodeHash/PurgeExpired (+IncrementAttempts/Consume); handlers repo-only (candidate_portal_handler.go:89,150,152). Original 2026-08-24-morning audit searched wrong location; AGENTS.md claim correct today. Fix plan 0.1 closure |
| D3 | F19-3.2 | Public-apply SQL belongs in repo (advisory lock + ON CONFLICT) | needs-verify | |
| D4 | F19-3.5 | Public-apply cleanup deletes re-applying candidate's CV; stuck `parsing` on enqueue failure | needs-verify | |
| D5 | F19-3.6 | Memory layer double-begin deadlock pattern + uncached SQLite handle | needs-verify | original M3 carryover said fixed; re-confirm |
| D6 | F19-3.7 | Context upload inside FOR UPDATE tx | needs-verify | |
| D7 | CR22#12 | Probe insert non-atomic (bank create vs interview update) | open | savepoint or outbox pattern |
| D8 | CR22#11 | Context MinIO upload failure after commit → orphaned row | open | compensating delete or upload-before-commit |
| D9 | CR22#10 | Invitation email enqueue error silently discarded | open | surface to recruiter UI |
| D10 | FP#6 | Webhook delivery status overwrite race | needs-verify | conditional UPDATE claimed |
| D11 | FP#7, FP#21 | Duplicate next-question after stream error; debounce lastTouch not updated | needs-verify | chat_handler heavily refactored (useChatSession/signalMu work landed). R24 note: duplicate-question CONFIRMED reachable via overlapping answer turns → D19; lastTouch debounce part still to check in fix-plan Batch 3 tests |
| D12 | FP#23 | json.Marshal error ignored in LLM provider | open | trivial fix |
| D13 | FP#4, FP#15, FP#16, FP#33, FP#34 | PDF layout overflow / malformed-eval zero-score / truncation / clamps | needs-verify | pdf.go had test additions (pdf_test.go modified) |
| D14 | FP#20, FP#31 | data_requests action CHECK constraint + STRICT | needs-verify | migration 024 exists; constraint content unchecked |
| D15 | R24 | Scorecard emails can never send — RLS `users` query outside tenant tx, always zero rows, silent | closed | 2026-08-24: recruiterEmails wrapped in db.RunInTx with tx handle (evaluation_worker.go:186–218, mirrors main.go webhook pattern); resolve errors no longer swallowed (Warn + org_id). Integration test TestEvaluationWorkerScorecardEmailsResolveOrgUsers red→green incl. cross-org non-leak assertion |
| D16 | R24 | Cached interview PDF served without ownership check on cache-hit path (cross-org read via known UUID) | closed | 2026-08-24: authorizeInterviewAccess (RunInTx + org compare, NotFoundError semantics) runs before Exists()/Download() fast path (service.go:296–301,:338–356). Integration tests: cross-org forbidden on cold AND warm cache; own-org warm cache still served |
| D17 | R24 | Proctoring events lost-update race — read-modify-write on JSONB vs atomic append used elsewhere | open | postgres_interview_repo.go:100–136 vs RecordCodingSession :161 pattern; two concurrent writers drop one event + stale summary. Fix plan 2.3 |
| D18 | R24 | wsWriter deadlock — close() waits on mutex held by sender blocked on connCtx canceled AFTER close; session lock stuck up to 35 min | open | chat_handler.go defer order :572–586 (`cancel()` LIFO after `close()`); writer death leaves ≤64 buffered frames; Release never runs. Fix plan 3.1 |
| D19 | R24 | WS protocol allows overlapping answer turns — interleaved streams, duplicate next-question, transcript cursor corruption | open | chat_handler.go:702–705 no busy guard; handleAnswer :307–345 overwrites turn/streamCancel mid-stream. Fix plan 3.2 |
| D20 | R24 | Webhook + index workers swallow ALL errors incl. transient as SkipRetry — deliveries stuck pending forever, silent | open | webhook_worker.go:99–102; context_service.go:248–250 same pattern. Violates AGENTS rule 8. Fix plan 4.1 |
| D21 | R24 | db.WrapError converts every unknown repo error into DomainError → renders 400 + leaks internal text (e.g. "context canceled") | open | pkg/db/errors.go:38; httpapi.Error default-maps DomainError→400. Fix plan 5.1 |
| D22 | R24 | Demo-token bypass compiled into prod auth path, no env gate — infinitely reusable portal creds if demo rows reach shared DB | open | postgres_portal_repo.go:84,:117 `strings.HasPrefix(token,"demo-")`. Fix plan 5.3 |
| D23 | R24 | Telemetry HTTP route rate-limit-exempt — userRateLimit keyFn returns "" without Actor, limiter skipped | open | main.go:428 + :392–396 keyFn; ratelimit.go:24–26 skip logic. Fix plan 5.4 |
| D24 | R24 | Sandbox unbounded concurrent executions per connection; HTTP execute test-cases uncapped; sidecar/daemon error text reaches candidates; kill failure unlogged | open | chat_handler.go:302 goroutine-per-run no semaphore; HTTP path lacks the WS 20-case cap (:714); runner.go:238 res.Error raw; :216 `_ = r.docker.Kill`. Fix plan 5.2/5.5 |
| D25 | R24 | Swallowed state-transition errors in interview advance (`_ = iv.Complete()`, probe insert skips) | open | interview_service.go:398–400, :391–395. Fix plan 4.5 |
| D26 | R24 | Extract worker persists raw cause.Error() to candidate-visible ErrorMessage; transient store failures mislabeled terminal failed_ocr | open | extract_worker.go:276 (leak), :65–68 (mislabel). Fix plan 4.3/5.7 |
| D27 | R24 | Bulk CV upload silently drops individual files — four bare continues, no logging, no per-file outcome | open | cv/application/cv_service.go:141,148,152,158. Fix plan 4.2 |

## E. Coverage / tests honesty

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| E1 | F19-3.3 | Coverage-gate parser misses bare `coverage: 0.0%` lines | closed | scripts/check-coverage.sh:27 matches tab-indented bare lines — verified 2026-08-24 |
| E2 | F19-3.4 | Zero tests: UpdateDecision transitions, OTP lockout, worker SkipRetry semantics | needs-verify | candidate_portal_test.go covers OTP lockout paths; UpdateDecision table test unconfirmed. R24 note: OTP lockout/replay integration CONFIRMED (candidate_portal_test.go:33,:124); UpdateDecision + SkipRetry-semantics tests still pending (SkipRetry → D20 fix) |
| E3 | R24 | Rubric worker has no dedicated test file — tx usage, retry/LLM idempotency, clobber-safety unverified by tests | open | job/application/rubric_worker.go:31. Fix plan 4.4 (also resolves D1 verification) |
| E4 | R24 | OpenAPI drift guard blind to helper-mounted routes — greps only direct route literals; the one live drift escaped exactly this way | closed | 2026-08-24: guard scans internal handler `.Verb("/path"` literals (v1-prefix assumption, documented); self-test scripts/test-check-openapi-drift.sh 5/5 — helper-missing detection, no-false-positive, main.go regression, synced case, spec-only case. Self-test exposed + fixed 2 guard bugs (set -e silent scan abort; comm -3 inverted label parsing). Real guard exit 0 on current tree |

## F. Product / competitive gaps

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| F1 | MAP P0-1, COMP22 | Magic link primary (OTP fallback) | closed | magic-link flow present in candidate_portal_handler + email worker templates — verified 2026-08-24 |
| F2 | MAP P0-4, COMP22 §5.1 | Outbound webhooks w/ HMAC + retries + delivery logs | closed | integration context shipped: routes, worker, SSRF guards — verified 2026-08-24 |
| F3 | MAP P0-5, COMP22 | Candidate data portal (export + erase + audit) | closed | GET /candidate/portal/export, DELETE /candidate/portal/me, migration 024 — verified 2026-08-24 |
| F4 | MAP P0-9 | Request-human-interviewer safety valve | closed | POST request-human route + handler + tests — verified 2026-08-24 |
| F5 | MAP P0-10, COMP22 §6.2 | Reposition as AI Technical Interview Platform | closed | PRD rewritten 2026-08-24 (docs reorg) |
| F6 | MAP P0-6 | Recruiter override of AI recommendation | closed | see B9 |
| F7 | MAP P0-7 | Full transcript on scorecard | needs-verify | TranscriptViewer component existence unconfirmed |
| F8 | MAP P0-2/P0-8, COMP22 | Mobile-first candidate flow + connection status indicator | needs-verify | Chat.tsx/useChatSession rewritten; run UAT matrix scenarios 6/8 on 375px |
| F9 | MAP P1-1 | SOC 2 prep docs | closed (docs only) | docs/compliance/* created; truth-pass corrections applied 2026-08-24; certification NOT claimed |
| F10 | MAP P1-2, COMP22 | DOCX parsing | closed | R24 2026-08-24: DOCX parsing SHIPPED — parse_worker.go:19 (`github.com/nguyenthenguyen/docx`), dispatch :73–76, extraction :206–228. Original finding searched cv/infrastructure; implementation lives in cv/application. Fix plan 0.1 closure |
| F11 | MAP P1-3, COMP22 | PDF scorecard export | closed | GET /interviews/{id}/report/pdf registered in main.go — verified 2026-08-24 |
| F12 | MAP P1-4, C1 | Encryption-at-rest claims | open | NOT implemented (no SSE-S3/pgcrypto config found); PRD now says Target Q4 2026 |
| F13 | MAP P1-5 | Credits pricing tier | closed | pricing.md published 2026-08-24 |
| F14 | COMP22 §5.1 | ATS native integrations (Greenhouse/Lever) | open | post-beta by design |
| F15 | COMP22 | Reporting depth vs Manatal | open | accepted gap; post-beta |
| F16 | COMP22 §9.1 | Kill Passport from MVP | closed | PRD Epic 8 = Won't-have (beta); ADR-worthy when revisited |
| F17 | MAP UI/UX phases 0–5 | Design tokens, component upgrades, page revamps, a11y sweep, Lighthouse ≥90 | needs-verify | components exist (alert-dialog, stages.ts, useChatSession, EmptyState?); run MAP acceptance lists as UAT pre-beta pass. R24 note: TDZ audit CLEAN — all static tables module-scope, in-render helpers declared before use; a11y effort real (aria-sort, role=log, focus mgmt, IME-aware Enter). Visual/UAT pass still pending |
| F18 | E2E 2026-08-24 | happy-path stalls after `advance` — Q2 frame never arrives within 120s against reasoning-model LLM (ox-alpha via OpenRouter); smoke's plain `answer` loop passes end-to-end, so backend turn machine works; suspect UI advance path (`submittingRef`/`pendingAnswer` gating) or multi-minute stream latency vs FE state | open | 30/31 specs pass after raising Playwright timeouts (420s test / 120s expect); test wait `/Interrupted\|Real-Time AI Session/i` matches the banner instantly — weak signal, needs a response-frame-driven wait. R24 note: fix folded into fix plan 7.3 (nightly E2E); D19 overlap guard may also touch advance path |

## G. Frontend defects

R24 source. React SPA defects from the 2026-08-24 frontend audit; grouped into `docs/plans/active/review-fix-plan-2026-08-24.md` Batches 3/6.

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| G1 | R24 | Every successful DELETE crashes client — 204 empty body → `(null).data` TypeError; GDPR erase, CV delete, webhook delete, context delete all affected; server succeeds but UI shows error + cache never invalidated | closed | 2026-08-24: request() early-returns undefined on 204 before body parse (api.ts); vitest red phase reproduced exact TypeError at api.ts:58, green: build + 22/22 tests pass. All four call sites verified to invalidate cache + toast in onSuccess once throw removed; `api.delete<void>` typing already satisfied |
| G2 | R24 | WS ticket TTL 10 min vs 30-min interview, no re-mint path — reconnect/refresh after minute 10 permanently locked out ("Connection Lost", reload reuses same expired ticket) | open | useChatSession.ts:88–92 reuses original ticket; interview_service.go:30 ticketTTL; MaxInterviewDuration 30 min. Fix plan 3.3 |
| G3 | R24 | Reconnect budget cumulative for whole session, never resets on success — six transient blips over healthy session = permanent disconnected | open | useChatSession.ts:33,:84–99 counter incremented, reset absent from :102–110 success handler. Fix plan 3.4 |
| G4 | R24 | Voice page sends recruiter auth JWT as `?ticket=` query param on public route — type confusion + leak into history/logs; candidates hit dead end with empty token | open | InterviewVoice.tsx:81–82; page public (App.tsx:79). DA decision 2026-08-24 = keep voice mounted → mandatory proper ws_ticket mint flow. Fix plan 3.6 |
| G5 | R24 | Voice onclose leaves mic + RTCPeerConnection alive while UI says "Call ended" (onerror cleans up, onclose doesn't) | open | InterviewVoice.tsx:132–142 vs :144–152. Fix plan 3.6 |
| G6 | R24 | Magic-link token never stripped from URL — refresh replays spent single-use token, onError wipes valid stored session via handleLogout | open | CandidatePortal.tsx:137–143 effect, :131–134 logout-on-error; reachable via Careers.tsx:162 navigate. Fix plan 3.5 |
| G7 | R24 | Silent failure on Publish/Unpublish/Archive job mutations — no onError, button does nothing on 4xx/5xx | open | Jobs.tsx:106–122 patchStatus/patchPublished. Fix plan 6.3 |
| G8 | R24 | Bulk stage-update/invite mask partial failures — Promise.allSettled never rejects, catch dead, success toast with count 0 possible | open | Candidates.tsx:220–231,:246–257. Fix plan 6.3 |
| G9 | R24 | Sentry Session Replay records candidate PII (transcripts, OTP screens) at 100% error-sampling with no masking — GDPR defect in Art.17 product | open | main.tsx:10–19 replayIntegration, no beforeSend/mask config. Fix plan 6.1 |
| G10 | R24 | Fabricated static "AI-derived" strengths/growth shown to every completed candidate regardless of performance or role | open | CandidatePortal.tsx:672–700 hardcoded strings under "Automated feedback". B1-class honesty defect. Fix plan 6.2 |
| G11 | R24 | FE minors bundle: no token-stream visibility during LLM response; per-keystroke code_change WS flood (no debounce); timer auto-submit toast fires even when send failed; toast spam on deep-linked missing candidate + polling; duplicate test-case IDs (length+1); dead questionIdx prop / no code-state reset between questions; `as unknown as` casts ×2; implicit-any JSON.parse in voice; dev health check always-green via Vite fallback; getBlob missing timeout/ApiError/401 parity; clipboard promise without catch; unclamped question count input; stale-auth CTA ignores expiry; SOC 2 footer claim before certification; emptyAutoRefetchRef not reset on identity change; formatting helpers inside render bodies (repo convention violation); SortIcon recreated per render; RequestHuman posts ticket value under invitation_token field; OTP resend no cooldown UI; StrictMode double-fire window on autostart; scroll-FAB positioning container | open | file:line per fix plan Batch 6 minors sweep list |

## H. Docs truth

R24 source. Documentation contradicting verified reality; violates repo governance "compliance/docs describe reality". All resolved by fix plan Batch 0 unless noted.

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| H1 | R24 | Four docs claim voice route unmounted while code mounts it unconditionally | closed | 2026-08-24: DA decision = keep mounted; ADR-0008 written (supersedes unmount clause), ADR-0006 amended, roadmap §5 + design-decisions §3 rewritten to mounted-demo-gated, openapi.yaml gained `/api/v1/interviews/{id}/voice` path; drift guard exits 0 |
| H2 | R24 | Backend Sentry claimed wired but never initialized — dep + config only | closed | 2026-08-24: docs made truthful — beta-gate #11 flipped to `[ ]` with evidence, design-decisions §4.6 marks backend Sentry NOT initialized. Actual wiring tracked as fix-plan 5.6 (DC decision) |
| H3 | R24 | P5 criteria checked `[x]` for STT adapter/audio playback while sync table says pipeline mocked, Opus unimplemented | closed | 2026-08-24: roadmap P5 criteria honesty pass — demo-pipeline rows annotated `[x] … demo pipeline`, real-STT/TTS/playback rows unchecked |
| H4 | R24 | Stale ranges/anchors in live docs: migrations "001–009" and fresh-boot "001–007" (actual 025); schemas.md line anchors off | closed | 2026-08-24: design-decisions §4.5 → 001–025; beta-gate #8 → 001–025; schemas.md anchors corrected to config.go:153 + context_service.go:29 (verified by grep) |
| H5 | R24 | Duplicate top-level YAML keys /health,/live,/ready in OpenAPI spec — last-wins parsing ambiguity | closed | 2026-08-24: deduplicated to single definitions at top of paths: (richer variants kept); guard self-test case pins spec-only detection |
| H6 | R24 | Executed docs-reorganization plan left in active/ with unchecked DoD §8; index claims completeness it lacks | closed | 2026-08-24: moved to plans/archive/docs-reorganization-plan.md with Status header + DoD ticked; docs/README.md updated (plan row + added design-system/* + frontend/README entries) |

## Ledger rules

1. New findings get the next free ID in their section (or a new section).
2. Changing status requires evidence link (file:line, command output, or commit).
3. Reviews never edit old rows except to append Notes.
