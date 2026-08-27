-- 031_candidate_review_cv_format.down.sql
-- Restore the 13-column candidate_by_review_token signature (pre-cv_format).

DROP FUNCTION IF EXISTS candidate_by_review_token(TEXT);

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

ALTER FUNCTION candidate_by_review_token(TEXT) OWNER TO intivai_rls_bypass;
