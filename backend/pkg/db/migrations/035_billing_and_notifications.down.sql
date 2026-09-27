DROP TABLE IF EXISTS recruiter_notifications;

ALTER TABLE orgs
    DROP COLUMN IF EXISTS stripe_customer_id,
    DROP COLUMN IF EXISTS stripe_subscription_id,
    DROP COLUMN IF EXISTS plan_status,
    DROP COLUMN IF EXISTS current_period_start,
    DROP COLUMN IF EXISTS current_period_end,
    DROP COLUMN IF EXISTS interview_credits;
