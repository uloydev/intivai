CREATE TABLE IF NOT EXISTS recruiter_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    original_recommendation TEXT NOT NULL,
    override_recommendation TEXT NOT NULL,
    reason TEXT NOT NULL,
    decided_by UUID NOT NULL REFERENCES users(id),
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recruiter_decisions_interview ON recruiter_decisions(interview_id);
