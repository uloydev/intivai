---
description: "New feature requirement: multi-perspective analysis, decision grill, batch plan with ownership maps and staging lists"
---

Turn one or more new functional requirements into executed, committed work. Follow every step in order.

## 1. Code-grounded analysis

Before any opinion: read the relevant domain, repo, handler, worker, and FE files the requirements touch. State current coverage per requirement ("~80% shipped" beats generic advice). Analyze from fixed perspectives: product/HR value, domain modeling, data/schema, LLM/prompt design, fairness/legal, security/abuse, cost, UX flows, API contract, testing strategy.

Done = each perspective addressed with file:line evidence; interplay between requirements stated.

## 2. Decision grill

Extract every open choice into question-tool questions — options first, recommendation labeled. Record answers as D1..Dn in the plan's decision register, each with its binding architectural consequence spelled out.

Done = zero open decisions; register frozen before implementation starts.

## 3. Feature plan file

Write `docs/plans/active/feature-plan-<date>-<slug>.md` containing:
- Decision register with consequences
- Batches ordered by risk/size; independent batches marked parallelizable
- Per batch: owned file list (ownership map — no file in two batches), TDD artifact per layer, schema/API impact, completion criteria
- Staging lists per planned commit (paths exactly) so the commit series is mechanical
- DoD checklist: make check, coverage floors, integration green, OpenAPI drift guard, schemas.md WS frames if protocol changed, smoke extension, fresh-DB boot if migration

## 4. Execution

One sub-agent per batch respecting the ownership map; shared harnesses (mock LLM provider, WS fiber+client harness) built as task 0 of their first consuming batch. TDD red→green throughout; LLM work uses deterministic mock chunks. Per-package targeted tests inside agents; full `make check && make coverage && make test-integration-dev` at the merge point.

## 5. Gates earned by prior sessions — never skip

- `-race` mandatory on interview/chat packages
- Protocol frame tests BEFORE handler wiring when WS changes
- FE api-lib contract tests cover status classes incl. empty-body success
- Composer-path changes re-assert safety-rails-last ordering
- Migration + repo + domain in ONE commit; fresh-DB boot verified

## 6. Commits

Mechanical series from the staging lists. Conventional Commits, imperative subject. Only on explicit user authorization.

## 7. Close-out

Plan file updated: executed criteria marked, deferrals recorded WHERE/WHY/DATE in carryover section. `docs/README.md` index refreshed if new docs appeared.
