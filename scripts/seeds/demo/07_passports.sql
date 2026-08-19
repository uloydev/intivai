-- 07_passports.sql
-- Seed Global Candidate Passports for cross-tenant moat testing

INSERT INTO global_candidate_passports (
    id, email, verified_profile, global_score, created_at, updated_at
)
VALUES (
    'aa112233-bb44-cc55-dd66-ee7788990011',
    'alex.rivera@example.com',
    '{
        "name": "Alex Rivera",
        "email": "alex.rivera@example.com",
        "phone": "+1 (555) 234-5678",
        "years_experience": 8,
        "skills": ["Go", "PostgreSQL", "Distributed Systems", "Docker", "Kubernetes", "Redis", "gRPC", "Kafka"],
        "education": [{"degree": "B.S. Computer Science", "institution": "University of Washington", "year": 2018}],
        "experience": [{"role": "Staff Software Engineer", "company": "CloudScale Inc", "duration": "2021 - Present"}]
    }'::jsonb,
    88.5,
    NOW() - INTERVAL '8 days',
    NOW() - INTERVAL '8 days'
)
ON CONFLICT (email) DO UPDATE SET
    verified_profile = EXCLUDED.verified_profile,
    global_score = EXCLUDED.global_score,
    updated_at = NOW();

INSERT INTO global_candidate_passports (
    id, email, verified_profile, global_score, created_at, updated_at
)
VALUES (
    'bb223344-cc55-dd66-ee77-ff8899001122',
    'elena.rostova@example.com',
    '{
        "name": "Elena Rostova",
        "email": "elena.rostova@example.com",
        "phone": "+1 (555) 345-6789",
        "years_experience": 7,
        "skills": ["React", "TypeScript", "Tailwind CSS", "WebSockets", "Architecture", "Next.js", "Design Systems"],
        "education": [{"degree": "M.S. Software Engineering", "institution": "Carnegie Mellon University", "year": 2019}],
        "experience": [{"role": "Lead Frontend Architect", "company": "UI Labs", "duration": "2020 - Present"}]
    }'::jsonb,
    92.0,
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '7 days'
)
ON CONFLICT (email) DO UPDATE SET
    verified_profile = EXCLUDED.verified_profile,
    global_score = EXCLUDED.global_score,
    updated_at = NOW();
