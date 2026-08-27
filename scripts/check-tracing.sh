#!/usr/bin/env bash
# check-tracing.sh — gate: new backend code inherits tracing ONLY through the
# instrumented seams (docs/plans/archive/otel-tracing-plan-2026-08-26.md).
# This script fails when any seam is bypassed, so un-instrumented code cannot
# merge. Grep + ordering-awareness — runs in milliseconds inside `make check`.
# Rules R1-R9 are the plan's original seams; R10-R13 are the J-series
# additions (ws context, span-attribute sanitization, error redaction,
# non-blocking export). Comment-only lines are ignored (J15).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../backend" && cd .. && pwd)/backend"
fail=0

err() { echo "TRACING GATE: $1"; fail=1; }

# code_lines — every non-comment, non-blank line of $1-$2 as file:line:content.
# Strips leading whitespace and '//'/'#' comments so the rules only match code
# (J15: bare grep would flag commented-out references).
code_lines() {
  grep -rn "$1" "$2" --include='*.go' 2>/dev/null \
    | sed 's/^\([^:]*:[0-9]*:\)[[:space:]]*//' \
    | grep -v '^[[:space:]]*//' \
    | grep -v '^[[:space:]]*#' \
    || true
}

# R1 — span-context loss: c.Context() is fasthttp's RequestCtx and carries NO
# OTel span; handlers must thread c.UserContext() downstream.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line uses c.Context() — span context lost; use c.UserContext()"
done < <(code_lines 'c\.Context()' "$ROOT/internal" | grep -v '_test.go' || true)

# R2 — worker deliveries must be traced: the queue middleware wraps panic
# recovery so every asynq task gets a queue.process span with producer link.
main_go="$ROOT/cmd/server/main.go"
if ! grep -q 'queue.TracingMiddleware()' "$main_go"; then
  err "cmd/server/main.go: worker mux missing queue.TracingMiddleware() (wire FIRST)"
fi

# R3 — SQL spans: pool must register the otelgorm plugin.
if ! grep -q 'otelgorm.NewPlugin()' "$main_go"; then
  err "cmd/server/main.go: db.NewPool missing db.WithPlugin(otelgorm.NewPlugin())"
fi

# R4 — HTTP server spans middleware present in the fiber chain.
if ! grep -q 'httpmw.Tracing(' "$main_go"; then
  err "cmd/server/main.go: fiber chain missing httpmw.Tracing(...)"
fi

# R5 — both binaries bootstrap telemetry (noop-safe when disabled).
for bin_main in cmd/server/main.go cmd/sandboxd/main.go; do
  if ! grep -q 'telemetry.Init(' "$ROOT/$bin_main"; then
    err "$bin_main: telemetry.Init(...) missing"
  fi
done

# R6 — global-provider footgun: a package-level otel.Tracer var freezes onto
# the FIRST provider ever installed (global delegation is once-only) and
# silently drops later Inits. Tracers must resolve per call.
while IFS=: read -r file line content; do
  [ -n "$file" ] || continue
  err "$file:$line package-level otel tracer var — resolves once globally; use otel.Tracer() per call"
done < <(grep -rnE '^var [a-zA-Z]+ *= *otel\.Tracer\(' "$ROOT" --include='*.go' || true)

# R7/R8 — enqueue bypass: only pkg/queue may touch asynq client primitives,
# otherwise traceparent header injection and typed-payload discipline leak.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line raw asynq enqueue outside pkg/queue — use queue.Client.Enqueue (header propagation)"
done < <(grep -rn 'EnqueueContext(\|asynq\.NewTask(\|asynq\.NewTaskWithHeaders(' "$ROOT/internal" "$ROOT/pkg" \
  --include='*.go' | grep -v '_test.go' | grep -v '/pkg/queue/' || true)

# R9 — LLM calls must flow through internal/llm (per-attempt spans live there);
# direct OpenAI SDK/provider construction outside llm is forbidden.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line direct LLM provider usage outside internal/llm — use llm.Client (spans + retries + ledger)"
done < <(grep -rn 'openai\.' "$ROOT/internal" --include='*.go' | grep -v '_test.go' | grep -v '/internal/llm/' || true)

# R10 — WS connections must inherit the Fiber handshake span: a raw
# context.WithCancel(context.Background()) root-drop orphans every interview
# span (J4). The connection ctx must be seeded from the stashed fiber ctx.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line WS connection context starts from context.Background() — handshake span lost; seed from Fiber UserContext (J4)"
done < <(code_lines 'context\.WithCancel\(context\.Background\(\)\)' "$ROOT/internal/interview/api" | grep -v '_test.go' || true)

# R11 — client-controlled telemetry attributes: the WS action must pass an
# allowlist before reaching a span attribute (J5), not string(m.Action) raw.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line raw client action copied into telemetry — sanitize via allowlist first (J5)"
done < <(code_lines 'attribute\.String\("interview\.action", string\(' "$ROOT/internal" | grep -v '_test.go' || true)

# R12 — span error data must be redacted (J9): raw err.Error() rides into
# span status/events; provider bodies can echo candidate content.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line raw err.Error() into span status — use a sanitized error class (J9)"
done < <(code_lines 'SetStatus\(codes\.Error, err\.Error\(\)\)' "$ROOT/internal/llm" | grep -v '_test.go' || true)

# R13 — non-blocking export: WithBlocking() lets collector backpressure block
# request/worker/WS paths (J3). Export must be bounded + non-blocking.
while IFS=: read -r file line _; do
  [ -n "$file" ] || continue
  err "$file:$line WithBlocking() — collector backpressure can stall business paths; use bounded non-blocking (J3)"
done < <(code_lines 'WithBlocking\(' "$ROOT/pkg/telemetry" | grep -v '_test.go' || true)

if [ "$fail" -ne 0 ]; then exit 1; fi
echo "tracing gate ok"
