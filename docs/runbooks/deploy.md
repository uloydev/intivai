# Runbook — Deploy (beta / production)

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM

## Preconditions

- `.env.prod` present on the VPS with every secret filled (compose fails fast on missing values).
- CI green on `main`: `make check` (incl. OpenAPI drift guard), integration tests, FE build/vitest, smoke.
- Deploying a **tagged release** (beta gate #14): tag exists, image pushed to ghcr.

## Standard deploy

```bash
# on the VPS
cd /opt/intivai
git fetch --tags && git checkout <TAG>
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d migrate
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d app
```

CI performs the same via SSH on pushes to `main`.

## Post-deploy verification (all must pass)

1. `curl -fsS https://<domain>/health` → 200; `GET /ready` → 200.
2. `make smoke BASE=https://<domain>` (with `CV_PDF` + LLM key) end-to-end green.
3. Sentry: no new error spike in the following 30 min.
4. Spot-check one candidate interview loop from the recruiter UI.

## Schema migrations

Migrations run as a separate one-shot service (`migrate`) before app start;
forward-only (see rollback runbook for the policy). Never edit an applied
migration — add `NNN_<context>_<what>.up/down.sql`.

## Failure handling

If `/ready` flaps or smoke fails: roll back (`rollback.md`) before debugging
on prod. Preserve logs first: `docker compose logs app > /tmp/app-$(date +%s).log`.
