-- Additions for Phase 5 Pivot: Proctoring modes, Bulk Upload batches, and Global Candidate Passports

-- Jobs: Proctoring Mode and Published flag
ALTER TABLE jobs 
    ADD COLUMN IF NOT EXISTS proctoring_mode TEXT DEFAULT 'optional',
    ADD COLUMN IF NOT EXISTS is_published BOOLEAN DEFAULT false;

-- Candidates: Batch ID for Bulk Upload and Magic Link Token for Review
ALTER TABLE candidates 
    ADD COLUMN IF NOT EXISTS batch_id UUID,
    ADD COLUMN IF NOT EXISTS review_token TEXT UNIQUE;

-- Global Candidate Passports (Cross-Tenant Moat)
-- Deliberately no org_id to allow candidates to port their verified profile across companies
CREATE TABLE IF NOT EXISTS global_candidate_passports (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    verified_profile JSONB,
    global_score DOUBLE PRECISION,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Note: RLS is NOT enabled on global_candidate_passports because it is explicitly cross-tenant.
-- Application logic will handle candidate authentication to access their passport.
