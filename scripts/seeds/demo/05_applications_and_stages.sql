-- 05_applications_and_stages.sql
-- Seed Candidate Applications across Standardized Recruitment Funnel Stages (ADR-0001)

-- Application 1: Alex Rivera -> Senior Distributed Systems (Status: passed, Stage: interview_completed)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'd5e6f1a2-b3c4-4d5e-8f6a-1a2b3c4d5e6f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'f6a1b2c3-d4e5-4f6a-8b7c-8d9e0f1a2b3c',
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    92.5,
    '{"skills_match": 0.95, "experience_years": 0.90, "semantic_match": 0.92, "education": 0.85, "certifications": 1.0}'::jsonb,
    true,
    'passed',
    'interview_completed',
    'Exceptional technical depth in Go concurrency, PostgreSQL RLS, and distributed queues. Strong hire recommendation from AI assessment.',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 2: Elena Rostova -> Staff Frontend Architect (Status: passed, Stage: offer_extended)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'e6f1a2b3-c4d5-4e6f-8a1b-2b3c4d5e6f1a',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'a2b3c4d5-e6f1-4a2b-8c3d-9e0f1a2b3c4d',
    'd4e5f6a1-b2c3-4d4e-8f5a-6b7c8d9e0f1a',
    94.0,
    '{"skills_match": 0.96, "experience_years": 0.92, "semantic_match": 0.95, "education": 0.90, "certifications": 1.0}'::jsonb,
    true,
    'passed',
    'offer_extended',
    'Superb frontend architect. Clear domain leadership in design systems and WebSockets. Offer package extended to candidate.',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 3: David Chen -> Principal AI/ML Systems Engineer (Status: passed, Stage: interview_invited)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'f1a2b3c4-d5e6-4f1a-8b2c-3c4d5e6f1a2b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'b3c4d5e6-f1a2-4b3c-8d4e-0f1a2b3c4d5e',
    'e5f6a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b',
    89.0,
    '{"skills_match": 0.90, "experience_years": 0.88, "semantic_match": 0.88, "education": 0.95, "certifications": 1.0}'::jsonb,
    true,
    'passed',
    'interview_invited',
    'High-caliber ML engineer with deep audio/vector search background. Live interview invitation generated.',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 4: Marcus Vance -> Senior Distributed Systems (Status: rejected, Stage: screening_failed)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'a3b4c5d6-e7f2-4a3b-8c4d-4c5d6e7f2a3b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'c4d5e6f1-a2b3-4c4d-8e5f-1a2b3c4d5e6f',
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    38.0,
    '{"skills_match": 0.30, "experience_years": 0.25, "semantic_match": 0.40, "education": 0.70, "certifications": 0.0}'::jsonb,
    false,
    'rejected',
    'screening_failed',
    'Candidate does not meet minimum experience threshold (1 yr vs 5 yrs required) and lacks core Go distributed systems background.',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 5: Liam O'Connor -> Senior Cloud Platform & SRE (Job 4)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444401',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444401',
    '11111111-2222-3333-4444-555555555501',
    91.0,
    '{"skills_match": 0.95, "experience_years": 0.90, "semantic_match": 0.88, "certifications": 0.95}'::jsonb,
    true,
    'passed',
    'interview_invited',
    'Outstanding Kubernetes and AWS platform experience with CKA/CKS certifications. Interview invitation dispatched.',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '4 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 6: Maya Patel -> Staff Mobile Engineer (Job 5)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444402',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444402',
    '11111111-2222-3333-4444-555555555502',
    93.5,
    '{"skills_match": 0.96, "experience_years": 0.92, "semantic_match": 0.94, "education": 0.90}'::jsonb,
    true,
    'passed',
    'screening_passed',
    'High-match mobile architect candidate with proven track record in native iOS bridges and WebRTC audio.',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '4 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 7: Henrik Lindqvist -> Lead Full-Stack Product Engineer (Job 6)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444403',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444403',
    '11111111-2222-3333-4444-555555555503',
    90.5,
    '{"skills_match": 0.92, "experience_years": 0.90, "semantic_match": 0.90, "education": 0.90}'::jsonb,
    true,
    'passed',
    'interview_invited',
    'Strong fullstack leadership profile with robust Next.js and Go experience. Interview invited.',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 8: Priya Sharma -> Senior Data & Streaming Pipeline Engineer (Job 7)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444404',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444404',
    '11111111-2222-3333-4444-555555555504',
    88.5,
    '{"skills_match": 0.90, "experience_years": 0.88, "semantic_match": 0.88, "education": 0.85}'::jsonb,
    true,
    'passed',
    'screening_passed',
    'Solid data engineering profile with Kafka, PySpark, and Snowflake data modeling experience.',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 9: Zachary Taylor -> Lead Application Security & DevSecOps (Job 8)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444405',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444405',
    '11111111-2222-3333-4444-555555555505',
    95.0,
    '{"skills_match": 0.98, "experience_years": 0.92, "semantic_match": 0.92, "certifications": 1.0}'::jsonb,
    true,
    'passed',
    'offer_extended',
    'Top-tier security architect. Perfect match for sandbox isolation hardening and SOC 2 governance. Offer extended.',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 10: Chloe Dubois -> Senior SDET & Test Automation (Job 9)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444406',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444406',
    '11111111-2222-3333-4444-555555555506',
    92.0,
    '{"skills_match": 0.95, "experience_years": 0.90, "semantic_match": 0.90, "education": 0.90}'::jsonb,
    true,
    'passed',
    'interview_completed',
    'Completed Playwright test architecture assessment with a 94/100 score. Excellent test fixture isolation capabilities.',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 11: Samuel Okafor -> Engineering Manager / Technical Lead (Job 10)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444407',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444407',
    '11111111-2222-3333-4444-555555555507',
    96.0,
    '{"experience_years": 0.98, "skills_match": 0.95, "semantic_match": 0.94, "education": 0.95}'::jsonb,
    true,
    'passed',
    'hired',
    'Stellar engineering manager with 10 years experience at Stripe. Successfully accepted offer and completed onboarding.',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 12: Jordan Brooks -> Junior Fullstack Engineer (Job 11)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444408',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444408',
    '11111111-2222-3333-4444-555555555508',
    85.0,
    '{"skills_match": 0.88, "semantic_match": 0.85, "education": 0.85, "experience_years": 0.80}'::jsonb,
    true,
    'passed',
    'screening_passed',
    'Strong junior candidate with solid TypeScript and React component skills. Passed initial threshold for contract role.',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

-- Application 13: Sophia Zhang -> AI Evaluation & Prompt Engineering Intern (Job 12)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, stage, recruiter_notes, created_at, updated_at
)
VALUES (
    'bbbbbbbb-1111-2222-3333-444444444409',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'aaaaaaaa-1111-2222-3333-444444444409',
    '11111111-2222-3333-4444-555555555509',
    88.0,
    '{"skills_match": 0.90, "education": 0.92, "semantic_match": 0.85, "experience_years": 0.75}'::jsonb,
    true,
    'passed',
    'interview_invited',
    'Exceptional AI Master student from UC Berkeley with direct research in prompt evaluation. Interview invited.',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    score_breakdown = EXCLUDED.score_breakdown,
    passed_screening = EXCLUDED.passed_screening,
    status = EXCLUDED.status,
    stage = EXCLUDED.stage,
    recruiter_notes = EXCLUDED.recruiter_notes,
    updated_at = NOW();

