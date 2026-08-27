-- 031_candidate_review_cv_format.up.sql
-- Regression fix: candidate_by_review_token did not expose cv_format while
-- postgres_candidate_repo.GetByReviewToken selected COALESCE(cv_format,'pdf')
-- from the function output → plan-time 42703 on every public review lookup,
-- surfacing as a generic 500 ("Review Link Invalid or Expired" in the FE).
-- Adds cv_format so the function matches the repo SELECT (same columns/order
-- as GetByID/List over the base table).
-- Postgres cannot change an existing function's return type via CREATE OR
-- REPLACE, so drop first. No data loss: functions hold no state.

DROP FUNCTION IF EXISTS candidate_by_review_token(TEXT);

CREATE OR REPLACE FUNCTION candidate_by_review_token(p_token TEXT)
RETURNS TABLE(
    id UUID, org_id UUID, name TEXT, email TEXT, cv_path TEXT, cv_format TEXT,
    cv_raw_text TEXT, cv_structured JSONB, cv_ocr_method TEXT,
    status TEXT, error_message TEXT, batch_id TEXT, review_token TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE sql SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id, org_id, name, email, cv_path, cv_format, cv_raw_text, cv_structured,
           cv_ocr_method, status, error_message, batch_id, review_token, created_at
    FROM candidates
    WHERE review_token = p_token AND status = 'pending_review';
$$;

ALTER FUNCTION candidate_by_review_token(TEXT) OWNER TO intivai_rls_bypass;
