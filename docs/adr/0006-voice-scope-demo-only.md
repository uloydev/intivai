# ADR-0006: Voice interview scope — demo only, out of the beta critical path

- **Status:** accepted (settles D6)
- **Date:** 2026-08-24 (history: deferred 2026-08-18; pivoted back 2026-08-18 in P4_Plan D6)
- **Deciders:** EM

## Context

Voice was originally gated on a paying customer (`AI_Interviewer_Phases.md` Phase 5), briefly re-pivoted INTO the MVP (P4_Plan carryover D6, "Product Pivot"), then flagged by the 2026-08-19 four-lens review: the shipped voice page reads recruiter tokens, WS never authenticates, STT/LLM are mocked, Opus decode/encode is unimplemented, and the route is not mounted in production. Three documents carried three different scope states.

## Decision

**Voice is demo-only and NOT on the beta critical path.** The gated demo (Pion signaling, VAD/STT/TTS adapters, FE `/voice/:id`) stays as a sales artifact. The backend route stays unmounted in `main.go` (the OpenAPI spec records this explicitly). Full Phase 5 (real duplex audio, TURN, recording) lands only when a paying customer requires it — the original phase gate stands.

## Alternatives Considered

| Option | Why not |
|---|---|
| Keep voice in MVP (D6 pivot) | Mock pipeline sold as live = fabricated capability; solo capacity better spent on beta blockers |
| Delete the demo code | Sales value of a working demo outweighs maintenance cost |

## Consequences

- Roadmap, PRD, spec now agree on one state (resolves contradiction C4).
- Marketing must not present voice as live until ADR superseded.
- Revisit trigger: first paying customer requiring voice → new ADR supersedes this one.
