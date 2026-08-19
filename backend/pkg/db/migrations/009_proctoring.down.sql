DROP INDEX IF EXISTS idx_interviews_proctoring_summary;

ALTER TABLE interviews
    DROP COLUMN IF EXISTS proctoring_events,
    DROP COLUMN IF EXISTS proctoring_summary;
