-- 05_applications_and_stages.sql
-- Seed Candidate Applications across Standardized Recruitment Funnel Stages

-- Application 1: Alex Rivera -> Senior Distributed Systems (Stage: interview_completed / Strong Hire)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, created_at
)
VALUES (
    'd5e6f1a2-b3c4-4d5e-8f6a-1a2b3c4d5e6f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'f6a1b2c3-d4e5-4f6a-8b7c-8d9e0f1a2b3c',
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    92.5,
    '{"skills_score": 95.0, "experience_score": 90.0, "semantic_score": 92.0, "education_score": 85.0}'::jsonb,
    true,
    'interview_completed',
    NOW() - INTERVAL '4 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    status = EXCLUDED.status;

-- Application 2: Elena Rostova -> Staff Frontend Architect (Stage: offer_extended)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, created_at
)
VALUES (
    'e6f1a2b3-c4d5-4e6f-8a1b-2b3c4d5e6f1a',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'a2b3c4d5-e6f1-4a2b-8c3d-9e0f1a2b3c4d',
    'd4e5f6a1-b2c3-4d4e-8f5a-6b7c8d9e0f1a',
    94.0,
    '{"skills_score": 96.0, "experience_score": 92.0, "semantic_score": 95.0, "education_score": 90.0}'::jsonb,
    true,
    'offer_extended',
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    status = EXCLUDED.status;

-- Application 3: David Chen -> Principal AI/ML Systems Engineer (Stage: interview_invited)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, created_at
)
VALUES (
    'f1a2b3c4-d5e6-4f1a-8b2c-3c4d5e6f1a2b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'b3c4d5e6-f1a2-4b3c-8d4e-0f1a2b3c4d5e',
    'e5f6a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b',
    89.0,
    '{"skills_score": 90.0, "experience_score": 88.0, "semantic_score": 88.0, "education_score": 95.0}'::jsonb,
    true,
    'interview_invited',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    status = EXCLUDED.status;

-- Application 4: Marcus Vance -> Senior Distributed Systems (Stage: screening_failed)
INSERT INTO applications (
    id, org_id, candidate_id, job_id, cv_score, score_breakdown,
    passed_screening, status, created_at
)
VALUES (
    'a3b4c5d6-e7f2-4a3b-8c4d-4c5d6e7f2a3b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'c4d5e6f1-a2b3-4c4d-8e5f-1a2b3c4d5e6f',
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    38.0,
    '{"skills_score": 30.0, "experience_score": 25.0, "semantic_score": 40.0, "education_score": 70.0}'::jsonb,
    false,
    'screening_failed',
    NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO UPDATE SET
    cv_score = EXCLUDED.cv_score,
    status = EXCLUDED.status;
