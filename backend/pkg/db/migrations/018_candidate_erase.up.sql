-- 018_candidate_erase.up.sql
-- GDPR right-to-erasure for the candidate portal: deletes every trace of a
-- candidate (identified by email) across ALL orgs — portal OTPs, applications
-- (cascades to interviews + scorecards via FK), and candidate rows.
-- SECURITY DEFINER + intivai_rls_bypass owner: the app user is RLS-bound to
-- one org and cannot delete cross-tenant rows.

CREATE OR REPLACE FUNCTION candidate_erase(p_email TEXT)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_candidate_ids UUID[];
    v_app_ids UUID[];
BEGIN
    SELECT ARRAY_AGG(id) INTO v_candidate_ids
    FROM candidates WHERE LOWER(email) = LOWER(p_email);

    IF v_candidate_ids IS NOT NULL THEN
        SELECT ARRAY_AGG(id) INTO v_app_ids
        FROM applications WHERE candidate_id = ANY(v_candidate_ids);

        IF v_app_ids IS NOT NULL THEN
            DELETE FROM interviews WHERE application_id = ANY(v_app_ids);
        END IF;
        DELETE FROM applications WHERE candidate_id = ANY(v_candidate_ids);
        DELETE FROM candidates WHERE id = ANY(v_candidate_ids);
    END IF;

    DELETE FROM candidate_otps WHERE LOWER(email) = LOWER(p_email);
END;
$$;

ALTER FUNCTION candidate_erase(TEXT) OWNER TO intivai_rls_bypass;
