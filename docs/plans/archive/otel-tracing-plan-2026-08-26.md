# OTel Distributed Tracing Plan — Jaeger v2

- **Date:** 2026-08-26
- **Status:** superseded by `docs/plans/archive/intivai-remediation-plan-2026-08-26.md`;
  execution record retained below
- **Authors:** senior backend + tech lead review pass
- **Source:** debugging-pain retrospective after I19 (500 with zero signal → psql archaeology)

---

## 1. Problem statement

Debugging cross-boundary failures today means grepping zerolog lines and hand-running psql.
Three concrete pain classes this month:

| Pain | Example | What a trace would show |
|---|---|---|
| Silent SQL failures | I19: every valid review token → generic 500 | DB span with `SQLSTATE 42703`, statement text, parent route |
| Async chains invisible | parse → extract → score → notify; retry storms swallow enqueues (I17 class) | One continuous trace across HTTP boundary and asynq retries |
| WS interview turns | connect → compose → LLM stream → answer, multi-goroutine | Turn waterfall with per-stage latency and interrupt points |

Existing observability: Prometheus metrics (`pkg/metrics`), Sentry error capture
(`pkg/observability/sentry.go`), zerolog with `request_id`. None of these answer
"where did time go and what failed downstream".

## 2. Goals / Non-goals

**Goals**

1. End-to-end traces: HTTP request → tenant tx → GORM SQL → response.
2. Trace continuity across asynq enqueue → worker execute (incl. retries).
3. LLM call spans with model/provider/token/attempt attributes.
4. WS chat turn spans that survive goroutine hops (existing ctx plumbing reused).
5. `trace_id` in every zerolog log line → jump log ↔ waterfall.
6. Fail-soft: collector down must never degrade API behavior.
7. Zero PII in span attributes (HR product — hard rule).

**Non-goals (this phase)**

- Metrics changes (Prometheus stays as-is).
- Log shipping/aggregation stack (Loki etc.) — later decision.
- Tail sampling, SLO burn alerts, frontend/browser tracing.
- Continuous profiling.

## 3. Current-state audit (verified in code)

| Seam | Location | Notes |
|---|---|---|
| Process topology | `cmd/server` runs HTTP **and** in-process asynq workers (`main.go:490`); `cmd/sandboxd` separate gRPC+TLS binary | Two instrumentable processes, one deploy unit |
| Middleware chain | `main.go:350–366`: prometheus → fibersentry → recover → RequestID → Audit → CORS | Tracing mw slots in before RequestID so RequestID/Audit see the span ctx |
| DB access | `pkg/db/postgres.go` `NewPool` — plain gorm, **no plugins yet**, pgx stdlib driver | Plugin insertion point; pool created once per process |
| Queue | `pkg/queue/asynq.go` — ALL producers go through `Client.Enqueue(ctx,…)` (`:32`), workers start via `Server.Start(mux)` (`:71`) | Perfect single choke points for propagation; typed payloads stay untouched |
| LLM | `internal/llm/client.go` `Chat` (`:38`) / `ChatStream` (`:95`) wrap `Provider` iface, retry loop inside client | Span per attempt + parent operation span |
| WS | `chat_handler.go`: `wsWriter{ch chan any}` single-writer pattern, goroutines at :75/:389/:799/:884 | ctx flows through channels already — spans attach without restructuring |
| Tenant tx | `db.RunInTx(ctx, pool, orgID, fn)` | Trace ctx flows through automatically; org_id available as attr source |
| Config | `pkg/config/config.go`, viper, flat env names (`SENTRY_DSN` style) | Add OTEL block, same style |
| OTel dep | `go.opentelemetry.io/otel v1.45.0` indirect only | Promote to direct; add sdk + exporters |

## 4. Decisions (D-register — frozen before implementation)

