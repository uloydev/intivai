-- 019_candidate_review.down.sql
REVOKE SELECT, UPDATE, DELETE ON candidates FROM intivai_rls_bypass;
REVOKE SELECT, DELETE ON applications FROM intivai_rls_bypass;
REVOKE SELECT, DELETE ON interviews FROM intivai_rls_bypass;
REVOKE SELECT, DELETE ON candidate_otps FROM intivai_rls_bypass;

DROP FUNCTION IF EXISTS candidate_confirm_review(TEXT, JSONB);
DROP FUNCTION IF EXISTS candidate_by_review_token(TEXT);
