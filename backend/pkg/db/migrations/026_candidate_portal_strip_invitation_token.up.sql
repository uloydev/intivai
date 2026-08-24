-- C9 hardening: candidate_applications_lookup exposed the live interview
-- invitation_token to the candidate-portal surface. A portal token minted for
-- an attacker-supplied email must never be able to read interview
-- credentials. Strip invitation_token from the function output; every other
-- column stays identical.
--
-- CREATE OR REPLACE cannot change a RETURNS TABLE column list, so drop +
-- recreate. golang-migrate stores no checksums; the down migration restores
-- the 011 definition verbatim.
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
    WHERE LOWER(c.email) = LOWER(p_email)
    ORDER BY a.created_at DESC;
END;
$$;

-- SECURITY DEFINER functions must be owned by the RLS-bypass role or they
-- silently return zero rows under FORCED RLS when run by a non-superuser.
ALTER FUNCTION candidate_applications_lookup(TEXT) OWNER TO intivai_rls_bypass;
