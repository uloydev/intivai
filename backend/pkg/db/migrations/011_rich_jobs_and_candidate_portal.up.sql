-- 1. Extend jobs table with rich job board specs
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS location TEXT DEFAULT 'Remote',
    ADD COLUMN IF NOT EXISTS employment_type TEXT DEFAULT 'Full-time',
    ADD COLUMN IF NOT EXISTS salary_min INT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS salary_max INT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS currency TEXT DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS responsibilities JSONB DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS requirements JSONB DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS nice_to_haves JSONB DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS benefits JSONB DEFAULT '[]';

-- 2. Candidate OTP / Magic Link passwordless authentication table
CREATE TABLE IF NOT EXISTS candidate_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_candidate_otps_email_code ON candidate_otps(email, code_hash);
CREATE INDEX IF NOT EXISTS idx_candidate_otps_token ON candidate_otps(token);

-- Drop previous functions before replacing return types
DROP FUNCTION IF EXISTS public_active_jobs_lookup(TEXT);
DROP FUNCTION IF EXISTS public_job_detail_lookup(UUID);

-- 3. Replace public_active_jobs_lookup to return enriched fields & org branding
CREATE OR REPLACE FUNCTION public_active_jobs_lookup(p_org_slug TEXT DEFAULT NULL)
RETURNS TABLE(
    id UUID,
    org_id UUID,
    org_name TEXT,
    org_slug TEXT,
    title TEXT,
    description TEXT,
    location TEXT,
    employment_type TEXT,
    salary_min INT,
    salary_max INT,
    currency TEXT,
    required_skills JSONB,
    min_experience INT,
    responsibilities JSONB,
    requirements JSONB,
    nice_to_haves JSONB,
    benefits JSONB,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        j.id, 
        j.org_id, 
        o.name AS org_name, 
        o.slug AS org_slug,
        j.title, 
        j.description,
        COALESCE(j.location, 'Remote') AS location,
        COALESCE(j.employment_type, 'Full-time') AS employment_type,
        j.salary_min,
        j.salary_max,
        COALESCE(j.currency, 'USD') AS currency,
        COALESCE(j.required_skills, '[]'::jsonb) AS required_skills, 
        COALESCE(j.min_experience, 0) AS min_experience,
        COALESCE(j.responsibilities, '[]'::jsonb) AS responsibilities,
        COALESCE(j.requirements, '[]'::jsonb) AS requirements,
        COALESCE(j.nice_to_haves, '[]'::jsonb) AS nice_to_haves,
        COALESCE(j.benefits, '[]'::jsonb) AS benefits,
        j.status, 
        j.created_at
    FROM jobs j
    JOIN orgs o ON o.id = j.org_id
    WHERE j.status = 'active'
      AND (p_org_slug IS NULL OR p_org_slug = '' OR o.slug = p_org_slug)
    ORDER BY j.created_at DESC;
END;
$$;

-- 4. Replace public_job_detail_lookup
CREATE OR REPLACE FUNCTION public_job_detail_lookup(p_job_id UUID)
RETURNS TABLE(
    id UUID,
    org_id UUID,
    org_name TEXT,
    org_slug TEXT,
    title TEXT,
    description TEXT,
    location TEXT,
    employment_type TEXT,
    salary_min INT,
    salary_max INT,
    currency TEXT,
    required_skills JSONB,
    min_experience INT,
    responsibilities JSONB,
    requirements JSONB,
    nice_to_haves JSONB,
    benefits JSONB,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        j.id, 
        j.org_id, 
        o.name AS org_name, 
        o.slug AS org_slug,
        j.title, 
        j.description,
        COALESCE(j.location, 'Remote') AS location,
        COALESCE(j.employment_type, 'Full-time') AS employment_type,
        j.salary_min,
        j.salary_max,
        COALESCE(j.currency, 'USD') AS currency,
        COALESCE(j.required_skills, '[]'::jsonb) AS required_skills, 
        COALESCE(j.min_experience, 0) AS min_experience,
        COALESCE(j.responsibilities, '[]'::jsonb) AS responsibilities,
        COALESCE(j.requirements, '[]'::jsonb) AS requirements,
        COALESCE(j.nice_to_haves, '[]'::jsonb) AS nice_to_haves,
        COALESCE(j.benefits, '[]'::jsonb) AS benefits,
        j.status, 
        j.created_at
    FROM jobs j
    JOIN orgs o ON o.id = j.org_id
    WHERE j.id = p_job_id AND j.status = 'active';
END;
$$;

-- 5. Candidate cross-tenant application tracker function
CREATE OR REPLACE FUNCTION candidate_applications_lookup(p_email TEXT)
RETURNS TABLE(
    application_id UUID,
    org_id UUID,
    org_name TEXT,
    org_slug TEXT,
    job_id UUID,
    job_title TEXT,
    job_location TEXT,
    job_employment_type TEXT,
    candidate_id UUID,
    candidate_name TEXT,
    candidate_email TEXT,
    cv_score DOUBLE PRECISION,
    passed_screening BOOLEAN,
    application_status TEXT,
    applied_at TIMESTAMPTZ,
    interview_id UUID,
    interview_status TEXT,
    interview_type TEXT,
    invitation_token TEXT,
    overall_score DOUBLE PRECISION,
    recommendation TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.id AS application_id,
        a.org_id,
        o.name AS org_name,
        o.slug AS org_slug,
        j.id AS job_id,
        j.title AS job_title,
        COALESCE(j.location, 'Remote') AS job_location,
        COALESCE(j.employment_type, 'Full-time') AS job_employment_type,
        c.id AS candidate_id,
        c.name AS candidate_name,
        c.email AS candidate_email,
        a.cv_score,
        a.passed_screening,
        a.status AS application_status,
        a.created_at AS applied_at,
        i.id AS interview_id,
        i.status AS interview_status,
        i.type AS interview_type,
        it.token AS invitation_token,
        COALESCE(
            NULLIF((i.evaluation->>'overall_score')::double precision, NULL),
            NULL
        ) AS overall_score,
        COALESCE(i.evaluation->>'recommendation', '') AS recommendation
    FROM candidates c
    JOIN applications a ON a.candidate_id = c.id
    JOIN jobs j ON j.id = a.job_id
    JOIN orgs o ON o.id = a.org_id
    LEFT JOIN interviews i ON i.application_id = a.id
    LEFT JOIN interview_tokens it ON it.interview_id = i.id AND it.expires_at > NOW()
    WHERE LOWER(c.email) = LOWER(p_email)
    ORDER BY a.created_at DESC;
END;
$$;

-- SECURITY DEFINER functions must be owned by the RLS-bypass role (same
-- pattern as login_lookup/validate_interview_token in 002) or they silently
-- return zero rows under FORCED RLS when run by a non-superuser.
ALTER FUNCTION public_active_jobs_lookup(TEXT) OWNER TO intivai_rls_bypass;
ALTER FUNCTION public_job_detail_lookup(UUID) OWNER TO intivai_rls_bypass;
ALTER FUNCTION candidate_applications_lookup(TEXT) OWNER TO intivai_rls_bypass;
