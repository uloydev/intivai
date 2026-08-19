-- 017_fix_public_job_published.down.sql

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
    proctoring_mode TEXT,
    is_published BOOLEAN,
    rubric JSONB,
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
        COALESCE(j.proctoring_mode, 'optional') AS proctoring_mode,
        COALESCE(j.is_published, false) AS is_published,
        j.rubric,
        j.created_at
    FROM jobs j
    JOIN orgs o ON o.id = j.org_id
    WHERE j.status = 'active'
      AND (p_org_slug IS NULL OR p_org_slug = '' OR o.slug = p_org_slug)
    ORDER BY j.created_at DESC;
END;
$$;

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
    proctoring_mode TEXT,
    is_published BOOLEAN,
    rubric JSONB,
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
        COALESCE(j.proctoring_mode, 'optional') AS proctoring_mode,
        COALESCE(j.is_published, false) AS is_published,
        j.rubric,
        j.created_at
    FROM jobs j
    JOIN orgs o ON o.id = j.org_id
    WHERE j.id = p_job_id AND j.status = 'active';
END;
$$;

ALTER FUNCTION public_active_jobs_lookup(TEXT) OWNER TO intivai_rls_bypass;
ALTER FUNCTION public_job_detail_lookup(UUID) OWNER TO intivai_rls_bypass;
