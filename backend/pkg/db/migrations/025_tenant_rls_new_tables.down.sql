-- 025_tenant_rls_new_tables.down.sql
DROP FUNCTION IF EXISTS log_data_request(TEXT, TEXT);
ALTER TABLE data_requests DROP CONSTRAINT IF EXISTS data_requests_action_check;
ALTER TABLE data_requests ADD CONSTRAINT data_requests_action_check
    CHECK (action IN ('export', 'delete'));
DROP POLICY IF EXISTS tenant_isolation_data_requests ON data_requests;
ALTER TABLE data_requests NO FORCE ROW LEVEL SECURITY;
ALTER TABLE data_requests DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_recruiter_decisions ON recruiter_decisions;
ALTER TABLE recruiter_decisions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE recruiter_decisions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_webhook_deliveries ON webhook_deliveries;
ALTER TABLE webhook_deliveries NO FORCE ROW LEVEL SECURITY;
ALTER TABLE webhook_deliveries DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_webhook_configs ON webhook_configs;
ALTER TABLE webhook_configs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE webhook_configs DISABLE ROW LEVEL SECURITY;
