-- 025_tenant_rls_new_tables.up.sql
-- Harden the P4a-era tenant tables with FORCED RLS + policies, matching the
-- 001/002 pattern (per-transaction app.org_id setting). Without this the
-- intivai_app role can read ANY org's webhook configs (incl. secrets),
-- deliveries, recruiter decisions, and GDPR data requests.

ALTER TABLE webhook_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_configs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_webhook_configs ON webhook_configs
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_webhook_deliveries ON webhook_deliveries
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE recruiter_decisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE recruiter_decisions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_recruiter_decisions ON recruiter_decisions
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE data_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_requests FORCE ROW LEVEL SECURITY;
ALTER TABLE data_requests DROP CONSTRAINT IF EXISTS data_requests_action_check;
ALTER TABLE data_requests ADD CONSTRAINT data_requests_action_check
    CHECK (action IN ('export', 'delete'));
CREATE POLICY tenant_isolation_data_requests ON data_requests
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

-- The candidate portal writes data_requests OUTSIDE any tenant context (email
-- lookup across orgs) — SECURITY DEFINER, same pattern as candidate_erase.
CREATE OR REPLACE FUNCTION log_data_request(p_email TEXT, p_action TEXT)
RETURNS void
LANGUAGE plpgsql
STRICT
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    IF p_action NOT IN ('export', 'delete') THEN
        RAISE EXCEPTION 'invalid data request action' USING ERRCODE = '23514';
    END IF;
    INSERT INTO data_requests (candidate_id, org_id, action, completed_at)
    SELECT id, org_id, p_action, NOW() FROM candidates WHERE LOWER(email) = LOWER(p_email);
END;
$$;

ALTER FUNCTION log_data_request(TEXT, TEXT) OWNER TO intivai_rls_bypass;
-- SECURITY DEFINER runs with the owner's privileges: the bypass role needs
-- INSERT on data_requests (and SELECT on candidates) to write the audit row.
GRANT SELECT ON candidates TO intivai_rls_bypass;
GRANT INSERT ON data_requests TO intivai_rls_bypass;
