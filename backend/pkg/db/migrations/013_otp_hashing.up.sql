-- 013_otp_hashing.up.sql
-- Bring existing dev DBs in line with 011's hashed-code schema: rename the
-- plaintext `code` column (011 creates `code_hash` on fresh installs).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'candidate_otps' AND column_name = 'code') THEN
        ALTER TABLE candidate_otps RENAME COLUMN code TO code_hash;
    END IF;
END $$;

ALTER TABLE candidate_otps ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
