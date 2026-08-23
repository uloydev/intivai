# UAT Matrix — HR User Acceptance Scenarios

> Status: stub · Last-reviewed: 2026-08-24 · Owner: EM
> The PRD historically referenced a 60+-scenario UAT matrix
> (`uat_test_matrix.md`) that was never committed. This is the recovery stub:
> scenarios re-derived from the PRD epics and the 2026-08-19/22 reviews
> (four-lens + competitive). Each scenario needs a named pilot tester before
> beta.

| # | Epic | Scenario | Pass criteria | Status |
|---|------|----------|---------------|--------|
| 1 | Jobs | Create job with comp band + rubric; publish; verify visible on /careers | Visible only when published | TBD |
| 2 | Jobs | Unpublish → hidden from board, existing applications unaffected | — | TBD |
| 3 | CVs | Upload single PDF → extracted profile editable/reviewable | Honest status on failure | TBD |
| 4 | CVs | Bulk upload 50 PDFs; per-file status observable | No silent failures | TBD |
| 5 | Screening | Candidate above threshold auto-passes; below does not; breakdown explains score | Weights used are shown | TBD |
| 6 | Candidate | Apply via public board on a phone (375px) end-to-end | No layout breakage; OTP/magic link works | TBD |
| 7 | Interview | Invite link → consent → chat interview → completion screen | Consent required before start | TBD |
| 8 | Interview | Network drop mid-interview → reconnect resumes at unanswered question | No lost answers | TBD |
| 9 | Interview | Coding sandbox: candidate runs Go solution against tests | Results render; timeout honest | TBD |
| 10 | Scorecard | Recruiter opens result: scores + verbatim quotes + transcript | No fabricated values anywhere | TBD |
| 11 | Scorecard | Override AI recommendation with reason; audit entry created | Original + override both visible | TBD |
| 12 | Proctoring | Tab-switch events visible to recruiter only; labeled unverified | Candidate sees no proctoring UI | TBD |
| 13 | Human request | Candidate requests human interviewer → recruiter notified, interview paused | — | TBD |
| 14 | Portal | Candidate views own applications + exports data; erasure works | GDPR rights functional | TBD |
| 15 | Context | Admin sets company context + tenant prompt; next interview reflects them; safety rails hold against injection attempts | Rails non-overridable | TBD |
| 16 | Webhooks | Pilot ATS receives signed `interview.completed` payload | HMAC verifies; retries observed | TBD |
| — | — | remaining scenarios (to 60+) | — | TBD |
