# P4 Plan — Beta Launch Build (P4a + Candidate/Recruiter FE + P6a)

Scope: re-scoped MVP (Phases doc): evaluation core + recruiter visibility
(P4a), candidate chat UI (P3 FE deliverable), beta ops (P6a). Exit =
**Beta Gate** (M3_Plan.md, 14 items). Plan owner: EM.

---

## 1. Workstream A — Backend P4a (Go, TDD)

### A1. Evaluation domain — `internal/evaluation/domain/`
Canonical schema (Phases doc, single source of truth):

```json
{
  "overall_score": 78,
  "dimensions": {
    "technical": {"score": 82, "weight": 0.4},
    "communication": {"score": 75, "weight": 0.2},
    "problem_solving": {"score": 80, "weight": 0.25},
    "culture_fit": {"score": 70, "weight": 0.15}
  },
  "per_question": [{"question_idx": 1, "score": 85, "rationale": "...",
    "strengths": [], "weaknesses": []}],
  "strengths": [], "weaknesses": [],
  "recommendation": "proceed"
}
```

- `internal/evaluation/domain/report.go`: `Report` struct (json tags snake_case),
  `Evaluate(transcript, dimensions)` — aggregation math: weighted overall =
  Σ dimension.score × weight; per_question scores from LLM, dimensions rolled
  up from per_question by mapping question.category → dimension
- `internal/evaluation/domain/report_test.go` FIRST (pure math):
  weights sum to 1.0 enforced; empty transcript → zeroed report, no panic;
  single answer; very long interview (50+ Q&A); category→dimension mapping
- Persistence: `evaluation JSONB` column on `interviews` (exists, migration
  001) — repo round-trip test (NULL + valid + malformed JSON)

### A2. Evaluation LLM — `internal/evaluation/infrastructure/llm/`
- Prompt: system = evaluator persona + schema contract (JSON only), user =
  transcript (questions + answers, windowed to last ~30 pairs for budget)
- `llm.Client.StructuredOutput` with `Schema: Report{}` (json_object mode,
  already the shared port — contexts never define their own LLMProvider)
- Mock provider test: deterministic report JSON via httptest; malformed LLM
  output → domain validation rejects → fallback to empty report + error flag
- Runs WHEN: interview completes (last answer, `next == nil`). **Decision A:
  inline in the WS stream goroutine** (candidate waits ≤5s for the
  `evaluation` frame) with 20s timeout; on failure send
  `{"type":"evaluation","status":"pending"}` and enqueue asynq retry task
  `evaluate_interview` (worker recomputes + persists; idempotent: skip if
  `evaluation` already non-null)
- WS change: `streamAndRespond` fills `EvaluationMessage.Scores` from the
  computed report (chat_handler.go:378 currently sends empty map)

### A3. Recruiter endpoints — `internal/evaluation/api/`
- `GET /interviews/:id` (authed, admin/recruiter): interview row + questions +
  answers + status + context_version + evaluation + candidate summary
- `GET /candidates/:id/report` (authed): candidate + latest interview report
  JSON (PDF later, P4b)
- `httpapi.Error` mapping: not found → 404, wrong org → 403 (FORBIDDEN)
- Handler tests: status + DTO shape (OpenAPI update in same change);
  integration: create → complete → eval persisted → endpoints return it

### A4. Consent — `internal/interview/`
- `POST /candidate/interviews/:id/consent` (public, invitation token in body):
  validates token (existing `Validate`), sets `consent_given=true` on the
  interview row (column exists)
- Chat start (`StartInterview`) REQUIRES consent_given=true → else error frame
  `CONSENT_REQUIRED` before any question
- Integration test: consent → start OK; no consent → rejected; token mismatch
  → 403; already consented → idempotent

### A5. Invite URL — FE + thin BE
- Share URL shape: `<origin>/invite/:interviewID?t=<invitation_token>`
- No new BE endpoint needed for resolve (ticket endpoint already takes
  interview_id + invitation_token); chat page exchanges token → ws_ticket on
  load. Validate: expired/revoked token shows friendly error page

### A6. Tests (workers/repos per AGENTS)
- Evaluation worker failure path: LLM down → retry → terminal `failed` state
  visible via `GET /interviews/:id`; idempotency on replay (no double LLM)

---

## 2. Workstream B — Frontend (new, `frontend/`)

### Stack (decision B1 — proposed)
- React 18 + Vite + TypeScript, Tailwind + shadcn/ui (design tokens from
  design-system/MASTER.md if present), `@phosphor-icons/react` (project
  convention), gorilla-compatible WS via native `WebSocket`
- Vite dev proxy → `http://localhost:8081`; `INTIVAI_ALLOWED_ORIGINS` must
  include the FE origin (CSWSH — already enforced server-side)

