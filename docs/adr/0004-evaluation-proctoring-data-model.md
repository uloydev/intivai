# ADR-0004: Evaluation & proctoring data model — canonical reports schema

- **Status:** accepted
- **Date:** 2025 (design), recorded 2026-08-24
- **Deciders:** EM
- **Source:** PRD Epic 7; `docs/engineering/schemas.md`; `internal/evaluation`

## Context

Post-interview results need one canonical shape across the LLM evaluator, the WS `evaluation` frame, persistence (`interviews.evaluation JSONB`), and the recruiter UI/PDF. Proctoring telemetry needs a storage model that supports an advisory-only posture.

## Decision

One evaluation JSON schema (overall_score, weighted dimensions, per_question with rationale + verbatim quotes, strengths/weaknesses, recommendation enum) — single source in `docs/engineering/schemas.md`, mirrored by the Go `Report` struct. The domain layer recomputes the final score from per-question data; **the LLM never sets the final number**. Proctoring events persist raw (capped at 500/interview) plus a summary; the integrity score is a separate report axis, never mixed into the evaluation score.

## Alternatives Considered

| Option | Why not |
|---|---|
| LLM returns overall directly | Hallucination risk becomes a hiring verdict |
| Free-form JSONB per tenant | Breaks cross-tenant comparability and PDF rendering |

## Consequences

- Every displayed number is recomputable and auditable; quotes ground each metric.
- Schema changes require migration + struct + schemas.md in one change.
