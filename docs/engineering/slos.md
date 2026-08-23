# Service Level Objectives

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> NFR targets from the PRD expressed as measurable SLOs. "Actual" columns are
> filled from load runs (`make load-ws`, `make load-k6`) and prod metrics —
> record a dated actual every run; missing actuals = SLO unverified.

| # | SLI | SLO (beta) | Measurement source | Actual (date) |
|---|-----|-----------|--------------------|---------------|
| S1 | First LLM streaming token latency | p95 < 1.5 s | Prometheus `llm_request_duration` + chat handler histogram | not yet recorded |
| S2 | CV extraction end-to-end | p95 < 30 s | worker duration metric / asynq observability | not yet recorded |
| S3 | Sandbox execution round-trip | p95 < 3 s | sandbox client timing | not yet recorded |
| S4 | Public page load (careers, invite) | p95 < 500 ms server TTFB | Caddy/app request duration | not yet recorded |
| S5 | API availability (monthly) | ≥ 99% of `/health` checks green | uptime probe (external once domain live) | not yet recorded |
| S6 | WS interview stability | ≥ 99% of started interviews reach completion or explicit expiry without server-caused disconnect | interview state transitions in DB | not yet recorded |
| S7 | Queue health | pending task age p95 < 5 min; dead-letter rate < 0.5% | asynq queue stats | not yet recorded |
| S8 | Evaluation freshness | report available ≤ 60 s after last answer (inline or retry) | evaluation timestamps | not yet recorded |

## Error-budget policy (beta)

- SLO breaches for 7 consecutive days on S1/S6 → pause feature work, stabilize first.
- Availability budget consumed by an LLM-provider outage is annotated, not counted against product error budget.

## Instrumentation gaps

- Latency histograms per flow need Prometheus counters/histograms wired where marked "not yet recorded".
- External uptime probe starts when beta gate #9 (domain + TLS) lands.
