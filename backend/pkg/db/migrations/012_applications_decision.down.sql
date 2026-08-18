-- 012_applications_decision.down.sql
ALTER TABLE applications
    DROP COLUMN IF EXISTS recruiter_notes,
    DROP COLUMN IF EXISTS stage;
