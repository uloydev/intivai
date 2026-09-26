# Changelog

All notable changes to Intivai. Format: [Keep a Changelog](https://keepachangelog.com);
versions follow git tags (semver-ish: MAJOR.MINOR.PATCH).

## [Unreleased]

## [0.1.0] - 2026-08-27

### Added
- Expose job-level `scoring_weights` on application DTO and OpenAPI schema.
- Candidate 360 role-specific weighting transparency indicator and transcript question quick-jump pills.
- Node 26 jsdom storage polyfill for Vitest test harness.
- Documentation governance system: `docs/` tree, FINDINGS ledger, ADRs 0003–0007,
  runbooks, threat model, SLOs; OpenAPI consolidated to a single spec with a
  CI drift guard (`scripts/check-openapi-drift.sh`).
- Candidate data portal: export (`GET /candidate/portal/export`) and GDPR
  erasure (`DELETE /candidate/portal/me`) with `data_requests` audit trail.
- Outbound webhooks with HMAC-SHA256 signing, retries, delivery logs.
- "Request human interviewer" safety valve on the candidate chat.
- Recruiter decision override (`PUT /interviews/{id}/decision`, migration 022)
  and PDF scorecard export endpoint.

### Changed
- Refactored UI copy across 15 views to eliminate em dashes (`—`) per Hard Gate R-02.
- Removed decorative button arrows (`→`, `←`) across all recruiter and candidate action buttons per R-08.
- Interview room `Chat.tsx` guards topic advance against empty turns and provides touch-safe "Next Topic" micro-label.
- Product positioning: AI Technical Interview Platform (not an ATS); pricing
  revised ($29 Starter, Free tier 10 interviews/mo, credits pack) — `docs/product/pricing.md`.
- Voice interview settled as demo-only, out of beta critical path — ADR-0006.

### Fixed
- Gosec G706 log injection taint sanitized in loadcheck test harness.
- Markdownlint syntax fixes in DESIGN.md.
- JWT validation requires expiry + issuer; auth rate limits fail closed.
- Webhook SSRF guards (redirect + private-range checks).
- LLM fallback retry logic; worker panic recovery; interview state-machine
  transitions from expired/completed rejected.

> Note: no tags existed before this file was introduced (2026-08-24). Tag the
> next release and append entries per deploy from here on.
