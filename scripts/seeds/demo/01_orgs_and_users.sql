-- 01_orgs_and_users.sql
-- Seed Demo Organization, Initial Admin/Recruiter Users, and Curated Question Bank

DO $$
DECLARE
    v_demo_id UUID;
BEGIN
    SELECT id INTO v_demo_id FROM orgs WHERE slug = 'demo';
    IF v_demo_id IS NOT NULL THEN
        DELETE FROM candidate_otps WHERE email IN (
            'alex.rivera@example.com',
            'elena.rostova@example.com',
            'david.chen@example.com',
            'marcus.vance@example.com'
        );
        DELETE FROM global_candidate_passports WHERE email IN (
            'alex.rivera@example.com',
            'elena.rostova@example.com',
            'david.chen@example.com',
            'marcus.vance@example.com'
        );
        DELETE FROM audit_logs WHERE org_id = v_demo_id;
        DELETE FROM recruiter_notifications WHERE org_id = v_demo_id;
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

INSERT INTO orgs (id, name, slug, plan, plan_status, current_period_start, current_period_end, interview_credits, scoring_weights, min_score_to_proceed, created_at)
VALUES (
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Demo Corp',
    'demo',
    'enterprise',
    'active',
    NOW() - INTERVAL '15 days',
    NOW() + INTERVAL '15 days',
    500,
    '{"skills_match": 0.35, "experience_years": 0.20, "semantic_match": 0.25, "education": 0.10, "certifications": 0.10}'::jsonb,
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

-- Reusable Question Bank for Demo Corp
INSERT INTO questions (id, org_id, category, difficulty, body, skills, created_at)
VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        '968f66ef-91c6-4db3-8764-ceeffb753b1f',
        'technical',
        'hard',
        'Can you describe a challenging distributed concurrency or race condition issue you diagnosed in Go, and how you resolved it?',
        ARRAY['Go', 'Concurrency'],
        NOW() - INTERVAL '20 days'
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        '968f66ef-91c6-4db3-8764-ceeffb753b1f',
        'technical',
        'medium',
        'How do you enforce PostgreSQL tenant isolation and transaction safety when handling asynchronous background tasks?',
        ARRAY['PostgreSQL', 'Multi-tenancy'],
        NOW() - INTERVAL '20 days'
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        '968f66ef-91c6-4db3-8764-ceeffb753b1f',
        'technical',
        'hard',
        'How do you architect a frontend streaming client with WebSocket reconnects, state reconciliation, and zero UI latency?',
        ARRAY['React', 'WebSockets'],
        NOW() - INTERVAL '20 days'
    ),
    (
        '44444444-4444-4444-4444-444444444444',
        '968f66ef-91c6-4db3-8764-ceeffb753b1f',
        'problem_solving',
        'medium',
        'Tell me about a time you had to make a critical architectural trade-off under strict delivery constraints.',
        ARRAY['Architecture', 'Trade-offs'],
        NOW() - INTERVAL '20 days'
    ),
    (
        '55555555-5555-5555-5555-555555555555',
        '968f66ef-91c6-4db3-8764-ceeffb753b1f',
        'culture_fit',
        'medium',
        'How do you establish engineering excellence and mentor team members when adopting new technologies?',
        ARRAY['Leadership', 'Mentorship'],
        NOW() - INTERVAL '20 days'
    )
ON CONFLICT (id) DO NOTHING;
