-- 009_proctoring.up.sql
-- Anti-Cheating & AI Proctoring Guardrails

ALTER TABLE interviews
    ADD COLUMN IF NOT EXISTS proctoring_events JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS proctoring_summary JSONB DEFAULT '{}'::jsonb;

-- GIN index on proctoring_summary for fast integrity/risk queries
CREATE INDEX IF NOT EXISTS idx_interviews_proctoring_summary ON interviews USING GIN (proctoring_summary);
