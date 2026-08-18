-- 02_company_intelligence.sql
-- Seed Tenant AI Prompt Rails and Company Context Knowledge

-- Tenant System Prompt Rails
INSERT INTO tenant_prompts (id, org_id, system_prompt, version, created_at)
VALUES (
    'a1c2e3f4-a5b6-4c7d-8e9f-0a1b2c3d4e5f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'You are a Principal Technical Interviewer at Demo Corp. Ask rigorous, probing questions on concurrency, race conditions, schema migrations, and real-time streaming architectures. Evaluate depth, trade-offs, and communication clarity.',
    1,
    NOW() - INTERVAL '20 days'
)
ON CONFLICT (org_id, version) DO NOTHING;

-- Vectorized Company Context Documents
INSERT INTO company_contexts (id, org_id, type, content_hash, version, storage_path, created_at, updated_at)
VALUES (
    'b2c3d4e5-f6a1-4b2c-8d3e-4f5a6b7c8d9e',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'text',
    'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    1,
    'contexts/968f66ef-91c6-4db3-8764-ceeffb753b1f/culture_handbook.txt',
    NOW() - INTERVAL '20 days',
    NOW() - INTERVAL '20 days'
)
ON CONFLICT (id) DO NOTHING;
