-- Revert J7 row-version guarantee: restore the nullable pre-default state.
ALTER TABLE jobs ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE jobs ALTER COLUMN updated_at DROP DEFAULT;
