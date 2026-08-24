# Docs Reorganization & Governance Plan

**Status:** archived — executed 2026-08-24 (see `docs/README.md` index)
**Created:** 2026-08-24
**Source:** Full documentation audit (28 files, ~8k lines) — findings summarized in the audit conversation of 2026-08-24.
**Goal:** Make the documentation system production-grade: one truth per fact, no contradictions, a governed structure, and tooling that keeps it that way.
**Scope:** Documentation only. No application code changes, except the OpenAPI consolidation task which reconciles two spec files (W1).
**Owner:** EM. Executor: any engineer/agent session; tasks are self-contained.

---

## 0. Ground Rules (apply to every task)

1. **`git mv` only** for relocations — preserves history. Never copy-paste a file to a new path and delete the old one.
2. **One concern per commit**, Conventional Commits (`docs: ...`). Never bundle a move with an edit — do the `git mv` commit first, then edit at the new path.
3. **Never delete content outright.** Anything with open action items migrates row-by-row into `docs/FINDINGS.md`; anything historically interesting moves to `docs/reviews/` or `docs/plans/archive/`. Deletion is reserved for exact duplicates already absorbed elsewhere.
4. **Contradictions resolve by decision, not by picking silently.** Each conflict in W2 has a "Resolution" field. Where a real product decision is required it is marked **DECISION** with a proposed default; the executor applies the default unless the owner overrides.
5. Every moved/edited doc gets a header block: `Status: current|superseded-by:<path>|archived`, `Last-reviewed: <date>`.
6. Gates: after every workstream run the verification commands in §7. Final gate = full checklist in §8.

---

## 1. Target Structure (end state)

```
/
├── README.md                        # root: what + quickstart + links to docs index
├── AGENTS.md                        # unchanged
├── CONTEXT.md                       # unchanged (ubiquitous language)
├── CHANGELOG.md                     # NEW: tag → user-visible changes
├── api/openapi.yaml                 # THE spec (single copy)
├── docs/
│   ├── README.md                    # NEW: index — every doc, status, last-reviewed
│   ├── FINDINGS.md                  # NEW: single remediation ledger (all reviews feed this)
│   ├── product/
│   │   ├── prd.md                   # from PRD_SUMMARY.md, rewritten (W3)
│   │   ├── pricing.md               # NEW: single pricing source (W2)
│   │   └── demo-seed.md             # from EXAMPLES.md (+ dev-only banner)
│   ├── engineering/
│   │   ├── architecture.md          # from AI_Interviewer_Project_Structure.md
│   │   ├── design-decisions.md      # from AI_Interviewer_Research.md
│   │   ├── roadmap.md               # from AI_Interviewer_Phases.md, one numbering scheme
│   │   ├── beta-gate.md             # extracted live gate from M3_Plan.md
│   │   ├── schemas.md               # NEW: canonical evaluation JSON + WS frame table (links to spec)
│   │   ├── threat-model.md          # NEW (W6)
│   │   └── slos.md                  # NEW (W6)
│   ├── adr/                         # 0001–0007 + template.md (W4)
│   ├── plans/
│   │   ├── active/                  # this file + anything genuinely in-flight
│   │   └── archive/                 # closed plans, frozen with status header
│   ├── reviews/                     # dated review snapshots, append-only
│   ├── compliance/                  # existing 4 docs, corrected (W5)
│   └── runbooks/                    # deploy, rollback, restore-drill, llm-outage (W6)
├── design-system/intivai/           # unchanged
└── frontend/README.md               # unchanged (subsystem readme)
```

Root ends with exactly 5 markdown files: README, AGENTS, CONTEXT, CHANGELOG (+ nothing else).

---

## 2. File Disposition Map (every existing doc, exactly once)

