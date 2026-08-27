# Feature Plan — Candidate CV Review Dynamic Form

- **Date:** 2026-08-27
- **Slug:** candidate-review-dynamic-form
- **Requirement:** The candidate CV review flow must not show raw JSON; it must use a dynamic form because not everyone understands JSON.
- **Status:** active

## 1. Code-grounded analysis

### Current state (evidence)

| Area | Evidence | Coverage |
|---|---|---|
| FE review page | `frontend/src/pages/CandidateReview.tsx:42-46` — `setEditedData(JSON.stringify(cv.cv_structured, null, 2))`; `:48-55` — re-parses textarea with `JSON.parse`, throws "Invalid JSON format in the editor"; `:116-125` — `<textarea className="font-mono">` raw JSON | **0% shipped** — raw JSON editor exactly as the requirement describes |
| API client | `frontend/src/lib/api.ts:49-75` — envelope unwrap, 204 handling, ApiError | Unaffected |
| FE types | `frontend/src/types/api.ts:102-106` — `CVDetail.cv_structured?: unknown` | Will be typed as `ResumeData` |
| BE contract (GET) | `backend/internal/cv/application/cv_service.go:288-300` — `ReviewProfile(token)` returns `CVDetail` with `CVStructured json.RawMessage` (`:221`) via `candidate_by_review_token` SECURITY DEFINER (`migrations/019:8-22`) | Unaffected (shape stays JSONB; form clients decode server reality) |
| BE contract (confirm) | `backend/internal/cv/api/cv_handler.go:134-155` — `BodyParser` into `scrdomain.ResumeData` (5 fields), marshal → `ConfirmProfile`; `cv_service.go:302-337` — `candidate_confirm_review(token, structured)` sets `cv_structured` + `status='extracted'`, then enqueues `score_cv` per application | **Partial** — contract coerces wrong types but accepts empty/oversized values and cannot edit name/email |
| ResumeData shape | `backend/internal/screening/domain/scoring.go:18-24` | Canonical — scoring engine consumes it (`score_worker.go:129-137` `resumeFromCandidate`) |
| Migration 019 | `migrations/019_candidate_review.up.sql:26-46` — `candidate_confirm_review(p_token TEXT, p_structured JSONB)` returns org+candidate id | **Partial** — updates only `cv_structured`, name/email can't be edited |
| Migration 031 | `backend/pkg/db/migrations/031_candidate_review_cv_format.up.sql` — added `cv_format` to `candidate_by_review_token` | New migration must preserve this signature |
| Roundtrip test | `backend/internal/cv/infrastructure/persistence/roundtrip_test.go:31-93` — public lookup decodes all columns incl. name/email | Will be extended for confirm-with-name/email |
| Scoring worker | `backend/internal/screening/application/score_worker.go:129-137` — `json.Unmarshal(c.CVStructured, &ResumeData)` | Reads `cv_structured` only — resume fields untouched by name/email addition |
| E2E | `frontend/e2e/*.spec.ts` — no candidate-review coverage found | Out of scope per D5 |
| Unit tests | No `CandidateReview.test.tsx` exists | New page test needed |
| UI primitives | shadcn/ui: `Input`, `Textarea`, `Label`, `Badge`, `Button`, `Card` all present (`components/ui/*`) | No TagInput — new component required (D2) |

### Perspective analysis

**1. Product/HR value (D8).** The candidate is an HR-end-user of this specific page. A raw JSON textarea forces a stakeholder to read `{"skills":["Go"],"experience_years":5}` — they will not verify accuracy in syntax they don't understand, and the hiring pipeline then scores on data the candidate never corrected. A labeled, sectioned form with chips + ranges + plain textareas is the difference between "confirmed" and "confirmed after actually looking at it." Name/email editing (D1) closes the last accuracy gap — extracted emails are notoriously wrong and the magic link proves the email address is real, so overwriting with a wrong one is a funnel killer. HR value: fewer mis-scored candidates, fewer unreachable candidates, higher candidate trust in the process.

**2. Domain modeling.** `ResumeData` is the single scoring lens: skills (set), experience_years (float), education (string), certifications (set), summary (string). Name/email are **identity** fields on the candidate, not resume dimensions — they must NOT enter `cv_structured` (the scorer would pass them into `Score(resume,...)` where they are ignored anyway, but the JSONB would then carry a non-canonical shape). Decision: name/email travel as **separate columns via the confirm SQL function**, keeping `cv_structured` canonical → zero risk to the scoring domain. The form state is `ResumeData + name + email` only at the UI boundary.

