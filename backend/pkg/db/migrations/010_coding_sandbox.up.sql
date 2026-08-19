-- Coding Sandbox & Live Pair-Programming Terminal

ALTER TABLE interviews
    ADD COLUMN IF NOT EXISTS coding_sessions JSONB DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_interviews_coding_sessions ON interviews USING GIN (coding_sessions);
