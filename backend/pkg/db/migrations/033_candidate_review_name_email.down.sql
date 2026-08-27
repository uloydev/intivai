-- 033_candidate_review_name_email.down.sql
-- Restore the 2-arg candidate_confirm_review (pre-name/email), exactly as in
-- migration 019. Functions hold no state, so the drop/recreate loses nothing.

DROP FUNCTION IF EXISTS candidate_confirm_review(TEXT, JSONB, TEXT, TEXT);

CREATE OR REPLACE FUNCTION candidate_confirm_review(p_token TEXT, p_structured JSONB)
RETURNS TABLE(org_id UUID, candidate_id UUID)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_org UUID;
    v_id  UUID;
BEGIN
    UPDATE candidates
    SET cv_structured = p_structured, status = 'extracted',
        review_token = NULL, error_message = NULL, updated_at = NOW()
    WHERE review_token = p_token AND status = 'pending_review'
    RETURNING candidates.org_id, candidates.id INTO v_org, v_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;
    RETURN QUERY SELECT v_org, v_id;
END;
$$;

ALTER FUNCTION candidate_confirm_review(TEXT, JSONB) OWNER TO intivai_rls_bypass;
