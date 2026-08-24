# Canonical Schemas, Protocol Frames & Limits

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Single home for: the evaluation JSON schema, the WS frame summary, and the
> size/token limit table. `api/openapi.yaml` remains the wire contract; this
> doc is the human-readable reference. Other docs link here instead of pasting.

## 1. Evaluation report schema (canonical)

The evaluator LLM proposes; **domain code recomputes `overall_score`** and
validates every field. The Go mirror is `internal/evaluation/domain/report.go`.

```json
{
  "overall_score": 78,
  "dimensions": {
    "technical":        { "score": 82, "weight": 0.4  },
    "communication":    { "score": 75, "weight": 0.2  },
    "problem_solving":  { "score": 80, "weight": 0.25 },
    "culture_fit":      { "score": 70, "weight": 0.15 }
  },
  "per_question": [
    {
      "question_idx": 1,
      "score": 85,
      "rationale": "Strong understanding of distributed systems",
      "strengths": ["Clear explanation", "Used real examples"],
      "weaknesses": []
    }
  ],
  "strengths":   ["Go expertise", "System design"],
  "weaknesses":  ["Limited cloud experience"],
  "recommendation": "proceed"
}
```

Rules:
- Dimension weights sum to ≈ 1 (normalized server-side if not).
- `recommendation` ∈ {`proceed`, `hold`, `reject`} — invalid values fall back to `hold`.
- Every per-question rationale must cite verbatim candidate quotes.
- Persisted on `interviews.evaluation JSONB`; served via `GET /interviews/{id}`.

## 2. WebSocket chat frames (summary)

Authoritative definition: `api/openapi.yaml` → `GET /api/v1/candidate/interviews/{id}/chat`.

| Direction | Frame | Payload |
|---|---|---|
| S→C | `interview.start` | `{session_id, total_questions, session_budget_sec}` |
| S→C | `question` | `{idx, content, archetype, time_limit_sec, session_remaining_sec}` |
| S→C | `token` | `{content}` — streamed LLM response |
| S→C | `response` | `{content}` — finalized answer feedback |
| S→C | `evaluation` | `{scores, overall, recommendation, status: complete\|pending}` |
| S→C | `error` | `{code, message}` |
| S→C | `pong`, `code.result` | keepalive; sandbox execution result |
| C→S | `answer` | `{idx, content, pacing_telemetry:{time_to_first_keystroke_ms, duration_ms, typed_chars, pasted_chars, pasted_ratio}}` |
| C→S | `interrupt` · `ping` · `resume {session_id}` · `telemetry` · `code.change` · `code.run` | control / telemetry / sandbox |

Auth: ws_ticket JWT (10-min, bound to interview + session) via `?ticket=` or Authorization header. Origin allowlist enforced. One active connection per interview.

## 3. Limits & budgets

| Limit | Value | Source of truth |
|---|---|---|
| CV upload size | 10 MB (`CV_MAX_UPLOAD_MB`) | `pkg/config/config.go:153` |
| Company context size | 64 KB (`ctxMaxBytes`) | `internal/context/application/context_service.go:29` |
| Tenant prompt length | 4K chars max + integrity keyword rails | prompt validation (`context/domain`) |
| Interview context budget | 8000 tokens (`DefaultTokenBudget`) | `interview/domain/service/context.go:24` |
| Context window | last 10 Q&A pairs | same file |
| Interview duration cap | 30 min | interview domain state machine |
| Per-question timeout | 3-min WS read deadline | chat handler (design intent 5m idle — see design-decisions §2) |
| Heartbeat | server ping 30s / pong wait 10s | chat handler |
| Proctoring raw events | capped at 500/interview | proctoring domain |
| Invitation token validity | 7 days, single start, reusable for resume | interview token repo |
| WS ticket TTL | 10 minutes | iam auth provider |
