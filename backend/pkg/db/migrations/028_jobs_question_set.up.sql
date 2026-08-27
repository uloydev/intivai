-- D5: interview questions are LLM-generated ONCE per job at publish time and
-- stored here, so every candidate for the job hears the SAME set and the
-- interview hot path never waits on a provider. Nullable on purpose: jobs that
-- pre-date the feature (or whose generation failed) fall back to the
-- deterministic template generator at interview start.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS question_set JSONB;

-- Terminal generation failure surface (mirrors candidates.error_message).
-- Set only when generation gave up; cleared on a successful generation.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS question_set_error TEXT;
