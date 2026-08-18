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

## 2. Workstream B — Frontend (new, `frontend/`) — DETAILED BUILD PLAN

Design system: `design-system/intivai/MASTER.md` (rev 2, corrected) +
`pages/candidate-interview.md` + `pages/dashboard.md` overrides. Page lookup
order: page file first → Master. shadcn token mapping table lives in Master
(§shadcn/ui Token Mapping) — copy the values into `globals.css` verbatim.

### B0. Scaffold (0.5d) — do FIRST, everything depends on it
```
frontend/
  package.json (react@18, vite, typescript, tailwindcss@3, shadcn/ui,
                @phosphor-icons/react, react-router-dom@6, @tanstack/react-query@5,
                ws-native (browser WebSocket — no dep), vitest, @testing-library/react,
                playwright)
  vite.config.ts        — proxy /api → http://localhost:8081 (ws:true for /chat)
  tsconfig.json         — strict
  tailwind.config.ts    — font-display: Space Grotesk; font-body: DM Sans;
                          colors from CSS vars (dark via .dark class)
  src/
    styles/globals.css  — shadcn theme vars (Master mapping) + tokens (light+dark)
    lib/api.ts          — fetch wrapper: baseURL from VITE_API_BASE, bearer token
                          injection, error normalization ({code,error} → typed ApiError)
    lib/auth.ts         — token storage (localStorage), login/logout, RequireAuth guard
    lib/ws.ts           — chat socket client (typed frames, auto-pong, reconnect+resume)
    types/api.ts        — DTO types mirroring OpenAPI (snake_case)
    components/ui/      — shadcn primitives (button, input, card, dialog, table,
                          skeleton, badge, toast, textarea, select)
    pages/…             — see B2
```
- `npm create vite@latest . -- --template react-ts`; init Tailwind + shadcn
  (`npx shadcn@latest init` — theme values from Master mapping)
- Commit scaffold + design tokens + CI smoke of `npm run build`

### B1. Auth + shell (0.5d)
- `/login` — email/password → POST /auth/login → store token; error surface
  (AUTH_FAILED etc.); redirect to /jobs
- `RequireAuth` route guard: no token → /login; 401 on any call → force logout
- App shell: sidebar (nav: Jobs, CVs, Candidates, Interviews) — Phosphor line
  icons, active = primary; org slug in header; mobile: bottom nav (≥44px)

### B2. Recruiter pages (2.5d) — each with react-query + skeleton/empty/error states
| Page | Route | API | Notes |
|------|-------|-----|-------|
| Jobs list | /jobs | GET /jobs | cards or table; create button |
| Job form (modal) | — | POST /jobs | required: title, description, required_skills, min_experience; PATCH status (active/archived) |
| CVs | /cvs | GET /cvs, POST /cvs (multipart), GET /cvs/:id | upload dropzone; status poll via react-query refetchInterval (2s while parsing/extracting); failed → error_message + retry (POST /cvs/:id/extract) |
| Candidates | /candidates | GET /applications | table: candidate_name/email, job_title, cv_score, passed pill; row → drawer: GET /candidates/:id/report (interviews + evaluations) |
| Interviews | /interviews | POST /interviews, GET /interviews/:id | pick passed application → question_count → create; response shows invitation_token + copy **invite link** (`${origin}/invite/:id?t=:token`) |
| Interview result | /interviews/:id | GET /interviews/:id | transcript (Q/A), status, context_version, evaluation report (scores, dimensions, per_question, strengths/weaknesses, recommendation pill) |

Status pill mapping (Master §Status Pills): parsing/processing → pill-processing;
passed → pill-passed; rejected/failed → pill-rejected; new → pill-neutral.

### B3. Candidate flow (2d)
- `/invite/:interviewID?t=:token` — entry page: role title, question count,
  duration notice, **consent checkbox** (POST /candidate/interviews/:id/consent,
  disabled until accepted), then auto-exchange ticket
  (POST /candidate/interviews/:id/ticket) → /chat/:interviewID
- `/chat/:interviewID` — `lib/ws.ts` client:
  - Connect: Authorization: Bearer ws_ticket
  - Frames: interview.start → header (question x of n); question → render +
    store as current; token → append to streaming bubble; response → finalize;
    evaluation (complete) → result screen (scores + recommendation);
    evaluation (pending) → "report preparing" + poll GET /interviews/:id (as
    recruiter would) — actually candidate: show thank-you, report later
    via invite link re-entry? — show summary from frame only
  - error frame: {code,message} → CONSENT_REQUIRED → redirect to invite page;
    session mismatch → reconnect with resume {session_id}; else show error + retry
  - Actions: answer (Enter send, Shift+Enter newline, disabled while streaming),
    interrupt (cancels stream, next question arrives), ping keepalive (native
    auto-pong for server pings — no code needed, but verify)
  - Reconnect: on ws close → banner (amber) → auto-reconnect with resume frame
    (session_id) → resume from current question; never lose transcript locally
  - Completion: thank-you + "what happens next"; transcript download link
    (GDPR) — local render of the session transcript
  - Mobile-first: 720px column (candidate override), input 48px, touch ≥44px
- `/login` shared with recruiters (no separate candidate account — token flow)

### B4. Quality (1d)
- Vitest: lib/ws.ts frame machine (mock WebSocket), lib/api.ts error mapping,
  auth guard, status pill mapping, invite URL parsing
- Playwright E2E happy path: login → create job → upload CV (fixture PDF) →
  poll to extracted → candidate passed → create interview → copy invite →
  consent → chat (answer + tokens) → evaluation frame → result visible
  (mirrors scripts/smoke.sh; needs stack + INTIVAI_DEEPSEEK_API_KEY)
- a11y + responsive pass (Master checklist), reduced-motion, dark mode toggle
  (recruiter shell only; candidate stays light)

### B5. FE → BE integration notes
- `INTIVAI_ALLOWED_ORIGINS` must include the FE dev origin (5173) + prod
  domain (VITE_API_BASE/origin) — CSWSH rejects browsers without it
- Vite dev proxy: `/api` → 8081 with `ws: true` for the chat socket
- All DTOs snake_case from OpenAPI — types/api.ts generated by hand from
  api/openapi.yaml (keep in sync when endpoints change)

### Definition of done per page
- [ ] Renders from real API (no fixtures), loading/empty/error states present
- [ ] Status codes handled (401 logout, 400 show {error}, 403 forbidden)
- [ ] Master tokens used (no ad-hoc hex), a11y checklist passed
- [ ] Vitest unit + Playwright E2E green for the flow it belongs to

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

**CARRYOVER D6 (2026-08-18, grilling session): Voice Interview (Phase 5)
formally deferred.** Gated demo stays in the tree (WS route + signaling +
TTS-as-audio-frames demo, FE page at `/voice/:id`); the real pipeline (Opus
mic decode, STT loop, Opus-over-RTP, TURN, recording) is out of beta scope.
WHY: no paying customer (phase gate per AI_Interviewer_Phases.md); resume
WHEN: first paying customer requires voice. Sandbox code execution moved to a
gRPC sidecar + per-language containers (ADR-0002) — `make sandbox-images` +
`make dev` build/run it; sandbox fails closed without the sidecar.

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