### Pages (routes)
| Route | Purpose | Depends on |
|---|---|---|
| `/login` | recruiter auth (JWT) | API |
| `/jobs` | list + create job | `POST /jobs`, `GET /jobs` |
| `/cvs` | upload CV, status poll (parsed/extracted/failed) | `POST /cvs`, `GET /cvs/:id` |
| `/candidates` | list, screening score, pass/fail | `GET /screenings` |
| `/interviews` | create interview (from passed app), invite link copy | `POST /interviews` |
| `/interviews/:id` | result view: answers + report | `GET /interviews/:id` (A3) |
| `/invite/:interviewID?t=` | candidate entry: consent checkbox → ws chat | consent endpoint (A4) + ticket |
| `/chat/:interviewID` | candidate chat: start, answer, stream tokens, interrupt, resume, evaluation frame | WS protocol (exists) |

### Candidate chat client spec
- Connect: `ws://…/candidate/interviews/:id/chat` with `Authorization: Bearer <ws_ticket>`
- Frames: render `interview.start`, `question`, `token` (streaming append),
  `response`, `evaluation` (show report summary + recommendation), `error`
- Actions: answer submit, interrupt button, ping keepalive (browser native
  WS auto-pongs server pings), resume on reconnect (re-use session_id)
- Consent gate: checkbox before WS connect; POST consent first
- Reduced-motion + basic a11y; mobile-first (candidate likely phones)

### FE tests
- Chat client: ws-mock harness in Vitest (frames → state); pages: component
  tests; E2E: Playwright happy path (login → job → cv → interview → invite →
  chat) — the P3 "browser end-to-end" criterion

---

## 3. Workstream C — P6a Beta Ops

| Item | Approach |
|---|---|
| Deploy | `docker-compose.prod.yml` overlay (no host port conflicts, restart: always, resource limits); push pipeline: GH Actions build → `ghcr.io` → SSH deploy `compose pull && up -d` from tagged release |
| TLS | Caddy sidecar container (auto cert, reverse proxy :443 → app:8080) |
| Backups | Cron (host) or sidecar: `pg_dump` → `mc mirror` MinIO (bucket `intivai-backups`); retention 14d; restore script `scripts/restore.sh` + documented monthly restore test (Beta Gate #10) |
| Alerting | Sentry Go SDK (DSN env, no-op without DSN) — error capture; health/ready already present |
| Env mgmt | `.env.prod` (never committed), secrets checklist: JWT secret, DB URLs, DeepSeek key, MinIO keys |
| Fresh-volume boot | `make dev` from clean volume after all migrations 001–007 + A4 consent (no new migration expected; verify) |

**DECISION D5 (2026-08-10): backups target MinIO only — no offsite copy.**
Deviation from docs ("1 server + 0 backups = losing everything"). Rationale:
beta scale, cost control. Risk: server loss = all data lost. Mitigation
recorded: add `rclone sync` → Backblaze B2 (or S3) no later than first paying
customer / 2026-09-30, whichever comes first (carryover item — tracked in
M3_Plan).

---

## 4. Execution order + estimates (solo)

| Step | Work | Est. |
|---|---|---|
| 1 | A1 evaluation domain + aggregation tests (RED→GREEN) | 0.5 d |
| 2 | A2 LLM evaluator + mock tests + worker + WS frame real scores | 1 d |
| 3 | A3 endpoints + OpenAPI + integration tests | 0.5 d |
| 4 | A4 consent + A5 invite URL (thin) + integration tests | 0.5 d |
| 5 | FE scaffold (Vite+Tailwind+shadcn, design tokens, auth) | 1 d |
| 6 | FE recruiter pages (jobs/cvs/candidates/interviews/result) | 2 d |
| 7 | FE candidate chat + consent + invite + a11y/mobile | 2 d |
| 8 | FE E2E (Playwright happy path) + CSWSH origin config | 1 d |
| 9 | P6a: prod overlay, Caddy, GH Actions deploy, backups+restore | 1.5 d |
| 10 | Beta Gate run: all 14 items, fixes, tagged release push | 1 d |
| | **Total** | **~11 d** |

Blocking order: 1–4 (backend) → 5–8 (FE; 6–7 parallelizable after 5) → 9 can
start after 3. Live DeepSeek verification + fresh-volume boot run during 2 and 10.

---

## 5. Decisions to confirm (before/at build start)

| # | Decision | Proposal | Impact if different |
|---|---|---|---|
| D1 | FE stack | React+Vite+TS+Tailwind+shadcn (D1 in §2) | Alternative: Next.js (SSR SEO, heavier); plain Vite+vanilla (faster, less maintainable) |
| D2 | Evaluation timing | Inline in WS (≤5s wait) + asynq retry | Pure async: candidate leaves, evaluation appears later — worse beta demo |
| D3 | Consent mechanism | REST endpoint pre-WS + gate at start | WS frame instead — weaker audit trail, harder to test |
| D4 | Hosting | Single VPS (docs: 1 server) + Caddy + ghcr | Managed (Render/Railway) — faster but different compose path |
| D5 | Backup target | **DECIDED 2026-08-10: MinIO only** (offsite B2 deferred — see §3 decision block; add no later than first paying customer / 2026-09-30) | Offsite copy lost until then — risk accepted + tracked |

## 6. Gate

Beta starts when M3_Plan Beta Gate items 1–13 checked + 14 done. No partial
beta. Carryover: new items tracked in M3_Plan (P4a tasks scheduled 08-11 → 08-22).