| # | Current file | Action | Destination | Notes |
|---|---|---|---|---|
| 1 | `README.md` | keep + update | `/README.md` | Rewrite "Documentation index" section to point at `docs/README.md`; fix "Docker Compose v5" version claim |
| 2 | `AGENTS.md` | keep | — | Add one line to PR checklist: "docs affected? update per docs/README.md" |
| 3 | `CONTEXT.md` | keep | — | — |
| 4 | `AI_Interviewer_Research.md` | git mv + patch | `docs/engineering/design-decisions.md` | Patches listed in T3 |
| 5 | `AI_Interviewer_Phases.md` | git mv + patch | `docs/engineering/roadmap.md` | Patches in T4 |
| 6 | `AI_Interviewer_Project_Structure.md` | git mv | `docs/engineering/architecture.md` | Update tree diagram to new layout; refresh migration count |
| 7 | `PRD_SUMMARY.md` | git mv + rewrite | `docs/product/prd.md` | Rewrite outline in T5 |
| 8 | `EXAMPLES.md` | git mv + patch | `docs/product/demo-seed.md` | Add DEV-ONLY warning banner (demo tokens/passwords); verify `fixtures/cvs/*.pdf` paths exist before moving references |
| 9 | `M3_Plan.md` | split → archive | Beta Gate rows → `docs/engineering/beta-gate.md`; remainder → `docs/plans/archive/m3-plan.md` | Gate items 9–14 still open — must land in an ACTIVE location, not archive |
| 10 | `P4_Plan.md` | archive | `docs/plans/archive/p4-plan.md` | Header: superseded. D5/D6 decisions extracted to ADRs (W4) first |
| 11 | `MERGED_ACTION_PLAN.md` | triage → archive | Unchecked acceptance boxes → `docs/FINDINGS.md` rows; then archive | See T7 |
| 12 | `COMPETITIVE_REVIEW_ACTION_PLAN.md` | delete | — | Fully absorbed by MERGED_ACTION_PLAN; confirm zero unchecked items not already captured in row 11's ledger pass |
| 13 | `UI_UX_REVAMP_PLAN.md` | delete | — | Same condition as row 12 |
| 14 | `REVIEW_2026-08-19.md` | git mv | `docs/reviews/2026-08-19-four-lens.md` | All 🔴/🟡 items → FINDINGS.md with source IDs R19-x |
| 15 | `CODE_REVIEW_2026-08-22.md` | git mv | `docs/reviews/2026-08-22-code.md` | Observations 9–15 (status: Pending) → FINDINGS.md CR22-9..15 |
| 16 | `COMPETITIVE_REVIEW_2026-08-22.md` | git mv | `docs/reviews/2026-08-22-competitive.md` | Product-strategy items (§5, §9) → FINDINGS.md or prd.md backlog section |
| 17 | `FIX_PLAN.md` (36 fixes) | convert → delete | Rows → FINDINGS.md FP-1..36 with real status each | Verify claimed-fixed statuses against code before marking done |
| 18 | `FIXING_PLAN_2026-08-19.md` | convert → delete | Rows → FINDINGS.md F19-P*.* | Same verification rule |
| 19 | `FIXING_PLAN_UIUX_2026-08-19.md` | convert → delete | Rows → FINDINGS.md UX19-* | Same |
| 20 | `frontend/README.md` | keep | — | Update its `../api/openapi.yaml` mention if spec path changes (it does not — stays at `api/openapi.yaml`) |
| 21 | `design-system/**` | keep | — | — |
| 22 | `docs/compliance/*.md` (4) | keep + correct | — | Corrections in W5 |
| 23 | `docs/adr/000{1,2}*.md` | keep | — | Add template.md + backfills in W4 |
| 24 | `backend/api/openapi.yaml` | reconcile → delete | Content merged into `api/openapi.yaml` (URGENT, untracked file) | W1 |

---

## 3. Workstreams

### W1 — OpenAPI single source (URGENT, day 0)

Why first: `backend/api/openapi.yaml` (1750 lines, richer: webhooks CRUD, bulk CV upload, proctoring HTTP fallback, candidate review, `/ready`, WS frames `pong`+`code.result`) is **untracked** — a fresh clone loses it. Root copy (1068 lines, tracked, referenced by all docs) is stale.

| Task | Detail |
|---|---|
| 1.1 | `git add backend/api/openapi.yaml` immediately (safety commit on its own). Commit message: `chore: track backend OpenAPI copy before reconciliation` |
| 1.2 | Diff both copies endpoint-by-endpoint against actual Fiber route registrations in `backend/internal/**/api/` and `cmd/server/main.go`. Produce a three-column list: route exists / in root spec / in backend spec |
| 1.3 | Reconcile into ONE file at `api/openapi.yaml`: union of endpoints, backend-copy summaries where they disagree, complete WS frame description (server→client AND client→server frames incl. `resume`, `pong`, `interrupt`, `code.result`, `audio`) |
| 1.4 | Delete `backend/api/openapi.yaml`. Grep repo for stray references: `grep -rn "backend/api/openapi" . --include="*.md"` |
| 1.5 | CI drift guard (interim, until codegen): script `scripts/check-openapi-drift.sh` — extracts registered paths from Go router setup, compares against `paths:` keys, fails on mismatch. Wire into `make check` |
| 1.6 | Follow-up ticket (not this plan): adopt oapi-codegen/ogen so Go DTOs + `frontend/src/types/api.ts` generate FROM the spec — kills hand-sync permanently |

