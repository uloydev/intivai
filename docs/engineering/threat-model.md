# Threat Model (STRIDE-lite)

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Living doc — revisit on every new external surface. Mitigation statuses link
> to FINDINGS IDs where a gap remains.

Assets at risk: candidate PII (CVs, transcripts, scores), tenant prompts/context
(trade secrets), LLM spend, hiring-decision integrity.

| # | Surface | Threat | Mitigation | Status |
|---|---|---|---|---|
| T1 | Multi-tenant DB | Cross-tenant data access | RLS ENABLE+FORCE on all tenant tables; least-privilege `intivai_app`; every query in tenant tx; SECURITY DEFINER functions for pre-auth lookups | Implemented |
| T2 | Public careers board | Spam applications → unauthenticated LLM-spend amplification | Per-IP rate limit on apply + auth routes; publish gate (`is_published`) | Partial — per-email daily cap open (C6) |
| T3 | Candidate portal / OTP | Brute force & enumeration of OTP/magic links | SHA256-hashed codes, attempt counter, 429 lockout, single-use tokens, expiry | Implemented (tests cover lockout) |
| T4 | WS chat upgrade | CSWSH / ticket theft / cross-interview access | Origin allowlist (`INTIVAI_ALLOWED_ORIGINS`); ws_ticket bound to interview+session, 10-min TTL; ticket type rejected on API routes; second socket rejected | Implemented |
| T5 | Prompt injection | Candidate or tenant content hijacks interviewer/evaluator | `ContainsInjection` rails at upload; tenant prompt length cap + integrity keywords; safety rails pinned LAST and non-overridable; evaluator output schema-validated, final score recomputed by domain | Implemented; adversarial test corpus open |
| T6 | Sandbox execution | Malicious code escapes / DoSes host | gRPC sidecar owns Docker socket; per-language images; non-root, no network, mem/pids/cpu caps, read-only rootfs, timeout, mTLS internal-only | Implemented in design+dev (ADR-0002); prod deploy verification = beta gate #9 |
| T7 | LLM provider | Cost blowout via retry storms or abusive tenants | Explicit MaxRetry(5); fail-fast on permanent errors | **Open**: token ledger + per-org daily caps (llm-outage runbook) |
| T8 | Webhooks | SSRF into internal network; replay/forgery | Redirect guard + private-range/IP checks; HMAC-SHA256 signatures; delivery logs | Implemented (redirect verified; rebinding spot-check C3) |
| T9 | Proctoring data | Tampered telemetry framing candidates | Advisory-only posture (ADR-0005): client-reported events labeled unverified, human review path, never auto-fails | Implemented (posture) |
| T10 | Object storage | Cross-tenant object reads; bucket exposure | Org-scoped object paths; buckets not public | Open hardening: verify org check at storage layer (CR22#15 / C5) |
| T11 | Secrets | Leakage via repo/logs/errors | `.env.prod` gitignored; compose fails fast on missing secrets; raw errors never returned by API | Implemented; periodic grep for accidental commits |
| T12 | Transport | MITM | TLS 1.3 via Caddy auto-TLS; HSTS pending (FINDINGS A-track) | Partial |

## Review triggers

New endpoint type · new third-party integration · sandbox runtime change ·
auth flow change · before enterprise pilot.
