# Feature Plan — Per-Job Weights, Descriptive Questions, Candidate Q&A

Date: 2026-08-24
Status: superseded by `docs/plans/archive/intivai-remediation-plan-2026-08-26.md`
Source: product requirements review (same day); decisions confirmed by owner.

## Confirmed decision register

| ID | Decision | Choice |
|---|---|---|
| D1 | Weight immutability | **Lock at publish.** Weights fully frozen once `is_published=true`. No post-publication edits, no re-score flow needed |
| D2 | Q&A knowledge scope | **Per-job candidate context + inherited org company context**, merged and version-pinned at connect |
| D3 | Q&A caps + visibility | Cap **configurable by HR user** (org-level setting, default 10). Full Q&A log **visible to recruiters** post-interview |
| D4 | Weight sum rule | **Strict reject**: create/update fails when `scoring_weights` present and keys don't cover all five dimensions summing to 1.0 ±0.01. Omitting the field keeps org/global defaults |
| D5 | Question generation | **Replace templates with LLM generation.** Deterministic templates demote to offline fallback only |

## Architecture consequences (binding)

### D5 requires generate-at-publish
LLM generation must NOT happen at interview start — that would add latency, cost variance, and cross-candidate nondeterminism to the hot path. Instead:

- Publishing a job enqueues a question-set generation task (same trigger and worker pattern as rubric generation).
- Generated set is stored versioned on the job (`question_set` JSON, mirroring how `rubric` is stored).
- Every candidate interviewed for that job receives the SAME stored set — consistency across candidates is preserved AND questions are LLM-quality.
- Fallback chain at interview start: stored set → deterministic templates (existing generator stays as code) if generation failed or job pre-dates the feature.
- Generation worker follows established worker conventions: typed payload, idempotency guard on status, terminal failures set `error_message`, transient errors retryable.

### D1 simplifies fairness posture
No weight mutation after applications exist → no re-scoring semantics, no audit replay problem. Publish becomes the freeze line. FE disables the weights controls on published jobs; backend rejects with a dedicated domain sentinel.

### D2/D3 ride existing seams
- Context merge extends the existing compose path (base → tenant prompt → company context → job candidate context → safety rails last).
- Version pinning: interview records the job-context version at connect; mid-interview edits cannot alter what a candidate hears.
- Cap lives in org settings (extend the existing org-settings seam in `screening/application`), enforced server-side per session.
- Q&A pairs persist to the interview transcript so recruiters see them beside chat history.

## Batches (TDD, layer-adapted)

### B1 — Weights strict validation + publish lock (smallest, independent)
- Domain unit tests FIRST: full-map requirement, sum tolerance, `JOB_WEIGHTS_LOCKED` sentinel on update of published job, nil-means-defaults unchanged.
- Handler/OpenAPI: error codes documented; create/update request semantics unchanged otherwise.
- FE: create-job weights form (5 inputs, live resolved-sum display, strict error surface, disabled when published).

### B2 — LLM question generation replaces templates
- Worker: question-set generation on publish (mock provider tests with deterministic chunks — instant TDD).
- `Question` struct gains descriptive fields (`context`, `expectation`) persisted in the stored set; resume/replay renders same framing.
- Bias filter (`IsBiased`) gates every generated question — last-line defense regardless of source.
- Strict output validation: malformed LLM JSON → retryable failure, never partial persistence.
- Integration: worker pipeline round-trip, fallback-to-template path when set absent.

### B3 — Per-job candidate context + AI suggest endpoint
- Migration + repo + domain in ONE commit: versioned job-candidate-context storage.
- `POST /jobs/:id/candidate-context/suggest` returns an LLM-generated DRAFT from JD fields. Drafts are never auto-persisted — recruiter edits then saves (trust rule G10).
- `ContainsInjection` rail on saved context before persistence.
- Compose path merge + version capture at connect (unit tests on composer ordering; rails stay last).

### B4 — WS `candidate_question` frame
- Protocol frames: `candidate_question` / `qa_answer`; turn state machine untouched (answer flow isolated — regression test proving current question does not advance and scoring transcript unaffected).
- Grounded-answer prompt built ONLY from pinned contexts; explicit decline behavior when answer not contained in context (no hallucinated facts about salary/benefits/company).
- Server-side cap per session from org setting; exceeded → polite refusal frame.
- Q&A pair logging to transcript (recruiter-visible).
- Protocol tests via in-process fiber + ws client harness; second-connection and session-mismatch rules apply unchanged.

### B5 — Frontend
- Chat UI: distinct "ask about this role" affordance vs answer composer; cap indicator; graceful cap-exhausted state.
- Job form: candidate-context editor with ✨Suggest button (draft clearly labeled, edit-before-save).
- Org settings page: QA limit configuration.

## Definition of Done

- [ ] `make check` green; coverage floors held (new domain logic ≥70%)
- [ ] `make test-integration-dev` green (B2/B3/B4 pipelines)
- [ ] OpenAPI updated (`suggest` endpoint, context CRUD, error codes); drift guard green
- [ ] `schemas.md`: WS frames section documents `candidate_question`/`qa_answer`
- [ ] Smoke: happy-path includes suggest→save→interview→candidate-question flow
- [ ] Fresh-DB boot verified (new migration)
- [ ] FINDINGS row opened for known crudeness: `scoreEducation` keyword matching + binary certifications score — weights amplify crude sub-scores (future refinement, out of scope here)

## Sequencing

B1 → B3 (recruiter-side value lands early, no protocol risk) → B2 → B4 → B5 interleaved per batch. B1 and B3 are independent enough to parallelize after B1 lands.