**Accept:** single yaml tracked in git; drift guard green in CI; zero references to the deleted path.

### W2 — Contradiction reconciliation (day 0–1)

Every conflict gets ONE resolution applied everywhere. Table of record:

| # | Topic | Conflicting claims | Resolution (**DECISION** = owner may override; default applied otherwise) | Touch points |
|---|---|---|---|---|
| C1 | Encryption at rest | PRD NFR: "planned, not implemented" vs soc2_readiness.md: "Implemented" | Compliance docs get a **Status column: Implemented / Partial / Target-Q4-2026**. Encryption-at-rest = Target until pgcrypto+SSE-S3 verified in prod compose | `PRD_SUMMARY.md` §5, `soc2_readiness.md`, `data_retention_policy.md` |
| C2 | Project status | PRD "M1–M4 Complete, 100% Green" vs Beta Gate items 9–14 open | Status statement lives ONLY in `beta-gate.md` + README one-liner. PRD carries no completion claim | prd.md, beta-gate.md, README.md |
| C3 | Pricing | $49/$3-free (Research) vs $29/$10-free (PRD + competitive rec) | **DECISION** — default: PRD numbers ($29 Starter, Free 10/mo, credits $99/500, volume $0.50). Single home `docs/product/pricing.md`; Research/prd link to it | pricing.md, design-decisions.md §6, roadmap.md |
| C4 | Voice scope | Deferred post-MVP (Phases) / pivoted back as MVP (P4 D6) / gated demo shipped | **DECISION** — default: voice = demo-only, NOT MVP-critical-path (matches Phases deferral + REVIEW findings that prod voice is mock). Record as ADR-0006 with history | roadmap.md Phase 5, ADR-0006, prd.md Epic 5 wording |
| C5 | Migration count | 001–007 / 001–009 / 014–017 / 025 refs | Truth = `ls backend/pkg/db/migrations/*.up.sql \| wc -l`. architecture.md states count with "as of" date; other docs never state counts again | architecture.md |
| C6 | FE stack | Research stack table "Next.js + React" | Vite SPA. Fix row | design-decisions.md §4.9 |
| C7 | Idle timeout | "idle 5m" config block vs 3-min read deadline impl note | Keep both but label: configured intent 5m, implemented mechanism = 3-min read deadline (existing impl note is right). One sentence, no ambiguity | design-decisions.md §2 |
| C8 | JWT sessions | access_control_policy: "refresh token rotation", "all JWTs invalidated on deactivation" vs no such infra | Policy rewritten to Implemented vs Target columns; refresh/revocation = Target | `access_control_policy.md` |
| C9 | Upload caps | 10MB CV vs 64KB context (different things, reads like conflict) | One config table in schemas.md listing all caps (CV size, context size, prompt length, token budget) with env var names | schemas.md, design-decisions.md |
| C10 | WS frames | Phases prose set ≠ both yaml sets | Prose deleted; frames live in api/openapi.yaml + summary table in schemas.md | roadmap.md, schemas.md |
| C11 | Test matrices | PRD cites sit_test_matrix.md / uat_test_matrix.md — files don't exist | **DECISION** — default: create stub matrices under `docs/engineering/test-matrices/` seeded from PRD case counts, OR remove references. Default: create stubs (the 58/60 cases reportedly exist somewhere — recover if findable, else mark TBD) | prd.md, new stubs |
| C12 | Numbering schemes | M1–M4 vs P0–P6 vs P4a/P4b/P6a/P6b | Single scheme: Phases P0–P6 with sub-splits (P4a…) kept. M-numbers appear only inside archived plans | roadmap.md header legend |

**Accept:** scripted greps return zero hits for known-conflicting strings:

```bash
# examples (extend per resolution):
grep -rn "Next.js" docs/ README.md            # expect: none (or explicit "rejected Next.js" rationale)
grep -rn "\$49/mo" docs/                      # expect: none outside archive/
grep -rn "Implemented" docs/compliance/       # expect: only where actually true
```

