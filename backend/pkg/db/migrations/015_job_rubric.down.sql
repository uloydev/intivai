-- 015_job_rubric.down.sql

ALTER TABLE jobs 
    DROP COLUMN IF EXISTS rubric;
