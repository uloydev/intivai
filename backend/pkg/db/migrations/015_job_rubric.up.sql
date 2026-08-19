-- 015_job_rubric.up.sql

ALTER TABLE jobs 
    ADD COLUMN IF NOT EXISTS rubric JSONB;
