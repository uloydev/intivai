# Beta Gate — definition of "beta started"

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Single home for beta-gate status. History: defined in M3_Plan (EM decision,
> 2026-08-10); scope change approved there moved P4a (evaluation core +
> recruiter dashboard-lite + invite + consent) and P6a (deploy + backup +
> alerting-lite) INTO the MVP, with the candidate chat UI as a P3 deliverable.
> Beta = Phase 0 cohort (5 pilots).

Scope change approved: P4a and P6a move INTO the MVP; the candidate chat UI
becomes a P3 deliverable. Beta = Phase 0 cohort (5 pilots).

| # | Gate item | Status |
|---|-----------|--------|
| 1 | P4a: evaluation LLM → report JSON; `evaluation` frame carries real scores | [x] |
| 2 | P4a: `GET /interviews` + `GET /interviews/:id` + `GET /candidates/:id/report` (JSON) | [x] |
| 3 | P3 FE: candidate chat UI (ticket → consent → interview → evaluation frame) | [x] |
| 4 | P4a FE: recruiter dashboard-lite (CV/job upload, interview list, result view) | [x] |
| 5 | Invite flow: shareable interview URL from invitation token | [x] |
| 6 | Consent capture: `consent_given` recorded at interview start | [x] |
| 7 | Live LLM streaming verified with real key (smoke + Playwright E2E) | [x] |
| 8 | Fresh-volume boot 001–025 (`make dev` from clean volume) | [x] |
| 9 | Deploy: compose on VPS, domain + TLS, env management, push pipeline | [~] pipeline + overlay ready; needs VPS/domain/secrets |
| 10 | Backup & DR: postgres dump + MinIO mirror → backup bucket; restore test executed | [~] scripts ready; needs host cron + first restore test (see ADR-0007 offsite deadline). R24 note: restore.sh recreates DR roles with repo-published passwords (ledger A11) — fix before first real drill |
| 11 | Error alerting lite (Sentry Go, DSN-gated) | [~] wired (no prod DSN yet) — `pkg/observability` init + fibersentry middleware + worker panic capture; enable by setting `INTIVAI_SENTRY_DSN`. DC decision 2026-08-24 executed (fix plan 5.6) |
| 12 | 5 pilot companies onboarded; feedback channel + retention criteria set | [ ] business |
| 13 | `make check` + `make coverage` + `make test-integration-dev` green | [x] |
| 14 | All commits pushed; deploy from tagged release | [ ] |

Beta starts when gates 1–13 are checked and 14 is done — no silent partial
beta. Gate status is counted on a **tagged release**, not the working tree.
