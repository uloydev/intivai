-- Revert Additions for Phase 5 Pivot

DROP TABLE IF EXISTS global_candidate_passports;

ALTER TABLE candidates 
    DROP COLUMN IF EXISTS review_token,
    DROP COLUMN IF EXISTS batch_id;

ALTER TABLE jobs 
    DROP COLUMN IF EXISTS is_published,
    DROP COLUMN IF EXISTS proctoring_mode;
