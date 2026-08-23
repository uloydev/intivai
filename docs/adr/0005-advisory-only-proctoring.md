# ADR-0005: Proctoring is advisory-only

- **Status:** accepted
- **Date:** 2026-08 (recorded 2026-08-24)
- **Deciders:** EM
- **Source:** `docs/engineering/design-decisions.md` sync table §6; competitive review candidate-lens

## Context

Proctoring telemetry (tab switches, paste ratios, away time, focus loss) is client-reported and therefore spoofable and lossy. Auto-failing candidates on spoofable signals creates fairness and legal exposure; hiding it entirely removes a recruiter trust feature.

## Decision

Proctoring stays strictly **advisory**: events are displayed only in the recruiter audit trail, labeled "unverified", with an explicit flag-for-human-review path. The integrity score is never mixed into the evaluation score and can never auto-fail or auto-pass an interview. Candidates are not shown proctoring UI during interviews, but the consent flow must disclose what is collected (FINDINGS B5/B7 track the copy fix).

## Alternatives Considered

| Option | Why not |
|---|---|
| Hard anti-cheat scoring (auto-fail) | Spoofable input + legal/fairness exposure |
| Drop proctoring from MVP | Removes differentiator recruiters asked for in discovery |

## Consequences

- Recruiters keep final judgment; no automated adverse decision on unverifiable data.
- Requires honest "no telemetry recorded" states (tracked FINDINGS B7).
