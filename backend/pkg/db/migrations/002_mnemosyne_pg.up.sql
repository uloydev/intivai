-- Mnemosyne semantic memory in Postgres (pgvector bank).
-- Production path: 1 bank = 1 org_id partition; tenant isolation via RLS,
-- same as every other org-scoped table. SQLite (native adapter) remains the
-- default dev adapter; swap via INTIVAI_MEMORY_DRIVER=postgres.

CREATE TABLE mnemosyne_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES orgs(id),
    entity_type TEXT NOT NULL,
    content TEXT NOT NULL,
    importance REAL NOT NULL DEFAULT 0.5,
    embedding VECTOR(384),          -- fastembed bge-small = 384 dims (M2)
    filter TEXT,                    -- structured filter for QueryGraph, e.g. "passed_screening=true"
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE mnemosyne_memories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_mnemosyne ON mnemosyne_memories
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

CREATE INDEX idx_mnemosyne_entity ON mnemosyne_memories(entity_type);
CREATE INDEX idx_mnemosyne_filter ON mnemosyne_memories(filter);
CREATE INDEX idx_mnemosyne_embedding ON mnemosyne_memories
    USING hnsw (embedding vector_cosine_ops);

-- ═══════════════════════════════════════════════════════════════════
-- HARDENING: FORCE RLS on every org-scoped table.
-- Without FORCE, the app DB user (table owner) bypasses RLS entirely —
-- policies would be decorative. FORCE makes isolation real even for owner.
-- Pre-auth security-definer functions move to a BYPASSRLS role so they
-- keep working without a tenant context.
-- ═══════════════════════════════════════════════════════════════════

CREATE ROLE intivai_rls_bypass BYPASSRLS;
GRANT intivai_rls_bypass TO CURRENT_USER;

ALTER FUNCTION login_lookup(TEXT, TEXT) OWNER TO intivai_rls_bypass;
ALTER FUNCTION validate_interview_token(TEXT) OWNER TO intivai_rls_bypass;

ALTER TABLE orgs FORCE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE questions FORCE ROW LEVEL SECURITY;
ALTER TABLE jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE candidates FORCE ROW LEVEL SECURITY;
ALTER TABLE applications FORCE ROW LEVEL SECURITY;
ALTER TABLE interviews FORCE ROW LEVEL SECURITY;
ALTER TABLE interview_tokens FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
ALTER TABLE company_contexts FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_prompts FORCE ROW LEVEL SECURITY;
ALTER TABLE mnemosyne_memories FORCE ROW LEVEL SECURITY;

-- ═══════════════════════════════════════════════════════════════════
-- APPLICATION ROLE: least privilege.
-- The app must NOT connect as superuser/owner — superusers bypass RLS
-- no matter what (FORCE included). This role is a plain table user:
-- RLS policies apply to it automatically.
-- Migrations run separately as the admin user (INTIVAI_MIGRATE_URL).
-- ═══════════════════════════════════════════════════════════════════

CREATE ROLE intivai_app LOGIN PASSWORD 'intivai_app';
DO $$ BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO intivai_app', current_database());
END $$;
GRANT USAGE ON SCHEMA public TO intivai_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO intivai_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO intivai_app;
ALTER DEFAULT PRIVILEGES FOR ROLE intivai IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO intivai_app;
ALTER DEFAULT PRIVILEGES FOR ROLE intivai IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO intivai_app;

-- Security-definer lookups run with the bypass role's privileges: it needs
-- read access to the tables it queries (login_lookup, validate_interview_token).
GRANT SELECT ON ALL TABLES IN SCHEMA public TO intivai_rls_bypass;
ALTER DEFAULT PRIVILEGES FOR ROLE intivai IN SCHEMA public
    GRANT SELECT ON TABLES TO intivai_rls_bypass;