### W3 — Core doc moves + rewrites (day 1–2)

| Task | Detail | Accept |
|---|---|---|
| T1 | `git mv AI_Interviewer_Project_Structure.md docs/engineering/architecture.md`; update internal tree diagram to target layout; migration count per C5; add header block | Tree matches reality; `git log --follow` shows history |
| T2 | `git mv AI_Interviewer_Research.md docs/engineering/design-decisions.md` | History preserved |
| T3 | Patch design-decisions.md: C4 (voice §3 → "gated demo"), C6 (stack row), C7 (timeout), add pointer "schema/config tables live in schemas.md"; strip pasted evaluation JSON + scoring tables → replace with links to schemas.md | No duplicated schema/config blocks remain (grep `"overall_score"` should hit schemas.md + archives only) |
| T4 | `git mv AI_Interviewer_Phases.md docs/engineering/roadmap.md`; apply C10 (delete WS protocol prose), C12 (add legend: M-numbers = old aliases); paste-frame sections become "see api/openapi.yaml"; extract Beta Gate → `docs/engineering/beta-gate.md` (with M3_Plan items 9–14 status verbatim); archive remainder of M3_Plan | roadmap has no frame JSON; beta-gate.md is the only place gate status lives |
| T5 | `git mv PRD_SUMMARY.md docs/product/prd.md` + rewrite: (a) title "AI Technical Interview Platform"; (b) Goals / **Non-goals** ("Not an ATS — integrates via webhooks") section; (c) success metrics made measurable + evidence-linked or removed ("Hallucination-Free", "100% Auditability", "$38,000+" → either cite measurement method or cut); (d) Epics tagged Must/Should/Could/Won't (MVP) per competitive review consensus; (e) mobile-first NFR added; (f) customer-validation section (pilot cohort, criteria); (g) pricing section replaced by link to pricing.md; (h) test-status section replaced by link to beta-gate.md | No unverifiable superlative remains without a citation; non-goals present; grep "Recruitment Platform" = 0 hits |
| T6 | Create `docs/product/pricing.md`: single tier table (per C3 default), credits rules, volume discounts, last-reviewed date. Research §6 + prd.md link here | Exactly one pricing table outside archives |
| T7 | MERGED_ACTION_PLAN triage: walk every unchecked `- [ ]` box; for each, create FINDINGS row (MAP-wX-y) with honest status (done-but-unmarked / open / obsolete); then `git mv` to `docs/plans/archive/merged-action-plan.md` with `Status: archived` header | No unchecked box left un-accounted |
| T8 | Rows 12–13 (`COMPETITIVE_REVIEW_ACTION_PLAN.md`, `UI_UX_REVAMP_PLAN.md`): diff their unchecked boxes against MAP ledger pass from T7; capture deltas; then `git rm` both | Zero unique unchecked items lost |
| T9 | EXAMPLES.md → `docs/product/demo-seed.md`: prepend "**DEV/DEMO ONLY — seeded credentials and tokens must never exist in staging/prod.**"; verify fixture paths (`fixtures/cvs/`, `scripts/seeds/demo/cvs/`, `frontend/generate_cv_pdfs.mjs`) exist, fix if drifted | Banner present; all relative links resolve |
| T10 | Reviews relocation per disposition map rows 14–16 (`git mv`, add `Status: snapshot — append-only` headers). No edits inside snapshots beyond header | Files under docs/reviews/, untouched bodies |
| T11 | Convert FIX_PLAN / FIXING_PLAN_2026-08-19 / FIXING_PLAN_UIUX_2026-08-19 into FINDINGS rows (IDs FP-n, F19-x.y, UX19-x.y). For every row claiming fixed: spot-verify against code (file/line cited) before marking `closed`; unverifiable → `needs-verify`. Then delete the three files | FINDINGS.md covers ≥100% of prior open items; three files gone |

### W4 — ADR program (day 2)

