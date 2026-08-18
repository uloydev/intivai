-- Public job lookup functions: security-definer so unauthenticated candidates
-- can browse active jobs and apply without tenant auth context.

CREATE OR REPLACE FUNCTION public_active_jobs_lookup(p_org_slug TEXT DEFAULT NULL)
RETURNS TABLE(
    id UUID,
    org_id UUID,
    org_name TEXT,
    title TEXT,
    description TEXT,
    required_skills JSONB,
    min_experience INT,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT j.id, j.org_id, o.name AS org_name, j.title, j.description, j.required_skills, j.min_experience, j.status, j.created_at
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
    title TEXT,
    description TEXT,
    required_skills JSONB,
    min_experience INT,
    status TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    RETURN QUERY
    SELECT j.id, j.org_id, o.name AS org_name, j.title, j.description, j.required_skills, j.min_experience, j.status, j.created_at
    FROM jobs j
    JOIN orgs o ON o.id = j.org_id
    WHERE j.id = p_job_id AND j.status = 'active';
END;
$$;