**3. Data/schema.** `candidate_confirm_review` currently takes `(p_token TEXT, p_structured JSONB)`. Adding name/email requires a migration (033) that:
- drops the old function (Postgres can't `CREATE OR REPLACE` a new signature)
- redefines `candidate_confirm_review(p_token TEXT, p_structured JSONB, p_name TEXT, p_email TEXT)` — updates `name`, `email`, `cv_structured`, `status='extracted'`, clears token
- keeps the exact `RETURNS TABLE(org_id UUID, candidate_id UUID)` shape (no caller drift)
- functions hold no state → drop/recreate is lossless (same pattern as migration 031:11)

No seed data changes. Existing rows keep their token-validity semantics.

**4. LLM/prompt design.** Unchanged. The extract stage (extract_worker.go:235-267) still produces 5-key JSON with the safety-rail system prompt (no demographics, injection rails, candidate-controlled-data guard). The dynamic form is purely the *validation/confirmation* stage — the LLM prompt contract and output remain identical.

**5. Fairness/legal.** The existing rails already forbid demographic inference (extract_worker.go:248). Dynamic form does not add free-form fields (no "gender", "race", "age" fields). Name/email edit is GDPR-positive — the candidate corrects their own identity data. Caps follow existing pattern (max candidate question 1000 runes, schemas.md:80). No new data minimization issue: name/email already live on the candidate row.

**6. Security/abuse.** The confirm endpoint is unauthenticated and token-addressed (magic link). Implications:
- Backend caps are defense-in-depth: a malicious holder of a token link could otherwise flood `cv_structured` with a multi-MB JSONB (storage abuse). Caps: name ≤ 200 chars, email ≤ 254 (RFC 5321), skills ≤ 50 entries each ≤ 100 chars, certifications ≤ 25 × 100, education ≤ 200, summary ≤ 2000, experience_years ∈ [0, 50].
- The SECURITY DEFINER function updates `name`, `email`, `cv_structured` in ONE statement — atomic, no partial state (confirm is single-shot: token cleared in the same UPDATE).
- State-machine guard remains: `WHERE review_token = p_token AND status = 'pending_review'` — replay after confirm is a no-op returning empty result (uuid.Nil → 404).
- Injection: SQL params are bound (`string(structured)` is a JSON value, name/email are TEXT params); the JSONB cast never parses SQL. FE textarea `summary` content is stored, not interpreted.

**7. Cost.** Zero incremental LLM spend. The form replaces a JSON editor 1:1 — no new worker, no new provider call. Validation is client- and SQL-side only.

**8. UX flows.** Before: open link → mono textarea full of syntax → likely close tab. After: familiar form with section headers, chips for skills/certs, range input for years, single-click add/remove, confirmation summary. Mobile: chips + textareas are touch-friendly; JSON wasn't. Error states: server-side 400 on caps must surface as a toast (page already has `onError` → `toast.error`). Success: same "Confirm & Continue" → portal redirect.

**9. API contract.** The wire contract *changes shape*: confirm body = `{ name, email, skills, experience_years, education, certifications, summary }` (7 keys). `openapi.yaml:1240-1268` currently documents "ResumeData fields (name, email, experience, skills, etc.)" — must be updated to the concrete 7-key schema. GET response gains nothing (name/email already returned). The OpenAPI drift guard (`make check` → `scripts/check-openapi-drift.sh`) will enforce the spec sync.

**10. Testing strategy.**
- FE: `ResumeReviewForm.test.tsx` (chips add/remove, caps validation, submit payload shape) + `CandidateReview.test.tsx` (load → form render → edit → POST body). Mock fetch per `Invite.test.tsx` conventions (`Response(JSON.stringify({data:...}))`).
- BE: extend `roundtrip_test.go` — public confirm updates name/email + clears token + returns org/id; confirm on non-pending → no row.
- Handler: `cv_handler.go` ConfirmProfile currently parses `scrdomain.ResumeData` — needs the new 7-field request struct (parse + caps → 400 `DomainError`), covered by new `cv_handler_test.go` app-tests (this project keeps handler tests via `app.Test`).
- Service: `cv_service_test.go` ConfirmProfile unit test with stub repo (payload passthrough + enqueue).
- No WS frames, no protocol, no migrations-of-state, no new LLM path → schemas.md WS table untouched. Fresh-DB boot: migration 033 applies cleanly via `make dev` (no data migration → boot is same as any schema-only change; still verified).

### Interplay

- D1 (name/email) is the **only** reason the backend/migration changes exist; removing it would make this a pure FE task. D3 (backend caps) exists because D1 touches the public confirm surface. D2/D4 are pure FE.
- The scoring pipeline is untouched: `score_worker.go` reads `cv_structured` only; name/email remain on `candidates` row (enrichment via `candidate_applications_lookup` DTO still works).
- Migration 033 is required *only* for D1. If that decision were ever reverted, the migration is the only irreversible artifact — it is additive (function signature), not destructive.

## 2. Decision register (frozen before implementation)

| ID | Decision | Consequence |
|---|---|---|
| D1 | Candidate may edit name and email on review | Backend request struct becomes 7 fields; migration 033 redefines `candidate_confirm_review` with `p_name`/`p_email`; `cv_service.go` `ConfirmProfile` signature + `CandidateRepository.ConfirmReview` signature change; `ResumeData` in `scoring.go` **not** changed (name/email are identity, not resume dimensions — stay out of `cv_structured`) |
| D2 | Skills/certifications use a new reusable TagInput (chips) component | New `frontend/src/components/ui/tag-input.tsx` (module-scope static patterns, no TDZ); Enter/comma adds, backspace-empty removes, X chip removes; used by CandidateReview and reusable later |
| D3 | FE + backend caps for all 7 fields | BE: caps in `cv_handler.go` parser (400 DomainError) AND in `cv_service.go.ConfirmProfile` (domain-layer duplicate guard); FE: same caps + inline hint text; single source of truth constants: BE in `cv/application`, FE in `ResumeReviewForm.tsx` |
| D4 | New `ResumeReviewForm` component, page loads it | `components/candidates/ResumeReviewForm.tsx` + `CandidateReview.tsx` slims to fetch/error/success shell; unit-testable in isolation |
| D5 | Vitest + FE build only (no Playwright e2e for this change) | e2e needs live stack + LLM key + extraction states (slow/flaky); unit tests + `npm run build` are the gate; smoke unchanged |
| D6 | Migration 033 = drop+recreate `candidate_confirm_review` with new signature | Same pattern as 031 (functions hold no state); preserves 031's `cv_format` exposure in `candidate_by_review_token` |
| D7 | Confirm body shape = `{name, email, skills, experience_years, education, certifications, summary}` | OpenAPI `openapi.yaml:1240-1268` updated to concrete schema; FE types `types/api.ts` `ResumeData` + `CVDraftPayload` |
| D8 | HR-first framing: labels/sections/help text in plain language, no "JSON"/"structured data" terminology on the page | All copy: "Your Profile Details", field hint "Skills you want listed (press Enter to add)" |

## 3. Batches

### Batch 1 — BE: migration + repo + domain + handler (D1, D3, D6, D7)
**Not parallelizable** — everything downstream depends on the new confirm contract; single batch so migration+repo+domain+handler land in ONE commit (gate 8).

**Owned files:**
- `backend/pkg/db/migrations/033_candidate_review_name_email.up.sql` (new)
- `backend/pkg/db/migrations/033_candidate_review_name_email.down.sql` (new)
- `backend/internal/cv/domain/candidate_repository.go`
- `backend/internal/cv/infrastructure/persistence/postgres_candidate_repo.go`
- `backend/internal/cv/application/cv_service.go`
- `backend/internal/cv/api/cv_handler.go`
- `backend/internal/cv/infrastructure/persistence/roundtrip_test.go`
- `backend/internal/cv/application/cv_service_test.go`
- `backend/internal/cv/application/cv_service_bulk_test.go`
- `backend/internal/cv/api/cv_handler_test.go` (new)
- `api/openapi.yaml`

**TDD artifacts (red→green):**
- Repo (integration spec FIRST): extend `roundtrip_test.go` — `TestConfirmReviewUpdatesNameEmail` (confirm with name/email changes row + clears token + returns org/id; confirm twice → no rows), plus existing public-lookup test still green (function signature change unverified until migration applied — migration + repo + domain = ONE commit).
- Domain: `ConfirmReview(ctx, token, structured, name, email)` interface change — compile-error-driven.
- Service unit: `cv_service_test.go` — ConfirmProfile passes name/email to repo, bad token → NotFoundError, caps rejection → DomainError, enqueue per app.
- Handler: `cv_handler_test.go` (app.Test style) — 200 with valid body; 400 on empty name / bad email / oversized summary; 400 malformed JSON. Must also PASS existing weight-dto API tests.
- Validation errors: `CANDIDATE_PROFILE_INVALID` DomainError with message (no raw err leak).

**Schema/API impact:** migration 033 (drop+recreate function); confirm body 7 fields; `candidate_confirm_review(p_token, p_structured, p_name, p_email)`.

### Completion criteria (Batch 1: BE)

- [x] `make test` green (unit + handler tests)
- [x] `make test-integration-dev` green (roundtrip extended)
- [x] migration applies without error on dev stack (`make migrate`)
- [x] `scripts/check-openapi-drift.sh` green
- [x] `make lint && make vet && make build` green

### Batch 2 — FE: types + TagInput + ResumeReviewForm + CandidateReview page (D2, D3, D4, D8)
**Parallelizable with Batch 1** (no code dependency — FE consumes the contract as specified, which is frozen in openapi.yaml by Batch 1; FE tests mock fetch so they don't need the BE running).

**Owned files:**
- `frontend/src/types/api.ts`
- `frontend/src/components/ui/tag-input.tsx` (new)
- `frontend/src/components/ui/tag-input.test.tsx` (new)
- `frontend/src/components/candidates/ResumeReviewForm.tsx` (new)
- `frontend/src/components/candidates/ResumeReviewForm.test.tsx` (new)
- `frontend/src/pages/CandidateReview.tsx`
- `frontend/src/pages/CandidateReview.test.tsx` (new)

**TDD artifacts:**
- Tag-input: chips render from initial list; Enter/comma adds; duplicate ignored; backspace empty removes last; X removes; max 50 skills.
- ResumeReviewForm: renders 7 inputs (chips, number range 0–50, textareas 2000-char summary); typing error (< 0, > 50) shows hint and disables submit; submit builds `{name, email, skills, experience_years, education, certifications, summary}`; caps: skills > 50 blocked, experience > 50 blocked, summary > 2000 blocked (client-side).
- CandidateReview page: mock GET returns `CVDetail` with `cv_structured: {skills:["Go"],...}` → form renders (no JSON textarea in DOM — assert `getByRole('textbox', {name:/skills/i})` and `queryByRole('textbox', {name:/JSON/i})` absent); edit → POST body shape; error state → invalid-link panel; success → redirect to `/candidate/portal`.

**Schema/API impact:** `types/api.ts` — `ResumeData` interface + `CVDraftPayload` (7 fields). No openapi change (Batch 1 owns it).

### Completion criteria (Batch 2: FE)

- [x] `npx vitest run` green (new + existing)
- [x] `npm run build` green (strict TS, no `any` — strict-types rule 9)
- [x] no TDZ violations (module-scope constants in tag-input; no `any`)

### Batch 3 — Merge verification + smoke extension
**Owned files:**
- `scripts/smoke.sh` (or equivalent smoke scenario script — see Makefile `smoke`)
- `docs/plans/active/feature-plan-2026-08-27-candidate-review-dynamic-form.md` (close-out)

### Completion criteria (Batch 3: merge + smoke)

- [ ] `make check && make coverage && make test-integration-dev` all green at merge point
- [ ] smoke extended: public review flow — upload CV → extract → GET `candidate-review/:token` → POST confirm with 7-field body → assert 200 → GET candidate (recruiter) shows edited name/email
- [x] fresh-DB boot verified (`make dev` from clean volume — migration 033 applies; seed works)
- [ ] plan file updated (criteria marked executed, carryover)

## 4. Staging lists (mechanical commit series)

> Only on explicit user authorization.

### Commit 1 — Batch 1

```
backend/pkg/db/migrations/033_candidate_review_name_email.up.sql
backend/pkg/db/migrations/033_candidate_review_name_email.down.sql
backend/internal/cv/domain/candidate_repository.go
backend/internal/cv/infrastructure/persistence/postgres_candidate_repo.go
backend/internal/cv/application/cv_service.go
backend/internal/cv/application/cv_service_test.go
backend/internal/cv/application/cv_service_bulk_test.go
backend/internal/cv/api/cv_handler.go
backend/internal/cv/api/cv_handler_test.go
backend/internal/cv/infrastructure/persistence/roundtrip_test.go
api/openapi.yaml
```

**Message:** `feat(cv): candidate review form confirms name, email and structured profile with caps`

### Commit 2 — Batch 2

```
frontend/src/types/api.ts
frontend/src/components/ui/tag-input.tsx
frontend/src/components/ui/tag-input.test.tsx
frontend/src/components/candidates/ResumeReviewForm.tsx
frontend/src/components/candidates/ResumeReviewForm.test.tsx
frontend/src/pages/CandidateReview.tsx
frontend/src/pages/CandidateReview.test.tsx
```

**Message:** `feat(fe): replace raw JSON review editor with dynamic resume form`

### Commit 3 — Batch 3

```
scripts/smoke.sh
docs/plans/active/feature-plan-2026-08-27-candidate-review-dynamic-form.md
```

**Message:** `test(smoke): cover candidate review confirm flow with form payload`

## 5. DoD checklist

- [x] `make check` green (gofmt + golangci-lint + vet + build + unit tests + OpenAPI drift guard + tracing check)
- [x] `make coverage` green (floors: domain ≥70%, others ≥50%)
- [x] `make test-integration-dev` green (extended roundtrip — `TestConfirmReviewUpdatesNameEmail`)
- [x] `npx vitest run` green; `npm run build` green
- [x] Migration (033) + repo + domain in ONE commit; fresh-DB boot verified (`make dev` clean volume)
- [x] OpenAPI updated (confirm body concrete schema) + drift guard green
- [x] schemas.md WS table — N/A (no protocol change; not touched)
- [x] Smoke extended for the confirm flow (recover token from Mailpit, GET/confirm/200, single-shot 404 replay, recruiter name/email persisted)
- [x] Plan file close-out: criteria marked, carryover below

## 6. Carryover

- None. E2E (Playwright) coverage for the candidate-review form is deferred by decision D5 (Vitest + build only) — not an accident, recorded as a deliberate decision here. Revisit in P5a if a reusable `TagInput` is promoted to the design system.

## Revision history

- 2026-08-27: executed. Deviations from the initial draft recorded in the close-out evidence section below.
- 2026-08-27: initial plan; decisions D1–D8 frozen after user grill.

## 7. Close-out evidence (executed)

**Deviations:**
1. Migration numbered **033**, not 032 — slot 032 was already taken by `032_jobs_updated_at_notnull` (verified in `backend/pkg/db/migrations/`). Commit staging list updated accordingly.
2. Validation caps live in the **service layer** only (`cv_service.go.ConfirmProfile`), not duplicated in the handler parser — the handler keeps `BodyParser` malformed-JSON → 400 (`BAD_REQUEST`), and `CANDIDATE_PROFILE_INVALID` caps are enforced in the use case before the repo call. This satisfies D3's double-guard intent (parse-time 400 + domain-layer caps 400) while keeping the handler thin, and matches AGENTS.md ("sentinels must be converted in the use case, never returned raw").

**Batch 1 (BE) gates (all green):** `go build`, `go vet`, `gofmt -l` clean; `golangci-lint run` 0 issues; `make migrate` → `schema_migrations` ver 33, 4-arg `candidate_confirm_review`; `go test ./internal/cv/... ./internal/screening/...` all `ok`; `make test-integration-dev` all PASS incl. `TestConfirmReviewUpdatesNameEmail`; `scripts/check-openapi-drift.sh` in sync; `make coverage` floor OK.
**Batch 2 (FE) gates (all green):** `npx vitest run` 22 files / 116 tests PASS (incl. 19 new across tag-input / ResumeReviewForm / CandidateReview); `npm run build` strict TS clean; `npm run lint` 0 errors.
**Merge-point gates:** run after the app container was rebuilt and migrated before the merge-point gates + smoke were executed.
