-- 012_applications_decision.up.sql
-- Recruiter lifecycle stage + hiring notes persisted on the application row
-- (previously frontend-only state that never reached the server).

ALTER TABLE applications
    ADD COLUMN IF NOT EXISTS stage TEXT,
    ADD COLUMN IF NOT EXISTS recruiter_notes TEXT;
