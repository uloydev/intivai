# Changelog

All notable changes to Intivai. Format: [Keep a Changelog](https://keepachangelog.com);
versions follow git tags (semver-ish: MAJOR.MINOR.PATCH).

## [Unreleased]

### Added
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
- Product positioning: AI Technical Interview Platform (not an ATS); pricing
  revised ($29 Starter, Free tier 10 interviews/mo, credits pack) — `docs/product/pricing.md`.
- Voice interview settled as demo-only, out of beta critical path — ADR-0006.

### Fixed
- JWT validation requires expiry + issuer; auth rate limits fail closed.
- Webhook SSRF guards (redirect + private-range checks).
- LLM fallback retry logic; worker panic recovery; interview state-machine
  transitions from expired/completed rejected.

> Note: no tags existed before this file was introduced (2026-08-24). Tag the
> next release and append entries per deploy from here on.
