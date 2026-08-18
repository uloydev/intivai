-- 06_interviews_and_evaluations.sql
-- Seed Interview Sessions, Multi-Turn Transcripts, Scorecards, and Proctoring Telemetry

-- Interview 1: Alex Rivera (Completed Senior Distributed Systems Chat Interview)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx,
    started_at, completed_at, expires_at, created_at, updated_at,
    transcript, evaluation
)
VALUES (
    'b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c',
    'd5e6f1a2-b3c4-4d5e-8f6a-1a2b3c4d5e6f',
    'chat',
    'completed',
    true,
    3,
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days' + INTERVAL '22 minutes',
    NOW() - INTERVAL '3 days' + INTERVAL '30 minutes',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days',
    '[
        {
            "role": "interviewer",
            "content": "Can you describe a challenging distributed concurrency or race condition issue you diagnosed in Go, and how you resolved it?",
            "timestamp": "2026-08-15T10:00:00Z"
        },
        {
            "role": "candidate",
            "content": "In our event processing pipeline, we experienced silent goroutine leaks and data races during high-throughput shard rebalancing. We utilized Go sync.RWMutex with atomic state pointers and context cancellation propagation. We ran race detector in CI and load tested with 50,000 concurrent socket events.",
            "timestamp": "2026-08-15T10:04:15Z"
        },
        {
            "role": "interviewer",
            "content": "How do you enforce PostgreSQL tenant isolation and transaction safety when handling asynchronous background tasks?",
            "timestamp": "2026-08-15T10:05:00Z"
        },
        {
            "role": "candidate",
            "content": "We enforce PostgreSQL Row-Level Security (RLS) with FORCE ROW LEVEL SECURITY on all tenant tables. In background workers, every task execution opens an isolated transaction executing SET LOCAL app.org_id before any queries run, guaranteeing strict multi-tenant isolation.",
            "timestamp": "2026-08-15T10:11:30Z"
        }
    ]'::jsonb,
    '{
        "overall": 92.0,
        "recommendation": "Strong Hire",
        "scores": {
            "technical_depth": 95.0,
            "architecture_design": 92.0,
            "problem_solving": 90.0,
            "communication": 91.0
        },
        "summary": "Outstanding candidate with deep hands-on expertise in Go concurrency primitives, PostgreSQL RLS tenant isolation, and distributed event architectures.",
        "strengths": [
            "Exceptional understanding of Go memory model, goroutines, and channels",
            "Production experience enforcing database-level Row-Level Security",
            "Structured and quantified explanations under technical probing"
        ],
        "growth_areas": [
            "Could expand on cross-region consensus mechanisms (e.g. Raft vs Paxos trade-offs)"
        ]
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    evaluation = EXCLUDED.evaluation;

-- Interview 2: David Chen (Invited to Interview with Active Token)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx,
    created_at, updated_at
)
VALUES (
    'c5d6e7f8-a4b5-4c5d-8e6f-6e7f8a4b5c5d',
    'f1a2b3c4-d5e6-4f1a-8b2c-3c4d5e6f1a2b',
    'chat',
    'pending',
    false,
    0,
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO NOTHING;

-- Interview Token for David Chen:
INSERT INTO interview_tokens (
    id, org_id, interview_id, token, expires_at, created_at
)
VALUES (
    'd6e7f8a4-b5c6-4d6e-8f7a-7f8a4b5c6d6e',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'c5d6e7f8-a4b5-4c5d-8e6f-6e7f8a4b5c5d',
    'demo-invitation-token-david-chen-2026',
    NOW() + INTERVAL '7 days',
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO NOTHING;
