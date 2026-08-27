-- 033_candidate_review_name_email.up.sql
-- D1: the public confirm endpoint now takes candidate-verified name + email
-- (identity columns, not resume dimensions) alongside the structured profile.
-- Postgres cannot change a function signature with CREATE OR REPLACE, so the
-- 2-arg form is dropped first — no data loss: functions hold no state (same
-- drop/recreate pattern as 031).

DROP FUNCTION IF EXISTS candidate_confirm_review(TEXT, JSONB);

CREATE OR REPLACE FUNCTION candidate_confirm_review(p_token TEXT, p_structured JSONB, p_name TEXT, p_email TEXT)
RETURNS TABLE(org_id UUID, candidate_id UUID)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_org UUID;
    v_id  UUID;
BEGIN
    UPDATE candidates
    SET name = p_name, email = p_email, cv_structured = p_structured,
        status = 'extracted', review_token = NULL, error_message = NULL,
        updated_at = NOW()
    WHERE review_token = p_token AND status = 'pending_review'
    RETURNING candidates.org_id, candidates.id INTO v_org, v_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;
    RETURN QUERY SELECT v_org, v_id;
END;
$$;

ALTER FUNCTION candidate_confirm_review(TEXT, JSONB, TEXT, TEXT) OWNER TO intivai_rls_bypass;