| Task | Detail |
|---|---|
| 4.1 | Create `docs/adr/template.md`: Context / Decision / Alternatives considered / Consequences / Date / Status: proposed / accepted / superseded-by |
| 4.2 | Backfill from existing decisions (each ≤15 lines, cite original doc section as provenance): **ADR-0003** speech architecture (Whisper STT + Edge/Kokoro TTS, Research §3); **ADR-0004** evaluation + proctoring data model (canonical reports schema, PRD ref now resolves); **ADR-0005** advisory-only proctoring posture (Research sync §6 row); **ADR-0006** voice scope history: deferred → pivoted (D6) → final state per C4; **ADR-0007** backup offsite deferral D5 incl. hard date 2026-09-30 |
| 4.3 | Rule recorded in docs/README.md: every future D-numbered decision or scope flip becomes an ADR same-day; plans may reference ADRs but never contain standalone decision blocks |

**Accept:** PRD reference "ADR-0001–0004" resolves to real files; D5/D6 have permanent homes.

### W5 — Compliance truth pass (day 2–3)

For each of the 4 compliance docs: add Status column per control/claim (Implemented / Partial / Target-<quarter>); align with C1/C8; cross-check every "Implemented" claim against code reality (e.g., MinIO SSE-S3 configured? purge jobs exist? cascade erase function exists?).

Rules: never describe intent in present tense; retention schedule must match what jobs actually do; backup section states current offsite gap + ADR-0007 link.

**Accept:** an auditor reading compliance docs finds zero claims contradicted elsewhere in repo; each claim traceable to a file/function name.

### W6 — Missing doc types (day 3–4)

| Doc | Minimum viable content |
|---|---|
| `CHANGELOG.md` | Backfill from git tags: tag → notable changes. Going forward: appended at release |
| `docs/runbooks/deploy.md` | Current deploy flow from README §Deploying, expanded: pre-deploy checks, compose commands, verify steps (`make smoke` against prod URL) |
| `docs/runbooks/rollback.md` | "redeploy previous TAG" expanded: how to list tags, pin image, DB migration down-policy (state it: forward-only? document decision) |
| `docs/runbooks/restore-drill.md` | Monthly restore test procedure: restore.sh into intivai_restore, role bootstrap check, promote steps, success criteria; links ADR-0007 + incident plan |
| `docs/runbooks/llm-outage.md` | Cost rails during provider outage: retry budget, MaxRetry policy, per-org token cap status, kill-switch procedure |
| `docs/engineering/threat-model.md` | STRIDE-lite per surface: public careers board (spam/LLM-spend), candidate portal (OTP abuse), WS chat (CSWSH/ticket theft), sandbox (escape/SSRF-class), webhooks (SSRF/replay), LLM (prompt injection, cost). Each threat: mitigation status + link to code/tests. Source material: CODE_REVIEW #2/#15, REVIEW security items, FIX_PLAN SSRF fixes |
| `docs/engineering/slos.md` | Take NFR table targets → SLO form: metric definition, measurement source (Prometheus metric names), p95/p99 targets, error-budget policy placeholder. Record actuals from load-ws/load-k6 runs where available |
| `docs/README.md` | Index table: path · what · status · last-reviewed · owner. Include EVERY md file in docs/ (script-assisted: `find docs -name "*.md"`) |

### W7 — Governance tooling (day 4)

| Task | Detail |
|---|---|
| 7.1 | `.markdownlint.yml` (MD013 line-length off; MD024 sibling-duplicates allowed; rest defaults) + markdownlint-cli2 step in `make fe-build`-style FE gate or separate `make lint-docs` wired into `make check` chain |
| 7.2 | Link checker: lychee (CI job) scoped to `*.md` excluding `node_modules/`, `docs/plans/archive/`, `docs/reviews/`; fails build on dead internal links — catches the ADR-0004/sit-matrix class permanently |
| 7.3 | `CODEOWNERS`: `docs/ @em` (adjust handle); `docs/compliance/ @em` |
| 7.4 | AGENTS.md additions: (a) Definition-of-Done item "docs affected? updated per docs/README.md index"; (b) review protocol: "new review findings append to docs/FINDINGS.md — never create a new plan file"; (c) language policy: English-only repo docs |
| 7.5 | Staleness tripwire: quarterly cron/issue — any doc with `Last-reviewed` older than 90 days gets a review issue |

---

## 4. FINDINGS.md Ledger Design

Columns: `ID | Source (file+item) | Severity | Area | Description (≤2 lines) | Status (open/in-progress/closed/obsolete/needs-verify) | Closed-date | Notes`.

Seeding order (W3-T11, T10): F19 → UX19 → R19 → FP → CR22 → MAP leftovers → COMP22 strategy items. Dedupe pass across sources before writing (same finding reported by multiple reviews → one row, multiple Source entries).

