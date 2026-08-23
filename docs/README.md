# Documentation Index

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Every doc in the repo tree, its role, and freshness. Rule: **one fact, one
> home** — if two docs state the same fact, one must link to the other.

## Root

| Path | What | Status | Last-reviewed |
|---|---|---|---|
| `README.md` | What + quickstart + deploy summary | current | 2026-08-24 |
| `AGENTS.md` | Engineering workflow, commands, conventions | current | 2026-08-19 |
| `CONTEXT.md` | Ubiquitous language | current | 2026-08-10 |
| `CHANGELOG.md` | Tag → notable changes | current | 2026-08-24 |

## docs/

| Path | What | Status | Last-reviewed |
|---|---|---|---|
| `docs/README.md` | This index | current | 2026-08-24 |
| `docs/FINDINGS.md` | Single remediation ledger (all reviews feed this) | current | 2026-08-24 |

### product/

| Path | What | Status | Last-reviewed |
|---|---|---|---|
| `product/prd.md` | PRD: goals, non-goals, epics, NFRs, validation plan | current | 2026-08-24 |
| `product/pricing.md` | Pricing — single source of truth | current | 2026-08-24 |
| `product/demo-seed.md` | Demo seed guide + fixtures (dev-only credentials) | current | 2026-08-24 |

### engineering/

| Path | What | Status | Last-reviewed |
|---|---|---|---|
| `engineering/architecture.md` | Code structure, layer rules, deviations | current | 2026-08-24 |
| `engineering/design-decisions.md` | Design record (research + implementation sync) | current | 2026-08-24 |
| `engineering/roadmap.md` | Phase plan P0–P6, testing criteria | current | 2026-08-24 |
| `engineering/beta-gate.md` | Beta gate status — single home for launch readiness | current | 2026-08-24 |
| `engineering/schemas.md` | Evaluation schema, WS frames, limits table | current | 2026-08-24 |
| `engineering/threat-model.md` | STRIDE-lite per surface | current | 2026-08-24 |
| `engineering/slos.md` | SLOs + error-budget policy | current | 2026-08-24 |
| `engineering/test-matrices/sit-matrix.md` | SIT cases (recovery stub) | stub | 2026-08-24 |
| `engineering/test-matrices/uat-matrix.md` | UAT scenarios (recovery stub) | stub | 2026-08-24 |

### adr/

| Path | Decision | Status |
|---|---|---|
| `adr/0001-application-status-vs-stage.md` | Two-dimensional application state machine | accepted |
| `adr/0002-sandbox-sidecar.md` | Sandbox execution via gRPC sidecar | accepted |
| `adr/0003-speech-stack.md` | Whisper STT + Edge/Piper TTS | accepted (design) |
| `adr/0004-evaluation-proctoring-data-model.md` | Canonical reports schema; LLM never sets final score | accepted |
| `adr/0005-advisory-only-proctoring.md` | Proctoring advisory-only posture | accepted |
| `adr/0006-voice-scope-demo-only.md` | Voice = demo only, out of beta critical path | accepted |
| `adr/0007-backup-offsite-deferral.md` | On-server backups until customer / 2026-09-30 | accepted |
| `adr/template.md` | ADR template | — |

### plans/, reviews/, compliance/, runbooks/

| Path | What | Status |
|---|---|---|
| `plans/active/docs-reorganization-plan.md` | The executed docs reorg plan | executed 2026-08-24 |
| `plans/archive/*` | Closed plans (m3, p4, merged-action) — frozen | archived |
| `reviews/2026-08-19-four-lens.md` · `reviews/2026-08-22-code.md` · `reviews/2026-08-22-competitive.md` | Review snapshots — append-only | snapshots |
| `compliance/access_control_policy.md` · `data_retention_policy.md` · `incident_response_plan.md` · `soc2_readiness.md` | Controls & policies — describe reality | corrected 2026-08-24 |
| `runbooks/deploy.md` · `rollback.md` · `restore-drill.md` · `llm-outage.md` | Operational procedures | current |

## Governance rules

1. New review findings append to `docs/FINDINGS.md` — never a new plan file.
2. Every scope flip or D-decision becomes an ADR the same day.
3. Compliance/policy docs state reality; unimplemented items are Target-Qx.
4. Update this index (+ `Last-reviewed`) whenever you add or touch a doc.
5. Docs older than 90 days without review get flagged by the quarterly tripwire.
