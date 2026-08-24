-- 07_passports.sql
-- Seed Global Candidate Passports and Candidate Portal OTP Authentication Records

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

-- Seed Demo Candidate Portal OTPs (Code: 123456 / SHA256: 8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92)
INSERT INTO candidate_otps (
    id, email, code_hash, token, attempts, expires_at, created_at
)
VALUES
    (
        '10101010-1010-1010-1010-101010101010',
        'alex.rivera@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-alex-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '20202020-2020-2020-2020-202020202020',
        'elena.rostova@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-elena-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '30303030-3030-3030-3030-303030303030',
        'david.chen@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-david-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '40404040-4040-4040-4040-404040404040',
        'liam.oconnor@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-liam-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '50505050-5050-5050-5050-505050505050',
        'maya.patel@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-maya-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '60606060-6060-6060-6060-606060606060',
        'henrik.lindqvist@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-henrik-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '70707070-7070-7070-7070-707070707070',
        'priya.sharma@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-priya-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '80808080-8080-8080-8080-808080808080',
        'zachary.taylor@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-zachary-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        '90909090-9090-9090-9090-909090909090',
        'chloe.dubois@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-chloe-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        'a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0',
        'samuel.okafor@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-samuel-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        'b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0',
        'jordan.brooks@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-jordan-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    ),
    (
        'c0c0c0c0-c0c0-c0c0-c0c0-c0c0c0c0c0c0',
        'sophia.zhang@example.com',
        '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92',
        'demo-magic-token-sophia-2026',
        0,
        NOW() + INTERVAL '30 days',
        NOW() - INTERVAL '1 hour'
    )
ON CONFLICT (token) DO UPDATE SET
    expires_at = EXCLUDED.expires_at,
    code_hash = EXCLUDED.code_hash;

