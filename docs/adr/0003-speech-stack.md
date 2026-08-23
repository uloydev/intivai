# ADR-0003: Speech architecture — Whisper STT + Edge/Kokoro TTS

- **Status:** accepted (as design; implementation remains a gated demo — see ADR-0006)
- **Date:** 2025 (design), recorded 2026-08-24
- **Deciders:** EM
- **Source:** `docs/engineering/design-decisions.md` §3

## Context

Voice interviews need STT and TTS. Paid stacks (OpenAI Realtime ~$0.90/interview, Deepgram+ElevenLabs ~$0.28) conflict with the volume-pricing strategy; the product must sustain $0.50–1.00/interview revenue with ~$0.001 COGS.

## Decision

Self-hosted open-source pipeline: WebRTC audio via Pion → VAD segmentation (silero or energy-based) → Whisper via whisper.cpp subprocess (`tiny` dev / `small`–`large-v3` prod for Indonesian accuracy) → LLM → Edge TTS (with Piper local fallback; the community Edge endpoint may disappear without notice) → browser.

## Alternatives Considered

| Option | Why not |
|---|---|
| OpenAI Realtime API | Cost + vendor lock-in |
| Deepgram + ElevenLabs | Premium tier only, three paid APIs |
| LiveKit | Chosen as the upgrade path when voice becomes paid/scaled (SFU, TURN, recording bundled) |

## Consequences

- Near-zero marginal cost per voice interview.
- Hardware-bound concurrency (~2–3 sessions on a single GPU box); scaling requires the LiveKit migration.
- Edge TTS endpoint is unofficial — Piper fallback is mandatory, not optional.
