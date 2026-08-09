ALTER TABLE candidates ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE applications ADD CONSTRAINT applications_candidate_job_key UNIQUE (candidate_id, job_id);

CREATE INDEX IF NOT EXISTS idx_candidates_org_status ON candidates(org_id, status);
CREATE INDEX IF NOT EXISTS idx_applications_org ON applications(org_id);
CREATE INDEX IF NOT EXISTS idx_applications_job ON applications(job_id);
CREATE INDEX IF NOT EXISTS idx_applications_candidate ON applications(candidate_id);
