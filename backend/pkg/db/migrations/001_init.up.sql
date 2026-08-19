CREATE EXTENSION IF NOT EXISTS vector;

-- IDs: `id UUID PRIMARY KEY`; new tables default to gen_random_uuid().
-- Organizations
CREATE TABLE orgs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    plan TEXT DEFAULT 'free',
    scoring_weights JSONB,  -- per-tenant configurable weights (partial override; falls back to global defaults)
    min_score_to_proceed DOUBLE PRECISION,  -- per-tenant threshold override (default 50 via code)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Users (UNIQUE per org_id, not global)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    email TEXT NOT NULL,
    role TEXT DEFAULT 'member',  -- admin, recruiter, interviewer
    password_hash TEXT,  -- NULL if OAuth-only (Google)
    auth_provider TEXT DEFAULT 'password',  -- password | google
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, email)  -- same email allowed in different orgs
);

-- Question bank: generated questions persisted automatically (reuse + audit; seeds the next generation)
CREATE TABLE questions (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    category TEXT NOT NULL,  -- technical, behavioral, situational
    difficulty TEXT DEFAULT 'medium',  -- junior, mid, senior
    body TEXT NOT NULL,
    skills TEXT[],  -- related skills
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Job Descriptions
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    required_skills JSONB,
    min_experience INT,
    scoring_weights JSONB,  -- per-job override (optional; falls back to org → global)
    min_score_to_proceed DOUBLE PRECISION DEFAULT 50,
    embedding VECTOR(384),  -- fastembed bge-small = 384 dims
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Candidates
CREATE TABLE candidates (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    name TEXT NOT NULL,
    email TEXT,
    cv_path TEXT,
    cv_raw_text TEXT,
    cv_structured JSONB,
    cv_embedding VECTOR(384),
    cv_ocr_method TEXT,  -- pdfcpu, tesseract, ollama-vision
    status TEXT DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Applications
CREATE TABLE applications (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    candidate_id UUID REFERENCES candidates(id),
    job_id UUID REFERENCES jobs(id),
    cv_score DOUBLE PRECISION,
    score_breakdown JSONB,
    passed_screening BOOLEAN,
    status TEXT DEFAULT 'screening',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Interview Sessions
CREATE TABLE interviews (
    id UUID PRIMARY KEY,
    application_id UUID REFERENCES applications(id),
    type TEXT DEFAULT 'chat',
    status TEXT DEFAULT 'pending',
    transcript JSONB,
    evaluation JSONB,
    recording_url TEXT,
    consent_given BOOLEAN DEFAULT false,
    last_question_idx INT DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,  -- max duration deadline
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Interview invitation token — invitation credential (1x START), not "single-use raw"
CREATE TABLE interview_tokens (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),        -- tenant: RLS + revoke
    interview_id UUID REFERENCES interviews(id),
    token TEXT UNIQUE NOT NULL,             -- random 32-char, high entropy
    expires_at TIMESTAMPTZ NOT NULL,        -- default +7 days
    used_at TIMESTAMPTZ,                    -- set when the interview FIRST starts
    revoked_at TIMESTAMPTZ,                 -- recruiter revoke → all access denied
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit Log
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,     -- view_candidate, start_interview, export_data
    resource_type TEXT,
    resource_id UUID,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Company context (versioned)
CREATE TABLE company_contexts (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    type TEXT NOT NULL,          -- file | text
    content_hash TEXT NOT NULL,  -- dedup + change detection
    version INT NOT NULL DEFAULT 1,
    storage_path TEXT,           -- MinIO path (file) / NULL (text)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, version)
);

-- Tenant system prompt (versioned)
CREATE TABLE tenant_prompts (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    system_prompt TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, version)
);

-- ═══════════════════════════════════════════════════════
-- ROW-LEVEL SECURITY (RLS) — multi-tenant isolation
-- Mandatory: enabled on EVERY table with org_id
-- ═══════════════════════════════════════════════════════
-- Each request sets app.org_id via the tenant middleware:
--   SELECT set_config('app.org_id', $1, true);  -- per-transaction

ALTER TABLE orgs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_orgs ON orgs
    USING (id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_users ON users
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE questions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_questions ON questions
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_jobs ON jobs
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE candidates ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_candidates ON candidates
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE applications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_applications ON applications
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

-- interviews have no direct org_id → via join application → job
ALTER TABLE interviews ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_interviews ON interviews
    USING (EXISTS (
        SELECT 1 FROM applications a
        JOIN jobs j ON j.id = a.job_id
        WHERE a.id = interviews.application_id
          AND j.org_id = NULLIF(current_setting('app.org_id', true), '')::uuid
    ));

ALTER TABLE interview_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_tokens ON interview_tokens
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);
-- Candidate token validation (no auth): via security-definer function validate_interview_token(token),
-- returns valid/expired/used/revoked — NOT direct table access

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_audit ON audit_logs
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE company_contexts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_contexts ON company_contexts
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE tenant_prompts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_prompts ON tenant_prompts
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

-- Pre-auth login lookup: security-definer so login works WITHOUT a tenant
-- context (RLS would otherwise hide all rows before authentication).
CREATE OR REPLACE FUNCTION login_lookup(p_slug TEXT, p_email TEXT)
RETURNS TABLE(
    org_id UUID,
    user_id UUID,
    email TEXT,
    password_hash TEXT,
    role TEXT,
    auth_provider TEXT,
    created_at TIMESTAMPTZ
) LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    RETURN QUERY
    SELECT u.org_id, u.id, u.email, u.password_hash, u.role::TEXT, u.auth_provider, u.created_at
    FROM users u
    JOIN orgs o ON o.id = u.org_id
    WHERE o.slug = p_slug AND LOWER(u.email) = LOWER(p_email);
END $$;

-- Candidate token validation (M3): security-definer so candidates never touch tables directly.
CREATE OR REPLACE FUNCTION validate_interview_token(p_token TEXT)
RETURNS TABLE(
    token_id UUID,
    interview_id UUID,
    org_id UUID,
    status TEXT  -- valid | expired | used | revoked | not_found
) LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
    rec interview_tokens%ROWTYPE;
BEGIN
    SELECT * INTO rec FROM interview_tokens WHERE token = p_token;
    IF NOT FOUND THEN
        RETURN QUERY SELECT NULL::UUID, NULL::UUID, NULL::UUID, 'not_found'::TEXT;
        RETURN;
    END IF;
    IF rec.revoked_at IS NOT NULL THEN
        RETURN QUERY SELECT rec.id, rec.interview_id, rec.org_id, 'revoked'::TEXT;
    ELSIF rec.expires_at < NOW() THEN
        RETURN QUERY SELECT rec.id, rec.interview_id, rec.org_id, 'expired'::TEXT;
    ELSIF rec.used_at IS NOT NULL THEN
        RETURN QUERY SELECT rec.id, rec.interview_id, rec.org_id, 'used'::TEXT;
    ELSE
        RETURN QUERY SELECT rec.id, rec.interview_id, rec.org_id, 'valid'::TEXT;
    END IF;
END $$;
