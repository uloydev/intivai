# ADR-0008: Voice demo route stays mounted (amends ADR-0006)

- **Status:** accepted (supersedes the unmount clause of ADR-0006; scope decision otherwise unchanged)
- **Date:** 2026-08-24
- **Deciders:** EM

## Context

ADR-0006 settled voice as demo-only and stated "the backend route stays unmounted in `main.go`". Reality diverged: `cmd/server/main.go:430` calls `chatHandler.RegisterVoiceRoutes(v1, cfg.App.AllowedOrigins)` unconditionally, serving `GET /interviews/:id/voice` (`interview/api/voice_handler.go`) behind full auth. The 2026-08-24 deep review flagged this as a four-document contradiction (ADR-0006, `api/openapi.yaml`, `design-decisions.md`, `roadmap.md`) and an OpenAPI drift-guard blind spot, since helper-mounted routes escape the drift check.

The route being live is not accidental scope creep to undo — staged sales demos depend on the page working without local rebuilds. The defect was documentation asserting an invariant the code never had, plus spec omission.

## Decision

**Keep the route mounted. Rewrite the documents to match reality.**

- Voice remains demo-only and off the beta critical path (ADR-0006 scope stands).
- The mounted route is auth-guarded (origin allowlist + interview auth path); it stays excluded from beta commitments and from any customer-facing capability claims.
- `api/openapi.yaml` gains the `GET /interviews/{id}/voice` path with a demo-gated annotation.
- Roadmap §5 and design-decisions §3 describe the route as mounted-but-demo-gated.
- The FE voice page defects found in the same review (recruiter auth JWT sent as `?ticket=`, missing cleanup on close) are fixed under the fix plan — the page becomes a proper ws_ticket consumer.

## Alternatives Considered

| Option | Why not |
|---|---|
| Unmount the route (align code to ADR-0006) | Breaks staged demos; the mount is auth-guarded and harmless while docs lie is the actual defect; unmounting also deletes the drift-guard regression fixture this incident created |
| Keep code and docs as-is | Four documents asserting a false invariant defeats the docs-truth governance rule and hides a live surface from the spec |

## Consequences

- Docs and code agree again; OpenAPI covers every registered route (drift guard can be extended to assert this).
- Drift guard gets a self-test proving helper-mounted routes are detected (fix plan 0.4).
- Marketing/demo constraint from ADR-0006 carries over: voice must not be presented as live until real STT/TTS land (P5).
- Revisit trigger unchanged: paying-customer voice requirement → P5 ADR supersedes both 0006 and 0008.
