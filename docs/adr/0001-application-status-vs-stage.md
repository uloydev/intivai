# ADR-0001: Application Status and Stage are separate concepts

`applications` carries two state machines: **Status** (screening/passed/rejected — written by background workers as pipeline mechanics) and **Stage** (applied→hired — the recruiter's hiring decision, written via `PATCH /applications/:id`). We deliberately keep them apart rather than merging into one lifecycle enum.

Status records where the automated pipeline is; Stage records what a human decided. Merging them would let worker automation and human decisions overwrite each other (e.g. a re-score flipping `hired` back to `screening`). Stage is nullable (null = no decision yet), transition-validated on the backend (linear ladder + `rejected`/`screening_failed` from any earlier point; backward moves admin-only), and has no side effects in beta.

**Consequences**: the FE must not derive Stage from Status fields — "undecided" is shown explicitly; the score/extract workers must never write Stage.
