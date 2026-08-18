-- 01_orgs_and_users.sql
-- Seed Demo Organization and Initial Admin/Recruiter Users

DO $$
DECLARE
    v_demo_id UUID;
BEGIN
    SELECT id INTO v_demo_id FROM orgs WHERE slug = 'demo';
    IF v_demo_id IS NOT NULL THEN
        DELETE FROM audit_logs WHERE org_id = v_demo_id;
        DELETE FROM mnemosyne_memories WHERE org_id = v_demo_id;
        DELETE FROM interview_tokens WHERE org_id = v_demo_id;
        DELETE FROM interviews WHERE application_id IN (SELECT id FROM applications WHERE org_id = v_demo_id);
        DELETE FROM applications WHERE org_id = v_demo_id;
        DELETE FROM candidates WHERE org_id = v_demo_id;
        DELETE FROM jobs WHERE org_id = v_demo_id;
        DELETE FROM questions WHERE org_id = v_demo_id;
        DELETE FROM company_contexts WHERE org_id = v_demo_id;
        DELETE FROM tenant_prompts WHERE org_id = v_demo_id;
        DELETE FROM users WHERE org_id = v_demo_id;
        DELETE FROM orgs WHERE id = v_demo_id;
    END IF;
END $$;

INSERT INTO orgs (id, name, slug, plan, scoring_weights, min_score_to_proceed, created_at)
VALUES (
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Demo Corp',
    'demo',
    'enterprise',
    '{"skills_match": 0.4, "experience_years": 0.3, "semantic_match": 0.2, "education": 0.1}',
    60.0,
    NOW() - INTERVAL '30 days'
);

-- Admin User: admin@demo.io (password: password123)
INSERT INTO users (id, org_id, email, role, password_hash, auth_provider, created_at)
VALUES (
    '38647293-4a4e-4060-b6f5-682bbc4cc467',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'admin@demo.io',
    'admin',
    '$2b$12$6A8o7PlDff98KsylcEd/0OKwF6hrsTsohPrFvPZVy.psTOpICZaaK',
    'password',
    NOW() - INTERVAL '30 days'
)
ON CONFLICT (org_id, email) DO UPDATE SET
    role = EXCLUDED.role,
    password_hash = EXCLUDED.password_hash;

-- Recruiter User: recruiter@demo.io (password: password123)
INSERT INTO users (id, org_id, email, role, password_hash, auth_provider, created_at)
VALUES (
    'a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'recruiter@demo.io',
    'recruiter',
    '$2b$12$6A8o7PlDff98KsylcEd/0OKwF6hrsTsohPrFvPZVy.psTOpICZaaK',
    'password',
    NOW() - INTERVAL '25 days'
)
ON CONFLICT (org_id, email) DO UPDATE SET
    role = EXCLUDED.role,
    password_hash = EXCLUDED.password_hash;
