-- 027_job_candidate_context.up.sql
-- B3: per-job candidate Q&A context (D2). One row per job, merged into the
-- interview system prompt alongside the org company context and version-pinned
-- at connect. Owned by the job's org so FORCED RLS (002 pattern) isolates it.

CREATE TABLE job_candidate_contexts (
    job_id     UUID PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    version    INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_candidate_contexts_org ON job_candidate_contexts (org_id);

ALTER TABLE job_candidate_contexts ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_candidate_contexts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_job_candidate_contexts ON job_candidate_contexts
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);
