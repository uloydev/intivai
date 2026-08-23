# Findings Ledger

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Single remediation ledger. Every review/audit appends rows here — never
> create a new plan file for findings. Statuses verified against code where
> evidence exists; `needs-verify` = claimed done somewhere but not confirmed
> in code during the 2026-08-24 docs reorganization.

Status values: `open` · `in-progress` · `closed` (evidence noted) · `needs-verify` · `obsolete`.

Sources: **R19** `docs/reviews/2026-08-19-four-lens.md` · **F19** FIXING_PLAN_2026-08-19 (deleted, absorbed here) · **UX19** FIXING_PLAN_UIUX_2026-08-19 (deleted, absorbed here) · **FP** FIX_PLAN 36 findings (untracked, deleted, absorbed here) · **CR22** `docs/reviews/2026-08-22-code.md` obs. 9–15 · **MAP** MERGED_ACTION_PLAN (archived at `docs/plans/archive/merged-action-plan.md`) · **COMP22** `docs/reviews/2026-08-22-competitive.md`.

---

## A. Release / deploy integrity

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| A1 | R19#1, F19-0.1 | Review round left uncommitted; tags ship pre-fix code | closed | 2026-08-24 reorg committed everything tracked; working tree still has unrelated modified files — keep committing discipline |
| A2 | F19-1.5, R19#6 | CI smoke port mismatch, no LLM key, smoke not a deploy gate, sed-rewrites compose | needs-verify | ci.yml was modified post-review; re-audit `.github/workflows/ci.yml` |
| A3 | FP#22 | CI smoke runs dev config (no --env-file) | needs-verify | same re-audit as A2 |
| A4 | R19 TL/PO | Sandbox dead on prod (image lacks sandboxd, certs, exec images unpublished) | open | `make sandbox-images` exists locally; prod pipeline verification pending (beta gate #9/#14) |
| A5 | R19 Ops | Backup has no cron/timer; restore lacks role bootstrap; offsite copy missing | open | ADR-0007 hard date 2026-09-30; beta gate #10 |
| A6 | R19 Ops | App container runs as root; mem limits only on app; no healthchecks on sidecar/caddy | open | hardening for first paying customer |
| A7 | FP#17 | Prod compose allows empty INTIVAI_LLM_API_KEY default | needs-verify | check docker-compose.prod.yml `:?` syntax |
| A8 | CR22#13 | Empty publicURL → invite URL malformed; validate at startup | needs-verify | |

## B. Data honesty / fairness

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| B1 | R19#2, F19-2.1, UX19-P1 | Fabricated decision data (`?? 85`, "Spotless" integrity, canned recommendations, fake resume facts) | needs-verify | Highest-priority re-audit: grep `?? 85\|\|\| 85\|Spotless` in FE + pdf.go before any customer |
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
| C3 | FP#1, FP#5 | Webhook SSRF: redirect bypass + DNS rebinding | closed / needs-verify | CheckRedirect guard verified webhook_worker.go:46; net.LookupIP rebinding check needs spot-check in domain/webhook.go |
| C4 | FP#3, FP#27 | Unsafe JWT claims type assertions (panic risk) in chat handler | needs-verify | comma-ok refactor claimed; unit-test grep pending |
| C5 | CR22#15 | Context download without org re-verification at storage layer | open | defense-in-depth: validate object path org prefix |
| C6 | R19 CEO | Public apply = unauthenticated LLM-spend/spam vector | needs-verify | per-IP limit added on apply route (authRateLimit present); per-email daily cap (F19-5.8) unverified |
| C7 | COMP22/MAP P0-3 | Hide proctoring UI from candidates | needs-verify | Chat.tsx rewritten since; confirm no indicators render |
| C8 | R19 HR | Candidate JWT 7d in localStorage | open | accepted risk for beta; revisit token storage |

## D. Architecture / correctness

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| D1 | F19-1.4 | Rubric worker: missing tenant tx → dead-letter; full-row clobber; retry re-invokes LLM | needs-verify | MaxRetry(5) + OrgID payload present (job_service.go:277); tx/clobber fixes need code read |
| D2 | F19-3.1 | OTP access must go through repo layer (no handler SQL) | open | No postgres_otp_repo found 2026-08-24 — AGENTS.md claims this exists; AGENTS row is WRONG until implemented (fix AGENTS or build it) |
| D3 | F19-3.2 | Public-apply SQL belongs in repo (advisory lock + ON CONFLICT) | needs-verify | |
| D4 | F19-3.5 | Public-apply cleanup deletes re-applying candidate's CV; stuck `parsing` on enqueue failure | needs-verify | |
| D5 | F19-3.6 | Memory layer double-begin deadlock pattern + uncached SQLite handle | needs-verify | original M3 carryover said fixed; re-confirm |
| D6 | F19-3.7 | Context upload inside FOR UPDATE tx | needs-verify | |
| D7 | CR22#12 | Probe insert non-atomic (bank create vs interview update) | open | savepoint or outbox pattern |
| D8 | CR22#11 | Context MinIO upload failure after commit → orphaned row | open | compensating delete or upload-before-commit |
| D9 | CR22#10 | Invitation email enqueue error silently discarded | open | surface to recruiter UI |
| D10 | FP#6 | Webhook delivery status overwrite race | needs-verify | conditional UPDATE claimed |
| D11 | FP#7, FP#21 | Duplicate next-question after stream error; debounce lastTouch not updated | needs-verify | chat_handler heavily refactored (useChatSession/signalMu work landed) |
| D12 | FP#23 | json.Marshal error ignored in LLM provider | open | trivial fix |
| D13 | FP#4, FP#15, FP#16, FP#33, FP#34 | PDF layout overflow / malformed-eval zero-score / truncation / clamps | needs-verify | pdf.go had test additions (pdf_test.go modified) |
| D14 | FP#20, FP#31 | data_requests action CHECK constraint + STRICT | needs-verify | migration 024 exists; constraint content unchecked |

## E. Coverage / tests honesty

| ID | Source | Finding | Status | Evidence / notes |
|---|---|---|---|---|
| E1 | F19-3.3 | Coverage-gate parser misses bare `coverage: 0.0%` lines | closed | scripts/check-coverage.sh:27 matches tab-indented bare lines — verified 2026-08-24 |
| E2 | F19-3.4 | Zero tests: UpdateDecision transitions, OTP lockout, worker SkipRetry semantics | needs-verify | candidate_portal_test.go covers OTP lockout paths; UpdateDecision table test unconfirmed |

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
| F10 | MAP P1-2, COMP22 | DOCX parsing | open | migration 023 cv_format exists but no docx parser package under cv/infrastructure — finish implementation |
| F11 | MAP P1-3, COMP22 | PDF scorecard export | closed | GET /interviews/{id}/report/pdf registered in main.go — verified 2026-08-24 |
| F12 | MAP P1-4, C1 | Encryption-at-rest claims | open | NOT implemented (no SSE-S3/pgcrypto config found); PRD now says Target Q4 2026 |
| F13 | MAP P1-5 | Credits pricing tier | closed | pricing.md published 2026-08-24 |
| F14 | COMP22 §5.1 | ATS native integrations (Greenhouse/Lever) | open | post-beta by design |
| F15 | COMP22 | Reporting depth vs Manatal | open | accepted gap; post-beta |
| F16 | COMP22 §9.1 | Kill Passport from MVP | closed | PRD Epic 8 = Won't-have (beta); ADR-worthy when revisited |
| F17 | MAP UI/UX phases 0–5 | Design tokens, component upgrades, page revamps, a11y sweep, Lighthouse ≥90 | needs-verify | components exist (alert-dialog, stages.ts, useChatSession, EmptyState?); run MAP acceptance lists as UAT pre-beta pass |

## Ledger rules

1. New findings get the next free ID in their section (or a new section).
2. Changing status requires evidence link (file:line, command output, or commit).
3. Reviews never edit old rows except to append Notes.
