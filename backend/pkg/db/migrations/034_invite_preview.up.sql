-- Migration 034: get_invite_preview SECURITY DEFINER function
-- Allows unauthenticated candidate invite pre-flight check without leaking PII or bypassing RLS.

CREATE OR REPLACE FUNCTION get_invite_preview(p_token TEXT)
RETURNS TABLE(
    valid BOOLEAN,
    status TEXT,
    org_name TEXT,
    job_title TEXT,
    question_count INT,
    estimated_duration_min INT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    tok interview_tokens%ROWTYPE;
    iv interviews%ROWTYPE;
    j jobs%ROWTYPE;
    o orgs%ROWTYPE;
    q_count INT := 0;
BEGIN
    SELECT * INTO tok FROM interview_tokens WHERE token = p_token;
    IF NOT FOUND THEN
        RETURN QUERY SELECT false, 'not_found'::TEXT, ''::TEXT, ''::TEXT, 0, 0;
        RETURN;
    END IF;

    IF tok.revoked_at IS NOT NULL THEN
        RETURN QUERY SELECT false, 'revoked'::TEXT, ''::TEXT, ''::TEXT, 0, 0;
        RETURN;
    ELSIF tok.expires_at < NOW() THEN
        RETURN QUERY SELECT false, 'expired'::TEXT, ''::TEXT, ''::TEXT, 0, 0;
        RETURN;
    END IF;

    SELECT * INTO iv FROM interviews WHERE id = tok.interview_id;
    IF NOT FOUND THEN
        RETURN QUERY SELECT false, 'not_found'::TEXT, ''::TEXT, ''::TEXT, 0, 0;
        RETURN;
    END IF;

    IF iv.transcript IS NOT NULL AND jsonb_typeof(iv.transcript->'questions') = 'array' THEN
        q_count := jsonb_array_length(iv.transcript->'questions');
    END IF;

    SELECT j_tbl.* INTO j FROM jobs j_tbl
    JOIN applications a ON a.job_id = j_tbl.id
    WHERE a.id = iv.application_id;

    SELECT * INTO o FROM orgs WHERE id = tok.org_id;

    RETURN QUERY SELECT
        true,
        CASE WHEN tok.used_at IS NOT NULL THEN 'used'::TEXT ELSE 'valid'::TEXT END,
        COALESCE(o.name, '')::TEXT,
        COALESCE(j.title, 'Technical Interview')::TEXT,
        q_count,
        GREATEST(15, q_count * 5);
END;
$$;

ALTER FUNCTION get_invite_preview(TEXT) OWNER TO intivai_rls_bypass;
GRANT EXECUTE ON FUNCTION get_invite_preview(TEXT) TO intivai_app;