### D1 — Backend: Jaeger v2 all-in-one
- Considered: SigNoz (unified traces+logs+metrics), Grafana Tempo, Sentry Performance, Jaeger.
- Chosen: **Jaeger v2 all-in-one** (OTel-collector-native distribution).
- Why: ~300MB RAM, best-in-class waterfall UI, OTLP-native ingestion, zero lock-in
  (swap backend = change one endpoint env). SigNoz brings ClickHouse (+1–4GB RAM) —
  wrong weight for a solo-VPS beta next to postgres/redis/minio/sandboxes. Tempo needs
  the whole Grafana ecosystem to be usable. Sentry couples trace volume to error-tool
  pricing and has weak backend-span views.
- Upgrade path documented in §11.

### D2 — Storage: in-memory (all-in-one default), capped
- Solo-VPS reality: losing traces on container restart is acceptable; they are debug
  data, not records. In-memory avoids a stateful volume + compaction tuning.
- Cap via sampling (D4) + container mem_limit 512MB (OOM-kills jaeger, never app).
- Upgrade path: badger storage volume when retention-across-restart hurts.

### D3 — Instrumentation: explicit OTel SDK, no auto-instrumentation
- Go idiom for this codebase: we control every seam (audit confirmed choke points).
- SDK init lives in ONE new package (`pkg/telemetry`) — vendor-neutral, swappable backend.

### D4 — Sampling: ParentBased(TraceIDRatioBased), ratio via env
- dev: `1.0`; prod: `0.1` initially.
- Head sampling only. Accepted trade-off: a rare error on an unsampled trace is lost —
  mitigated by Sentry (errors always captured there) carrying `sentry.trace.tag`.
- Revisit tail sampling (collector `tail_sampling` processor) post-beta if error-visibility
  gaps appear. Documented, deferred — not silently dropped.

### D5 — Propagation: W3C `traceparent` everywhere
- HTTP: standard headers. asynq: `Task.Header()` map (verify asynq ≥ v0.24 in task 0;
  fallback rejected — payload envelopes break the typed-payload rule).
- sandboxd gRPC (phase-2 batch): otelgrpc metadata injection.

### D6 — Logs↔traces: `trace_id` zerolog field, injected in middleware
- `httpmw.RequestID` already threads a logger; extend the chain: tracing mw stores
  `trace_id` in ctx → RequestID/Audit include it. Worker side: queue logger wrapper adds it.

### D7 — Sentry↔traces: tag sentry events with `trace_id`
- Cheap correlation win; keeps systems decoupled otherwise.

### D8 — Exporter: OTLP/HTTP (`:4318`), fail-soft
- Blocking exporter with small queue + drop-on-full; startup tolerates absent collector
  (logs warn once). API latency budget impact target <1ms p99.

## 5. Instrumentation matrix

Span naming follows OTel semconv: `HTTP {method} {route}`, `db.<operation>`,
`queue.enqueue <task>`, `queue.process <task>`, `llm.chat <provider>`, `interview.turn`.

| Component | Span type | Key attributes | Cardinality guard |
|---|---|---|---|
| Fiber HTTP | server span (otelfiber) | method, route template, status, request_id | route template, never raw path (IDs out) |
| Tenant tx | child span in `RunInTx` | `org.id` (UUID only) | UUIDs only, no slugs/names |
| GORM queries | client spans (otelgorm) | db.system=postgres, statement (otelgorm truncates), rows_affected, error.type | pgx driver compatible — verify in task 0 spike |
| asynq enqueue | producer span | task name, queue, task_id | task name constants only |
| asynq process | consumer span (new root per delivery, linked to producer) | task name, attempt, max_retry, trace link | — |
| LLM Chat/ChatStream | client span per attempt | llm.provider, llm.model, tokens in/out, attempt, finish_reason | NEVER prompt/completion content |
| WS interview turn | internal spans: compose, stream, persist, refusal | interview.id, question.idx, bytes streamed | no transcript text |
| Embeddings (batch E, optional) | client span | model, input length | no input text |

**PII redaction rule (hard):** candidate name/email, CV text, prompts, completions,
review tokens, JWTs — never as span attributes or resources. Attributes are IDs,
counts, durations, enum states. Code review gate: any `.SetString(` with user-domain
field requires justification comment.

## 6. Infrastructure changes

