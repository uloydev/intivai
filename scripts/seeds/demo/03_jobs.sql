-- 03_jobs.sql
-- Seed Core Engineering Job Requisitions

-- Job 1: Senior Distributed Systems Engineer
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published
)
VALUES (
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Senior Distributed Systems Engineer',
    'Lead the design of high-throughput distributed microservices, event streaming pipelines, and fault-tolerant consensus mechanisms in Go and PostgreSQL.',
    'Remote (US / EU)',
    'Full-time',
    160000,
    210000,
    'USD',
    '["Go", "PostgreSQL", "Distributed Systems", "Docker", "Kubernetes", "Redis"]'::jsonb,
    5,
    '["Architect and maintain core distributed microservices handling millions of transactions.", "Design fault-tolerant event sourcing and async queue processing with Asynq and Redis.", "Optimize PostgreSQL queries, connection pooling, and multi-tenant RLS isolation.", "Lead technical design reviews and establish engineering standards across the team."]'::jsonb,
    '["5+ years of production experience building distributed systems in Go.", "Deep expertise in PostgreSQL (indexing, concurrency control, transaction isolation).", "Hands-on experience with containerization (Docker), orchestration (K8s), and CI/CD.", "Strong background in API design, gRPC, WebSockets, and distributed telemetry."]'::jsonb,
    '["Experience with WebRTC signaling or real-time audio streaming.", "Familiarity with pgvector and semantic search embeddings."]'::jsonb,
    '["Competitive salary & equity options", "100% remote flexibility with home office stipend", "Comprehensive health, dental, and vision insurance", "Unlimited PTO and annual learning budget ($3,000)"]'::jsonb,
    '{"skills_match": 0.4, "experience_years": 0.25, "semantic_match": 0.25, "education": 0.1}'::jsonb,
    60.0,
    'active',
    NOW() - INTERVAL '15 days',
    NOW() - INTERVAL '15 days',
    'optional',
    true
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published;

-- Job 2: Staff Frontend Architect
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published
)
VALUES (
    'd4e5f6a1-b2c3-4d4e-8f5a-6b7c8d9e0f1a',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Staff Frontend Architect',
    'Architect modern, ultra-responsive web applications using React 19, TypeScript, Tailwind CSS, and WebSockets with strict performance SLAs.',
    'San Francisco, CA / Remote',
    'Full-time',
    175000,
    230000,
    'USD',
    '["React", "TypeScript", "Tailwind CSS", "WebSockets", "Architecture"]'::jsonb,
    6,
    '["Architect component design systems and state management for real-time interview interfaces.", "Build live streaming token visualizers, WebRTC audio monitors, and Monaco code editors.", "Guarantee 60fps animations and sub-100ms UI interaction latencies across all browsers.", "Mentor frontend engineers and champion clean design systems."]'::jsonb,
    '["6+ years building complex, high-performance web applications with React and TypeScript.", "Deep understanding of browser DOM performance, WebSockets, and WebRTC streaming.", "Mastery of modern CSS (Tailwind, animations, responsive layouts, dark modes).", "Track record of creating reusable, accessible design systems."]'::jsonb,
    '["Experience with Monaco Editor integration or browser-based IDEs.", "Contributions to open-source UI libraries."]'::jsonb,
    '["Top-tier compensation package + high-growth equity", "Health, dental, vision coverage & 401(k) matching", "Flexible working hours and remote-first setup"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.3, "semantic_match": 0.25, "education": 0.1}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '12 days',
    NOW() - INTERVAL '12 days',
    'strict',
    true
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published;

-- Job 3: Principal AI & ML Systems Engineer
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published
)
VALUES (
    'e5f6a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Principal AI & ML Systems Engineer',
    'Build high-scale inference engines, real-time audio WebRTC pipelines, and vector retrieval infrastructure for intelligent voice agents.',
    'New York, NY / Remote',
    'Full-time',
    190000,
    250000,
    'USD',
    '["Python", "PyTorch", "WebRTC", "pgvector", "LLM", "Whisper"]'::jsonb,
    7,
    '["Design real-time voice-to-voice interview synthesis pipelines with sub-300ms latency.", "Train, fine-tune, and optimize Whisper STT and Kokoro TTS models for technical evaluation.", "Build vector semantic search infrastructure using PostgreSQL pgvector and HNSW indexing.", "Implement deterministic prompt injection guardrails and anti-cheating anomaly detection."]'::jsonb,
    '["7+ years engineering ML/AI systems in production with Python and PyTorch.", "Expertise in LLM inference, structured output generation, and prompt safety rails.", "Experience optimizing audio processing models (STT/TTS) for real-time WebRTC streams.", "Strong understanding of vector databases, embeddings, and cosine similarity ranking."]'::jsonb,
    '["Publications or open-source projects in speech processing or LLM evaluation.", "Experience with ONNX runtime, TensorRT, or CUDA kernel optimization."]'::jsonb,
    '["Top-of-market base salary + equity package", "High-end workstation hardware (M4 Max / RTX 4090)", "Comprehensive premium medical coverage"]'::jsonb,
    '{"skills_match": 0.45, "experience_years": 0.2, "semantic_match": 0.25, "education": 0.1}'::jsonb,
    70.0,
    'active',
    NOW() - INTERVAL '10 days',
    NOW() - INTERVAL '10 days',
    'none',
    false
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published;
