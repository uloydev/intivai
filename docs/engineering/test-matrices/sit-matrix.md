# SIT Matrix — System Integration Test Cases

> Status: stub · Last-reviewed: 2026-08-24 · Owner: EM
> The PRD historically referenced a 58-case SIT matrix (`sit_test_matrix.md`)
> that was never committed. This is the recovery stub: cases are re-derived
> from the shipped integration suites and marked TBD until re-enumerated.
> Automated coverage lives in `backend/internal/**/*_test.go` (env-gated
> integration tests) + `frontend/e2e/` (Playwright).

| # | Area | Case | Autom. | Status |
|---|------|------|--------|--------|
| 1 | Auth | login/register happy path + weak password + duplicate email | yes (iam) | covered |
| 2 | Auth | ws_ticket rejected on API routes; auth token accepted only for `type=auth` | yes | covered |
| 3 | Tenancy | RLS isolation: cross-org reads denied on all tenant tables | yes (rls) | covered |
| 4 | Tenancy | candidate OTP lockout (5 attempts → 429), replay, expiry | yes (portal) | covered |
| 5 | CV | parse→extract→score pipeline incl. OCR fixture + failure states | yes (cv) | covered |
| 6 | Screening | weighted score round-trips; threshold job>org>default; savepoint re-score | yes (screening) | covered |
| 7 | Interview | ticket → start → answer → tokens → next question; interrupt; second-connection rejection; session mismatch; LLM-error advance | yes (ws) | covered |
| 8 | Interview | request-human pauses interview + notifies recruiter | yes | covered |
| 9 | Evaluation | report aggregation; evaluation worker retry idempotency; decision override transitions | partial | verify |
| 10 | Webhooks | delivery + HMAC signature + retry/backoff + SSRF guards | yes (integration) | covered |
| 11 | Sandbox | execute/evaluate round-trip vs sidecar; fail-closed on sidecar outage | env-gated | verify |
| 12 | Portal | apply → portal_token → applications listed; export; erase cascade | yes (portal) | covered |
| 13 | Public board | published-only visibility; per-IP rate limits | yes (job) | covered |
| — | — | remaining historical cases (to 58) | — | TBD |

Runbook: `make check && make test-integration-dev && make smoke`.
