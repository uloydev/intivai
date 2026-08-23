# Intivai — AI Interview Platform

AI-powered screening interviews: recruiters upload CVs, the system scores
candidates against jobs, and candidates take a real-time chat interview with
an AI interviewer. Post-interview, a structured evaluation report lands back
with the recruiter.

**Status:** MVP backend (P1–P3) + beta scope (P4a + FE + P6a) complete.
Tracking the beta gate: `docs/engineering/beta-gate.md`.

---

## Quickstart (development)

Prereqs: Docker + Docker Compose v2 (compose spec), Go 1.26, Node 22, Python 3 (smoke/OCR).

```bash
# 1. Backend stack (fresh redis each run; postgres/minio volumes persist)
cd backend && make dev

# 2. Seed a LLM API key for the full pipeline (extract/evaluation)
#    (make dev already loads it from .env if present)

# 3. Quick Demo Seeder (optional)
make seed           # seeds demo org (admin@demo.io / password123) + jobs + contexts

# 4. Backend gates
make check          # gofmt + golangci-lint + vet + build + unit tests
make test-integration-dev   # integration tests vs the running stack
make coverage       # per-package coverage floors
make smoke          # full E2E API scenario (real LLM; CV_PDF=/tmp/kilo/cv.pdf)
make load-ws        # 100-concurrent WS load check
make load-k6        # k6 REST load test (100 concurrent users)

# 5. Frontend (separate terminal)
cd frontend
npm ci
npm run dev         # http://localhost:5173 (proxies /api + WS to :8081)
npm run build && npx vitest run   # gates
npx playwright test # E2E happy path (needs stack + LLM API key)
```

Registration is self-serve: `/register` creates an org + admin, then the
recruiter loop is jobs → CV upload → candidates → interview → invite link →
candidate chat / voice interview (`/voice/:id`) → evaluation.

## Architecture

```
Browser (React SPA) ──► Caddy (prod: TLS + static + /api proxy)
        │                        │
        ▼                        ▼
   WebSocket ──────────► Go monolith (Fiber + asynq workers)
                            │
        ┌─────────┬─────────┼──────────┬───────────┐
        ▼         ▼         ▼          ▼           ▼
    PostgreSQL   Redis    MinIO    SumoPod  (memory: SQLite dev /
     (+pgvector,  (queue,   (CVs,    (extract, pgvector bank prod)
     RLS FORCE)  rate     contexts) chat,
                 limits)            evaluation)
```

- **Backend**: modular monolith, DDD + hexagonal per context (IAM, job, cv,
  screening, context, interview, evaluation, memory, llm). Details:
  `docs/engineering/architecture.md`.
- **Multi-tenancy**: RLS `FORCE` on all org tables; app connects as the
  least-privilege `intivai_app` role; pre-auth lookups via SECURITY DEFINER
  functions owned by a BYPASSRLS role. Migrations run separately
  (`-migrate-only` / compose `migrate` service).
- **Frontend**: React + Vite + TS + Tailwind v4 + shadcn/ui; tokens in
  `design-system/intivai/MASTER.md` (mapped to shadcn vars); candidate chat
  is a typed WS client with reconnect/resume.
- **API contract**: `api/openapi.yaml` (DTOs mirror it; FE types in
  `frontend/src/types/api.ts`).

## Documentation index

Full index with status + freshness: **`docs/README.md`**. Landmarks:

| Doc | Contents |
|-----|----------|
| `docs/product/prd.md` | PRD — goals, non-goals, epics, NFRs |
| `docs/product/pricing.md` | Pricing (single source) |
| `docs/engineering/architecture.md` | Code structure, layer rules |
| `docs/engineering/design-decisions.md` | Design record + implementation sync |
| `docs/engineering/roadmap.md` | Phase plan (P0–P6), testing criteria |
| `docs/engineering/beta-gate.md` | Beta gate status |
| `docs/engineering/schemas.md` | Evaluation schema, WS frames, limits |
| `docs/engineering/threat-model.md` · `slos.md` | Security model · SLOs |
| `docs/adr/` | Decision records (0001–0007 + template) |
| `docs/FINDINGS.md` | Single findings/remediation ledger |
| `docs/runbooks/` | Deploy, rollback, restore drill, LLM outage |
| `docs/compliance/` | Access control, retention, IR, SOC 2 readiness |
| `api/openapi.yaml` | HTTP + WS protocol contract (single spec) |
| `design-system/intivai/` | Design tokens, shadcn mapping, page overrides |
| `AGENTS.md` | Engineering workflow: commands, TDD, conventions |

## Deploying (beta / production)

Single VPS per docs. Pipeline: CI builds the backend image (ghcr) + ships
`frontend/dist`; the server runs compose with `.env.prod`:

```bash
# server (one-time)
git clone <repo> /opt/intivai && cd /opt/intivai
cp .env.prod.example .env.prod   # fill every value (see comments)
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d

# nightly backup (cron)
0 2 * * * /opt/intivai/scripts/backup.sh >> /var/log/intivai-backup.log 2>&1

# restore (test monthly)
scripts/restore.sh <dump-file>
```

Deployments: push to `main` → CI builds/pushes ghcr + ships FE + SSH
`compose pull && up -d migrate && up -d app` (rollback = redeploy previous
TAG). Secrets: `.env.prod` (gitignored), GH Actions secrets
(`PROD_HOST/PROD_USER/PROD_SSH_KEY`).

> Pre-deploy checklist + beta gate: `docs/engineering/beta-gate.md`. Known
> deviations (D5): backups target MinIO only — offsite (B2/S3) no later than
> first paying customer / 2026-09-30.

## Testing

- **Backend**: unit (domain, pure) + integration (env-gated, real RLS via
  `TEST_DATABASE_URL`); coverage floors (domain ≥70, others ≥50)
- **FE**: Vitest (lib/api, lib/ws) + Playwright E2E (full candidate journey
  with real LLM; step-logged)
- **CI**: backend (fmt/lint/vet/build/test/race), integration (postgres +
  redis + minio services), coverage gate, FE (build + vitest), smoke,
  deploy-on-main

## License

Private / internal.
