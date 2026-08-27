-- B4: candidate Q&A pairs (free-form questions about the company/role) are
-- logged to the interview so recruiters see them beside the chat transcript.
-- Column-scoped (never rewritten by the scored transcript / answer commits).
ALTER TABLE interviews
    ADD COLUMN IF NOT EXISTS qa_pairs JSONB;
