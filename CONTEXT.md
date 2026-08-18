# Intivai Context

Intivai runs autonomous technical interviews: candidates apply publicly, get screened by AI, sit live chat/coding interviews with proctoring telemetry, and receive recruiter decisions — all inside per-tenant workspaces.

## Language

**Application**:
A candidate's submission for one job in one org. The row that ties candidate, job, screening outcome, interview, and recruiter decision together.
_Avoid_: submission (ambiguous), record

**Candidate**:
A person who applied (via public job board or recruiter CV upload). Identified by email within an org; may hold several applications.
_Avoid_: applicant (fine in UI copy, not in the model), user (reserved for recruiter/org accounts)

**Status**:
The pipeline-mechanics state of an application, written by background workers: `screening`, `passed`, `rejected`.
_Avoid_: state, lifecycle, phase

**Stage**:
The recruiter's authoritative hiring decision for an application: `applied`, `screening_passed`, `screening_failed`, `interview_invited`, `interview_completed`, `offer_extended`, `hired`, `rejected`. Null until a recruiter decides; never derived from Status.
_Avoid_: status, progress, milestone

**Interview**:
A live assessment session for one application (chat, coding, optionally voice). Has its own status machine (`pending`, `in_progress`, `completed`, `expired`) driven by the candidate's progression.
_Avoid_: session (that's the WS connection), assessment (that's the evaluation phase)

**Invitation Token**:
The 32-char high-entropy credential a candidate uses to enter an interview and obtain a WS ticket. Valid 7 days, single-use-then-reusable for resume.
_Avoid_: magic link, invite link (UI copy only)

**WS Ticket**:
Short-lived (10-min) JWT binding one connection to one interview + session; the chat/voice socket's bearer credential.
_Avoid_: token (collides with Invitation Token)

**Proctoring Event**:
A client-reported integrity telemetry record (tab switch, focus loss, paste, resize, audio anomaly) attached to an interview.
_Avoid_: telemetry (transport term), flag (that's the summary output)
