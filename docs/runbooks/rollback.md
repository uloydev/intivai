# Runbook — Rollback

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM

## Policy

- **Migrations are forward-only.** A rollback redeploys the previous *image tag*; it does NOT run `.down.sql` in production. New code must therefore be compatible with the previous migration version (expand/contract pattern: add columns nullable, backfill, then remove later).
- Keep the last N release tags available on ghcr (do not prune; documented deviation tracked in FINDINGS A6/A9 follow-ups).

## Procedure

```bash
cd /opt/intivai
git fetch --tags
git checkout <PREVIOUS_TAG>          # last known-good tag
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d app
```

1. Confirm image digest matches the tag: `docker compose ... images | grep app`.
2. Run post-deploy verification from `deploy.md` (health + smoke).
3. If the bad release wrote data under a NEW schema contract (e.g. rows with new required columns), decide:
   - tolerate (forward-compatible writes) → leave data, fix forward;
   - intolerable → restore from backup (`restore-drill.md`) and accept RPO ≤ 24h.

## When rollback is not enough

- Data corruption / breach → incident response (`docs/compliance/incident_response_plan.md`) + `restore-drill.md`.
- Bad migration already applied → write a new forward-fixing migration; never edit history.
