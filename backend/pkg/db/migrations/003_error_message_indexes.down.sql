DROP INDEX IF EXISTS idx_applications_candidate;
DROP INDEX IF EXISTS idx_applications_job;
DROP INDEX IF EXISTS idx_applications_org;
DROP INDEX IF EXISTS idx_candidates_org_status;
ALTER TABLE applications DROP CONSTRAINT IF EXISTS applications_candidate_job_key;
ALTER TABLE candidates DROP COLUMN IF EXISTS error_message;