```yaml
# docker-compose.dev.yml (dev overlay) — new service
  jaeger:
    image: jaegertracing/jaeger:2 <pin digest>
    command: ["--set", "extensions.jaeger_storage.backends.memory.max_traces=10000"]
    ports:
      - "16686:16686"   # UI (dev only)
      - "4318:4318"     # OTLP/HTTP
    mem_limit: 512m
    healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:16686"], interval: 10s, retries: 5 }
```

- **Prod:** ship jaeger service in `docker-compose.prod.yml` behind the internal network
  (no host port; Caddy route optional, admin-only later). Start enabled at 10% sampling;
  flip `OTEL_ENABLE=false` = instant off.
- App connects via `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318` in compose env.
- CI: no jaeger service needed — all tracing tests use in-memory exporter (§8).

## 7. Code changes (file-by-file ownership map)

One concern per batch; batches land as separate commits.

### Batch A — foundation (noop-safe bootstrap)
| File | Change |
|---|---|
| `backend/pkg/telemetry/telemetry.go` (NEW) | `Init(ctx, cfg) (shutdown func(error), err)`; TracerProvider + resource (`service.name=intivai-server|sandboxd`, `deployment.environment`) + OTLP exporter; `OTEL_ENABLE=false` → noop provider. Options pattern for injecting a custom exporter (test seam). |
| `backend/pkg/telemetry/telemetry_test.go` (NEW) | Unit: noop when disabled; provider starts/stops clean; custom exporter receives one span end-to-end through a child span. |
| `backend/pkg/config/config.go` | `Telemetry` struct: `Enable bool`, `OTLPEndpoint string`, `SampleRatio float64`, `Env string` — viper keys `OTEL_ENABLE` (default false), `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER_ARG`. |
| `backend/cmd/server/main.go` | Init before fiber build; `defer` shutdown in graceful path; pass shutdown into lifecycle. Also wire into sandboxd boot (same helper). |
| `docker-compose.dev.yml` | jaeger service (§6) + app env vars. |
| `docs/engineering/architecture.md` | Observability section: add tracing paragraph + runbook link. |

Gate: `make check` green; app boots with `OTEL_ENABLE=false` unchanged behavior;
with true + jaeger up → hello-world span visible in UI.

