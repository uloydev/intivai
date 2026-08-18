-- 010_coding_sandbox.down.sql
DROP INDEX IF EXISTS idx_interviews_coding_sessions;

ALTER TABLE interviews
    DROP COLUMN IF EXISTS coding_sessions;
