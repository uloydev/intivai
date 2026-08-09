ALTER TABLE company_contexts DROP COLUMN IF EXISTS updated_at;
ALTER TABLE applications DROP COLUMN IF EXISTS updated_at;
ALTER TABLE jobs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE candidates DROP COLUMN IF EXISTS updated_at;
