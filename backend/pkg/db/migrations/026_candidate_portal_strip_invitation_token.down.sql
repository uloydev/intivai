-- Restore the pre-026 candidate_applications_lookup definition verbatim
-- (011_rich_jobs_and_candidate_portal.up.sql lines 145-216), including the
-- invitation_token column.
DROP FUNCTION IF EXISTS candidate_applications_lookup(TEXT);

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

ALTER FUNCTION candidate_applications_lookup(TEXT) OWNER TO intivai_rls_bypass;
