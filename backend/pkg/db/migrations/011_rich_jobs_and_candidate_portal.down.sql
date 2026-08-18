-- 011_rich_jobs_and_candidate_portal.down.sql
DROP FUNCTION IF EXISTS candidate_applications_lookup(TEXT);
DROP FUNCTION IF EXISTS public_job_detail_lookup(UUID);
DROP FUNCTION IF EXISTS public_active_jobs_lookup(TEXT);
DROP TABLE IF EXISTS candidate_otps;
ALTER TABLE jobs
    DROP COLUMN IF EXISTS benefits,
    DROP COLUMN IF EXISTS nice_to_haves,
    DROP COLUMN IF EXISTS requirements,
    DROP COLUMN IF EXISTS responsibilities,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS salary_max,
    DROP COLUMN IF EXISTS salary_min,
    DROP COLUMN IF EXISTS employment_type,
    DROP COLUMN IF EXISTS location;