### Batch B — HTTP server spans + log correlation
| File | Change |
|---|---|
| `backend/internal/shared/httpmw/tracing.go` (NEW) | otelfiber middleware (or thin custom if otelfiber's route handling fights group templates — decide in spike) placed BEFORE RequestID. |
| `backend/internal/shared/httpmw/requestid.go` | Include `trace_id` from ctx in the scoped logger (fallback: omit when noop). |
| `backend/pkg/metrics/prometheus.go` | No change — confirm no double-count interference (read-only check). |
| tests | `httptest` fiber app + sdktrace in-memory exporter: assert server span exists, route template attr, trace_id reachable in logger output (capture writer). |

Gate: unit green; `make smoke` passes; waterfall shows route → nested children in later batches.

### Batch C — GORM spans
| File | Change |
|---|---|
| `backend/pkg/db/postgres.go` | `NewPool(ctx, url, opts ...Option)` variadic (backward compat — zero churn at 30+ call sites); register otelgorm plugin when tracer enabled. Verify pgx-stdlib compatibility in task-0 spike (known issue class: prepared-statement spans). |
| `backend/internal/shared/dbtx/tenant_tx.go` (RunInTx location) | Child span `tenant.tx` with `org.id` attr wrapping fn. |
| tests | Roundtrip integration test with in-memory exporter registered via Option: SELECT produces db span with statement + rows; RLS tx produces `tenant.tx` child. |

Gate: `make test-integration-dev` green; I19-style failure now visible as errored db span.

### Batch D — asynq propagation
| File | Change |
|---|---|
| `backend/pkg/queue/asynq.go` | `Enqueue`: extract span ctx → `propagation.TraceContext{}.Inject` into `task.Header()` (guard: nil header map init). `Server.Start`: wrap mux — each delivery becomes root span `queue.process <task>` with LINK to producer trace (extracted header may be stale after retries — links over parents, honest causality); attrs: attempt, retried, max_retry. ErrorHandler log gains `trace_id`. |
| tests | Pure unit: inject→serialize task→extract round-trip preserves traceparent. Integration (env-gated): enqueue real task against TEST_REDIS, worker handler asserts span exported with link. |

Gate: integration green; CV upload → extract → score appears as ONE trace tree in dev UI.

### Batch E — LLM + WS interview spans
| File | Change |
|---|---|
| `backend/internal/llm/client.go` | Parent op span per `Chat`/`ChatStream` + child span per provider attempt (retry loop already structured for this); attrs per matrix §5; record error types (transient vs fatal distinction already in code). |
| `backend/internal/interview/api/chat_handler.go` | Turn-scoped internal spans reusing existing ctx plumbing (`wsWriter`, `streamDone` goroutines): `turn.compose`, `turn.stream`, `turn.persist`. No structural change — spans only. |
| tests | Fake Provider captures spans via in-memory exporter: attempt count == span count; stream cancel produces ended span with `interrupted=true`. Protocol tests untouched. |

Gate: `-race` green on interview packages (mandatory); dev waterfall of one interview turn matches §5 matrix.

### Batch F (optional, follow-up) — sandboxd gRPC + embeddings
- otelgrpc server interceptors in `cmd/sandboxd`, client interceptors at the sandbox
  gateway call site; embeddings client spans. Deferred with date: revisit 2026-09-05
  after A–E prove value.

## 8. Testing strategy (TDD, layer-adapted)

| Layer | Artifact | Cycle |
|---|---|---|
| telemetry pkg | pure unit (noop mode, provider lifecycle) | instant |
| propagation | pure unit (inject/extract round-trip, malformed-header tolerance) | instant |
| middleware/plugins | sdktrace in-memory exporter via Option injection — assert spans WITHOUT network | instant |
| asynq end-to-end | env-gated integration vs TEST_REDIS (project convention: repo/worker bugs live here) | slow batch |
| live view | manual: `make dev` + click through smoke scenario in jaeger UI | once per batch |

Red→green applies where logic exists (propagation, span attrs, noop gating).
Pure glue/config wiring is exempt per AGENTS.md but each batch ships an observable
verification step (UI screenshot evidence noted in commit body).

## 9. Performance & safety analysis

- **Overhead budget:** head sampling 10% prod + noop-capable SDK ≈ <1% CPU, <1ms p99 added
  latency; spans buffered async, dropped under pressure (never blocking request path).
- **Failure isolation:** exporter errors logged once, rate-limited; collector down ≠ app degraded.
  Verified by test: exporter pointing at closed port → requests still succeed.
- **Cardinality:** route templates only; task-name constants only (already enforced by
  `pkg/queue` const discipline); org/interview/job IDs are bounded UUID sets.
- **Security:** §5 PII rule; jaeger UI never exposed publicly (internal network; dev localhost);
  no auth on jaeger acceptable because network-isolated — same posture as redis/postgres today.
- **RLS:** tracing adds no new DB identity; spans run inside existing tx ctx. Zero new grants.

## 10. Rollout / rollback

1. Merge batches A–E behind `OTEL_ENABLE=false` default. Nothing observable changes in prod.
2. Dev: enable, run `make smoke` + one manual interview; inspect waterfall; fix attr gaps.
3. Prod: deploy jaeger service + enable at 0.1 sampling during low-traffic window; watch
   app p99 (prometheus) for 24h; raise sampling only after stable week.
4. **Rollback:** set `OTEL_ENABLE=false` (env change, redeploy-free via compose env) —
   code paths become noop. Jaeger container removable independently.

## 11. Upgrade paths (documented deferrals)

| Trigger | Action |
|---|---|
| Need traces to survive restarts | jaeger badger storage + volume |
| Error-on-unsampled-trace gaps hurt | tail_sampling processor (keep errors @100%) |
| Want unified logs+metrics+traces UI | evaluate SigNoz again — migration cost = exporter endpoint + backend only (SDK untouched) |
| Multi-service sprawl post-beta | OTel collector as deployment layer between app and backends |

## 12. Definition of Done (this phase)

- [ ] Batches A–E merged, `make check && make coverage && make test-integration-dev` green
- [ ] `-race` green on `./internal/interview/...` (batch E gate)
- [ ] Live evidence artifact: screenshot/exported JSON of one full trace
      (HTTP → tenant.tx → gorm → asynq enqueue → worker → llm) attached to plan
- [ ] `trace_id` present in audit log lines (grep proof)
- [ ] Runbook section in architecture.md: how to read the UI, how to find a trace from
      a request_id, how to flip sampling/off
- [ ] OpenAPI/docs lint unaffected; FINDINGS ledger row opened for any bug discovered
      while instrumenting (expected: at least one latent issue will surface — that is the point)

## 13. Task-0 spikes (before Batch A merge)

1. Pin asynq version; confirm `Task.Header()` availability (≥ v0.24). If missing:
   STOP — escalate D5 alternative discussion before writing code.
2. otelgorm × pgx-stdlib compatibility probe (prepared statements, RLS tx nesting).
3. otelfiber route-template fidelity with fiber group patterns (`/api/v1/cvs/:id`)
   — else fall back to thin custom middleware (~40 LOC).

## 14. Effort estimate

| Batch | Size | Risk |
|---|---|---|
| A foundation | 0.5 day | low (noop default isolates) |
| B HTTP+logs | 0.5 day | low |
| C GORM | 0.5 day | medium (plugin/driver interplay) |
| D asynq | 1 day | medium (links semantics, retries) |
| E LLM+WS | 1 day | medium (race discipline) |
| Total | ~3.5 days serial; A+B+C parallelizable after task-0 | |

## 15. Execution record (2026-08-26)

All batches landed in one session. Deviations + findings:

- **Task-0:** asynq v0.26.0 (Header ✓), otelgorm v0.3.2 via `NewPlugin()` factory
  (no struct), middleware = `gofiber/contrib/otelfiber/v2` v2.2.3 (module path differs
  from plan guess `otel-fiber`).
- **Jaeger v2.20 quirks (probed live):** digest-pinned; `--set` path is
  `extensions.jaeger_storage.backends.some_storage.memory.max_traces`; receivers bind
  `localhost` by default — **`JAEGER_LISTEN_HOST=0.0.0.0` env is mandatory** for
  cross-container OTLP.
- **OTel global footgun found & fixed:** package-level `otel.Tracer` vars freeze onto the
  FIRST provider ever installed (`delegateTraceOnce`); resolved per-call in queue/db/llm/api.
- **Bug class caught by instrumentation (I20):** public review handlers passed
  `c.Context()` (fasthttp ctx, no span) → orphaned gorm traces → fixed to
  `c.UserContext()`; zero other call sites (audited).
- **WS wiring fix:** stream goroutine now derives from the answer span ctx — one trace
  per candidate answer; span-flow test asserts nesting under `-race`.
- **Live evidence (dev stack, smoke run):**
  - `GET /api/v1/public/candidate-review/:token` → server span + nested `gorm.Row`
    (CHILD_OF verified via jaeger API).
  - Interview answer trace: `interview.answer.process ⊃ tenant.tx ⊃ gorm.* +
    llm.attempt 1..3 + llm.chat_stream` — retry storm visible as spans.
  - 20× `queue.process <task>` roots (extract_cv / generate_question_set /
    generate_rubric …), every one FOLLOWS_FROM-linked to its producing HTTP trace.
  - Audit log lines carry `trace_id` next to `request_id`.
- **Gates:** make check ✓, test-integration-dev exit 0 ✓, `-race` interview packages ✓,
  smoke PASSED (after fixing script drift: PATCH must send `is_published:true`, required
  by migration 017 lookup). Coverage: 15 LOW vs HEAD baseline 16 LOW (I18 unchanged,
  pre-existing).
- **Deferred (batch F):** sandboxd gRPC interceptors + embeddings spans — revisit 2026-09-05.
