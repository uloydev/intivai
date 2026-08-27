-- J7: question-set generation is CAS'd on jobs.updated_at (the row version the
-- enqueue payload and the store UPDATE both use). Migration 004 added the
-- column as nullable WITHOUT a default, so raw inserts (seeds, worker fixtures)
-- leave it NULL and every WHERE updated_at = $version predicate fails closed.
-- Backfill the existing rows, then make the column non-null with a default so
-- the row version is always stamped for INSERTs that don't pass one.
UPDATE jobs SET updated_at = created_at WHERE updated_at IS NULL;
ALTER TABLE jobs ALTER COLUMN updated_at SET DEFAULT NOW();
ALTER TABLE jobs ALTER COLUMN updated_at SET NOT NULL;
