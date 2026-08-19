-- 019_candidate_review.up.sql
-- Public candidate-review flow (email magic link): the review endpoints run
-- WITHOUT tenant middleware, and candidates is RLS-forced — a cross-org token
-- lookup can only work through SECURITY DEFINER functions owned by the
-- RLS-bypass role (same pattern as candidate_applications_lookup/018).

-- View: the pending_review candidate for a review token (cross-org).
CREATE OR REPLACE FUNCTION candidate_by_review_token(p_token TEXT)
RETURNS TABLE(
    id UUID, org_id UUID, name TEXT, email TEXT, cv_path TEXT,
    cv_raw_text TEXT, cv_structured JSONB, cv_ocr_method TEXT,
    status TEXT, error_message TEXT, batch_id TEXT, review_token TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE sql SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
           cv_ocr_method, status, error_message, batch_id, review_token, created_at
    FROM candidates
    WHERE review_token = p_token AND status = 'pending_review';
$$;

-- Confirm: atomically mark extracted + clear the token; returns the
-- candidate's org + id so the app can enqueue scoring inside a tenant tx.
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

ALTER FUNCTION candidate_by_review_token(TEXT) OWNER TO intivai_rls_bypass;
ALTER FUNCTION candidate_confirm_review(TEXT, JSONB) OWNER TO intivai_rls_bypass;

-- The bypass role has no table grants yet: 018's candidate_erase needs
-- DELETE and 019's confirm needs UPDATE (SECURITY DEFINER runs with the
-- owner's privileges, so missing grants fail at runtime).
GRANT SELECT, UPDATE, DELETE ON candidates TO intivai_rls_bypass;
GRANT SELECT, DELETE ON applications TO intivai_rls_bypass;
GRANT SELECT, DELETE ON interviews TO intivai_rls_bypass;
GRANT SELECT, DELETE ON candidate_otps TO intivai_rls_bypass;