Rule of use: any future review/audit appends rows here. Plans may group rows into phases but the ledger is the only status authority.

---

## 5. Sequencing

```
Day 0   W1 (spec rescue + merge)          ← data-loss risk, do first
Day 0-1 W2 (contradiction resolutions)    ← needs DECISION confirmations up front
Day 1-2 W3 (moves, rewrites, ledger seed) ← depends on W2 answers for patched text
Day 2   W4 (ADRs)                         ← C4/C5 outcomes from W2 feed ADR-0006/0007
Day 2-3 W5 (compliance)                   ← independent of W3 except C1/C8 inputs
Day 3-4 W6 (new docs)                     ← needs final paths from W3
Day 4   W7 (tooling + gates)              ← last; lints everything landed above
```

Parallelizable: W5 ∥ W3; W6 ∥ W4.

Estimate: ~3–4 focused days total (solo), ~2 days with two executors.

---

## 6. DECISIONS Needed From Owner (answer before/at W2 start)

| # | Question | Proposed default |
|---|---|---|
| D-1 | Final pricing tiers? | $29 Starter / Free 10 interviews / $99-500 credits / $0.50 volume |
| D-2 | Voice final scope for beta? | Demo-only, out of critical path |
| D-3 | sit/uat test matrices — recover from somewhere, or recreate as stubs? | Recreate stubs, mark recovered cases TBD |
| D-4 | DB migrations forward-only or down-migrations supported? (needed for rollback runbook) | Forward-only documented explicitly |
| D-5 | CODEOWNERS handle(s)? | Repo owner account |

---

## 7. Verification Commands (run per workstream gate + final)

```bash
# Structure
ls *.md                                  # expect: README.md AGENTS.md CONTEXT.md CHANGELOG.md only
test ! -f backend/api/openapi.yaml       # expect pass
test -f api/openapi.yaml && test -f docs/README.md && test -f docs/FINDINGS.md

# History preserved for major moves
git log --follow --oneline -- docs/engineering/design-decisions.md | tail -n 1   # reaches original creation
git log --follow --oneline -- docs/product/prd.md | tail -n 1

# Contradiction sweeps (C-table)
grep -rniE "next\.js" README.md docs/ --exclude-dir=archive --exclude-dir=reviews
grep -rn '\$49/mo' . --include="*.md" | grep -v archive
grep -rn "Recruitment Platform" README.md docs/ --exclude-dir=reviews
grep -rn "sit_test_matrix\|uat_test_matrix" docs/product/prd.md   # must resolve to real paths or be gone

# Duplication sweeps
grep -rln '"overall_score"' docs/ | sort           # expect: engineering/schemas.md only (+archive)
grep -rln 'interview.start' docs/ *.md             # expect: schemas.md, maybe roadmap link-text only

# Tooling (after W7)
markdownlint-cli2 "docs/**/*.md" "*.md"
lychee --offline --exclude '(^file:|node_modules|archive/|reviews/)' .
bash scripts/check-openapi-drift.sh          # exit 0
```

---

## 8. Definition of Done

- [x] Root contains exactly README / AGENTS / CONTEXT / CHANGELOG
- [x] One openapi.yaml, tracked, drift-guarded in CI
- [x] docs/README.md indexes every docs file with status + last-reviewed
- [x] docs/FINDINGS.md accounts for every open item from all five prior review/fix documents (deduped, statuses verified)
- [x] Zero unresolved rows in the W2 contradiction table (each marked resolved-with-decision-ID)
- [x] Compliance docs carry truthful Implemented/Partial/Target statuses matching code
- [x] ADRs 0001–0007 + template exist; PRD ADR references resolve
- [x] PRD rewritten: non-goals, measurable claims, single pricing link, no fake matrix references
- [x] Runbooks (deploy/rollback/restore-drill/llm-outage), threat-model, slos, schemas, changelog exist
- [x] markdownlint + lychee green in CI; AGENTS.md governance rules recorded
- [x] `make check` still green (doc changes must not break gates; scripts/check-openapi-drift.sh added to it)

> DoD ticked 2026-08-24 by the R24 deep review (H6): plan verified executed via
> spot-checks — structure, spec consolidation, ledger seeding, ADR backfills,
> compliance truth-pass all present in repo. Known residue: README index
> omissions fixed same day; drift-guard helper-route blind spot tracked as E4.
