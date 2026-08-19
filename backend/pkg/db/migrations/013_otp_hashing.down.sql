ALTER TABLE candidate_otps DROP COLUMN IF EXISTS attempts;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'candidate_otps' AND column_name = 'code_hash') THEN
        ALTER TABLE candidate_otps RENAME COLUMN code_hash TO code;
    END IF;
END $$;
