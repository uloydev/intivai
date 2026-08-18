-- 04_candidates.sql
-- Seed Candidate Talent Profiles with Parsed CV Intelligence

-- Candidate 1: Alex Rivera (Senior Go Distributed Systems Engineer)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at
)
VALUES (
    'f6a1b2c3-d4e5-4f6a-8b7c-8d9e0f1a2b3c',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Alex Rivera',
    'alex.rivera@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/alex_rivera_cv.pdf',
    'Alex Rivera. Staff Software Engineer. 8+ years building high throughput distributed systems in Go and PostgreSQL. Led architecture at CloudScale Inc.',
    '{
        "name": "Alex Rivera",
        "email": "alex.rivera@example.com",
        "phone": "+1 (555) 234-5678",
        "years_experience": 8,
        "skills": ["Go", "PostgreSQL", "Distributed Systems", "Docker", "Kubernetes", "Redis", "gRPC", "Kafka"],
        "education": [{"degree": "B.S. Computer Science", "institution": "University of Washington", "year": 2018}],
        "experience": [{"role": "Staff Software Engineer", "company": "CloudScale Inc", "duration": "2021 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '8 days'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured;

-- Candidate 2: Elena Rostova (Staff Frontend Architect)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at
)
VALUES (
    'a2b3c4d5-e6f1-4a2b-8c3d-9e0f1a2b3c4d',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Elena Rostova',
    'elena.rostova@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/elena_rostova_cv.pdf',
    'Elena Rostova. Lead Frontend Architect with 7 years of deep React, TypeScript, and design systems experience at UI Labs.',
    '{
        "name": "Elena Rostova",
        "email": "elena.rostova@example.com",
        "phone": "+1 (555) 345-6789",
        "years_experience": 7,
        "skills": ["React", "TypeScript", "Tailwind CSS", "WebSockets", "Architecture", "Next.js", "Design Systems"],
        "education": [{"degree": "M.S. Software Engineering", "institution": "Carnegie Mellon University", "year": 2019}],
        "experience": [{"role": "Lead Frontend Architect", "company": "UI Labs", "duration": "2020 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '7 days'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured;

-- Candidate 3: David Chen (Principal AI/ML Systems Engineer)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at
)
VALUES (
    'b3c4d5e6-f1a2-4b3c-8d4e-0f1a2b3c4d5e',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'David Chen',
    'david.chen@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/david_chen_cv.pdf',
    'David Chen. Principal AI Engineer. 9 years experience in real-time inference, Whisper audio processing, and vector search with PyTorch.',
    '{
        "name": "David Chen",
        "email": "david.chen@example.com",
        "phone": "+1 (555) 456-7890",
        "years_experience": 9,
        "skills": ["Python", "PyTorch", "WebRTC", "pgvector", "LLM", "Whisper", "CUDA", "C++"],
        "education": [{"degree": "Ph.D. Computer Science", "institution": "Stanford University", "year": 2017}],
        "experience": [{"role": "Principal AI Engineer", "company": "Synthetix AI", "duration": "2019 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '6 days'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured;

-- Candidate 4: Marcus Vance (Junior Developer - Below threshold)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at
)
VALUES (
    'c4d5e6f1-a2b3-4c4d-8e5f-1a2b3c4d5e6f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Marcus Vance',
    'marcus.vance@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/marcus_vance_cv.pdf',
    'Marcus Vance. Junior developer with 1 year Python and web basics.',
    '{
        "name": "Marcus Vance",
        "email": "marcus.vance@example.com",
        "phone": "+1 (555) 567-8901",
        "years_experience": 1,
        "skills": ["Python", "HTML", "CSS"],
        "education": [{"degree": "B.A. Information Systems", "institution": "State College", "year": 2024}],
        "experience": [{"role": "Junior Web Intern", "company": "Local Agency", "duration": "2024 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '5 days'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured;
