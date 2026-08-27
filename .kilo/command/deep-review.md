---
description: "Deep review playbook - 4-agent audit, findings ledger, decision register, fix plan"
---

Run the Intivai deep-review playbook end to end. Follow every step in order; do not skip gates.

## 0. Preconditions

- Working tree MUST be clean (`git status --porcelain` empty except excluded tooling dirs). Land or stash drift FIRST — dirty baselines poison honest commits.
- Read `docs/FINDINGS.md`, `docs/engineering/beta-gate.md`, and `docs/engineering/roadmap.md` before spawning agents.

## 1. Parallel audit (4 agents)

Spawn 4 explore agents concurrently with disjoint scopes:
1. **docs-sync**: docs vs code reality — every present-tense claim verified against an implementation file.
2. **ops/CI/security**: workflows, Dockerfiles, compose files, secrets handling, backup/restore, rate limits, auth surfaces.
3. **Go backend**: domain rules, transaction boundaries, RLS usage, error taxonomy, worker idempotency, race-prone seams.
4. **Frontend**: contract mismatches with API, trust/honesty of rendered claims, token/secret handling, dead code.

Each agent returns findings as ledger rows: severity, evidence (file:line), proposed fix. Done = all four reports returned.

## 2. Ledger absorption

Append all rows to `docs/FINDINGS.md` under the current review tag (e.g. `R25`). Reconcile stale open rows: close with evidence or mark needs-verify. One fact, one home — no new plan files for findings.

Done = every finding has a row; no duplicate meanings across docs.

## 3. Decision register BEFORE any code

For every finding needing a product/architecture choice: grill the owner with the question tool, options-first with recommendations. Record confirmed choices as DA, DB, … in the fix plan's decision register. Scope flips become ADRs same day.

Done = register complete; zero undecided items entering execution.

## 4. Fix plan

Write `docs/plans/active/<name>-plan-<date>.md`: decision register, batches ordered smallest-risk-first, per batch — TDD artifact by layer (unit / integration / app.Test), files owned, acceptance criteria mapping to ledger rows. Every deferral records WHERE, WHY, DATE.

## 5. Execution

Parallel sub-agents per batch group with strict file-ownership partitioning (no shared files). TDD red→green; regression tests must fail for the expected reason. Per-package targeted tests inside each agent; full `make check` at the merge point only.

## 6. Commit series

Logical commits from prepared staging lists — one concern each, baseline drift isolated into its own honest commits. Conventional Commits. Never commit automatically without explicit authorization.

## 7. Close-out

Flip ledger rows to closed with evidence; update beta-gate honesty markers; refresh `docs/README.md` status column; run `make check && make coverage && make test-integration-dev` green on final tree.
