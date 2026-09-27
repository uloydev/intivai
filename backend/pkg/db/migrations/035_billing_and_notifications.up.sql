-- 035_billing_and_notifications.up.sql
-- Subscriptions & quotas on orgs + in-app recruiter notifications.

ALTER TABLE orgs
    ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT,
    ADD COLUMN IF NOT EXISTS stripe_subscription_id TEXT,
    ADD COLUMN IF NOT EXISTS plan_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS current_period_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_period_end TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS interview_credits INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS recruiter_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    action_url TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE recruiter_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE recruiter_notifications FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_recruiter_notifications ON recruiter_notifications
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

CREATE INDEX IF NOT EXISTS idx_recruiter_notifications_org_read_created
    ON recruiter_notifications (org_id, read, created_at DESC);
