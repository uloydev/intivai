-- 04_candidates.sql
-- Seed Candidate Talent Profiles with Parsed CV Intelligence

-- Candidate 1: Alex Rivera (Senior Go Distributed Systems Engineer)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
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
    NOW() - INTERVAL '8 days',
    NOW() - INTERVAL '8 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-alex-rivera-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 2: Elena Rostova (Staff Frontend Architect)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
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
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '7 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-elena-rostova-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 3: David Chen (Principal AI/ML Systems Engineer)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
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
    NOW() - INTERVAL '6 days',
    NOW() - INTERVAL '6 days',
    NULL,
    'magic-link-token-david-chen-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 4: Marcus Vance (Junior Developer - Below threshold)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
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
    'pending_review',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '5 days',
    NULL,
    'magic-link-token-marcus-vance-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 5: Liam O'Connor (Senior Cloud Platform & SRE -> Job 4)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444401',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Liam O''Connor',
    'liam.oconnor@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/liam_oconnor_cv.pdf',
    'Liam O''Connor. Lead Site Reliability Engineer with 6+ years managing high availability multi-region Kubernetes clusters on AWS and bare metal.',
    '{
        "name": "Liam O''Connor",
        "email": "liam.oconnor@example.com",
        "phone": "+44 20 7946 0192",
        "years_experience": 6,
        "skills": ["Kubernetes", "Terraform", "AWS", "Prometheus", "Docker", "Golang", "CI/CD", "Helm", "ArgoCD", "Grafana"],
        "education": [{"degree": "B.Sc. Computer Networks", "institution": "Imperial College London", "year": 2019}],
        "experience": [{"role": "Senior SRE Lead", "company": "FinTech Cloud UK", "duration": "2021 - Present"}],
        "certifications": ["Certified Kubernetes Administrator (CKA)", "AWS Solutions Architect Professional"]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '5 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-liam-oconnor-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 6: Maya Patel (Staff Mobile Engineer -> Job 5)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444402',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Maya Patel',
    'maya.patel@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/maya_patel_cv.pdf',
    'Maya Patel. Staff Mobile Architect with 7 years specializing in React Native, Swift iOS native bridges, and high performance offline audio streaming.',
    '{
        "name": "Maya Patel",
        "email": "maya.patel@example.com",
        "phone": "+1 (555) 678-9012",
        "years_experience": 7,
        "skills": ["React Native", "TypeScript", "Swift", "iOS", "Android", "Performance", "Offline Sync", "Redux", "WebRTC"],
        "education": [{"degree": "B.S. Electrical Engineering & CS", "institution": "UC Berkeley", "year": 2018}],
        "experience": [{"role": "Staff Mobile Architect", "company": "Nomad Health App", "duration": "2020 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '5 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-maya-patel-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 7: Henrik Lindqvist (Lead Full-Stack Product Engineer -> Job 6)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444403',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Henrik Lindqvist',
    'henrik.lindqvist@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/henrik_lindqvist_cv.pdf',
    'Henrik Lindqvist. Lead Fullstack Engineer with 8 years building Next.js web applications, high performance GraphQL endpoints, and Go services.',
    '{
        "name": "Henrik Lindqvist",
        "email": "henrik.lindqvist@example.com",
        "phone": "+49 30 1234567",
        "years_experience": 8,
        "skills": ["Next.js", "Node.js", "TypeScript", "PostgreSQL", "GraphQL", "Tailwind CSS", "Go", "Prisma"],
        "education": [{"degree": "M.Sc. Computer Science", "institution": "KTH Royal Institute of Technology", "year": 2017}],
        "experience": [{"role": "Principal Product Engineer", "company": "Klarna Ecosystem", "duration": "2019 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '4 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-henrik-lindqvist-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 8: Priya Sharma (Senior Data & Streaming Pipeline Engineer -> Job 7)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444404',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Priya Sharma',
    'priya.sharma@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/priya_sharma_cv.pdf',
    'Priya Sharma. Senior Data Engineer with 6 years experience in Apache Kafka event streams, Spark transformations, and Snowflake data warehouses.',
    '{
        "name": "Priya Sharma",
        "email": "priya.sharma@example.com",
        "phone": "+65 6789 0123",
        "years_experience": 6,
        "skills": ["Apache Kafka", "Spark", "Python", "Snowflake", "dbt", "SQL", "Airflow", "ClickHouse"],
        "education": [{"degree": "B.Eng. Computer Science", "institution": "National University of Singapore", "year": 2019}],
        "experience": [{"role": "Senior Data Platform Engineer", "company": "Grab Data Systems", "duration": "2020 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '4 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-priya-sharma-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 9: Zachary Taylor (Lead Application Security & DevSecOps -> Job 8)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444405',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Zachary Taylor',
    'zachary.taylor@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/zachary_taylor_cv.pdf',
    'Zachary Taylor. Lead Security Engineer with 7 years in AppSec, container hardening, zero trust OAuth2 architectures, and penetration testing.',
    '{
        "name": "Zachary Taylor",
        "email": "zachary.taylor@example.com",
        "phone": "+1 (555) 789-0123",
        "years_experience": 7,
        "skills": ["AppSec", "OWASP", "Penetration Testing", "IAM", "SOC 2", "Python", "Go", "Docker Security", "OAuth2"],
        "education": [{"degree": "B.S. Cybersecurity & Systems", "institution": "Georgia Institute of Technology", "year": 2018}],
        "experience": [{"role": "Head of Product Security", "company": "DefenseScale Cloud", "duration": "2021 - Present"}],
        "certifications": ["Offensive Security Certified Professional (OSCP)", "CISSP"]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-zachary-taylor-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 10: Chloe Dubois (Senior SDET & Test Automation Architect -> Job 9)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444406',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Chloe Dubois',
    'chloe.dubois@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/chloe_dubois_cv.pdf',
    'Chloe Dubois. Senior Test Automation Architect with 6 years leading Playwright E2E testing, k6 performance load generation, and hermetic CI fixtures.',
    '{
        "name": "Chloe Dubois",
        "email": "chloe.dubois@example.com",
        "phone": "+1 (555) 890-1234",
        "years_experience": 6,
        "skills": ["Playwright", "TypeScript", "CI/CD", "Docker", "Performance Testing", "Vitest", "k6", "Golang Testing"],
        "education": [{"degree": "M.S. Software Quality Engineering", "institution": "Sorbonne Université", "year": 2019}],
        "experience": [{"role": "Principal Quality Engineer", "company": "Spotify Tooling", "duration": "2021 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-chloe-dubois-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 11: Samuel Okafor (Engineering Manager / Technical Lead -> Job 10)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444407',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Samuel Okafor',
    'samuel.okafor@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/samuel_okafor_cv.pdf',
    'Samuel Okafor. Engineering Manager with 10 years experience leading high-trust engineering squads, scaling distributed platforms, and coaching staff engineers.',
    '{
        "name": "Samuel Okafor",
        "email": "samuel.okafor@example.com",
        "phone": "+1 (555) 901-2345",
        "years_experience": 10,
        "skills": ["Engineering Management", "System Design", "Mentorship", "Agile", "Hiring", "Architecture", "Distributed Systems"],
        "education": [{"degree": "M.S. Management & Computer Science", "institution": "MIT", "year": 2015}],
        "experience": [{"role": "Engineering Manager", "company": "Stripe Infrastructure", "duration": "2020 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-samuel-okafor-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 12: Jordan Brooks (Junior Fullstack Engineer -> Job 11)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444408',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Jordan Brooks',
    'jordan.brooks@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/jordan_brooks_cv.pdf',
    'Jordan Brooks. Fullstack developer with 2 years building React and Node.js web applications, responsive components, and REST APIs.',
    '{
        "name": "Jordan Brooks",
        "email": "jordan.brooks@example.com",
        "phone": "+1 (555) 012-3456",
        "years_experience": 2,
        "skills": ["TypeScript", "React", "Node.js", "PostgreSQL", "Git", "Tailwind CSS"],
        "education": [{"degree": "B.S. Software Engineering", "institution": "University of Texas at Austin", "year": 2023}],
        "experience": [{"role": "Junior Fullstack Developer", "company": "Venture Studio", "duration": "2023 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-jordan-brooks-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

-- Candidate 13: Sophia Zhang (AI Evaluation & Prompt Engineering Intern -> Job 12)
INSERT INTO candidates (
    id, org_id, name, email, cv_path, cv_raw_text, cv_structured,
    cv_ocr_method, status, created_at, updated_at, batch_id, review_token
)
VALUES (
    'aaaaaaaa-1111-2222-3333-444444444409',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Sophia Zhang',
    'sophia.zhang@example.com',
    'cvs/968f66ef-91c6-4db3-8764-ceeffb753b1f/sophia_zhang_cv.pdf',
    'Sophia Zhang. AI & NLP Master candidate at UC Berkeley researching automated prompt evaluation benchmarks, jailbreak safety rails, and LLM structured outputs.',
    '{
        "name": "Sophia Zhang",
        "email": "sophia.zhang@example.com",
        "phone": "+1 (555) 123-4567",
        "years_experience": 1,
        "skills": ["Python", "Prompt Engineering", "LLM Evaluation", "PyTorch", "NLP", "LangChain", "Hugging Face"],
        "education": [{"degree": "M.S. Artificial Intelligence", "institution": "UC Berkeley", "year": 2025}],
        "experience": [{"role": "AI Research Assistant", "company": "Berkeley AI Research (BAIR)", "duration": "2024 - Present"}]
    }'::jsonb,
    'pdfcpu',
    'extracted',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day',
    '33333333-3333-3333-3333-333333333333',
    'magic-link-token-sophia-zhang-2026'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    cv_structured = EXCLUDED.cv_structured,
    status = EXCLUDED.status,
    batch_id = EXCLUDED.batch_id,
    review_token = EXCLUDED.review_token,
    updated_at = NOW();

