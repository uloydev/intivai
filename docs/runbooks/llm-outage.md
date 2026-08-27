# Runbook — LLM Outage & Cost Control

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM

## What depends on the LLM

| Flow | Failure behavior today |
|---|---|
| CV extraction (`cv:extract`) | Retries with backoff; terminal `failed_extract` visible in UI; manual retry endpoint |
| Interview chat responses | Stream error → next question still dispatched (interview never strands); candidate sees degraded experience |
| Evaluation | Inline attempt fails → `evaluation{status:"pending"}` + asynq retry worker recomputes |
| Rubric generation | `MaxRetry(5)` then dead-letter |

## Outage playbook

1. **Confirm scope**: provider 429/5xx vs auth failure vs network. Check Sentry + provider status.
2. **Stop the spend**: if retries are amplifying cost or the outage is long, scale workers to zero —
   `docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d --scale app=0 app`
   is too broad (kills HTTP); prefer revoking the LLM key so calls fail fast instead of burning retries (worker concurrency is fixed at 10, main.go:526 — no runtime pause switch today).
3. **Communicate**: interviews already started degrade gracefully; new extractions show honest `failed_extract` states. Post a status note; no fabricated data is ever shown.
4. **Recover**: after the provider recovers, re-enable, then replay failed tasks:
   - `POST /cvs/{id}/extract` for stuck extractions;
   - evaluation workers self-heal via asynq retries (verify queue drains: pending count → 0).
5. **Post-mortem** if user-facing > 30 min: blameless write-up linked from FINDINGS.

## Cost rails (state of play)

- Explicit `MaxRetry(5)` on LLM-invoking tasks — prevents the default-25 retry storm (verified on rubric/public paths; sweep other tasks: FINDINGS D-track).
- Token ledger at the LLM port + per-org daily cap — **not yet implemented** (threat model T7); without it, "what does this customer cost me" is unanswerable and an outage can multiply spend.
- Alerting: queue-depth / worker-death / error-rate alerts are beta-gate #11 scope.

## Escalation

Solo dev during beta: founder is on-call (`incident_response_plan.md` reality note).
