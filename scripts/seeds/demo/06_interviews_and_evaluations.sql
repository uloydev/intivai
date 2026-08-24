-- 06_interviews_and_evaluations.sql
-- Seed Interview Sessions, Structured Transcripts, Canonical Report Scorecards, Proctoring Telemetry & Coding Sessions

-- Interview 1: Alex Rivera (Completed Senior Distributed Systems Chat & Coding Interview)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx, context_version,
    started_at, completed_at, expires_at, created_at, updated_at,
    transcript, evaluation, proctoring_events, proctoring_summary, coding_sessions
)
VALUES (
    'b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c',
    'd5e6f1a2-b3c4-4d5e-8f6a-1a2b3c4d5e6f',
    'chat',
    'completed',
    true,
    2,
    1,
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days' + INTERVAL '22 minutes',
    NOW() - INTERVAL '3 days' + INTERVAL '30 minutes',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days',
    '{
        "questions": [
            {
                "idx": 1,
                "content": "Can you describe a challenging distributed concurrency or race condition issue you diagnosed in Go, and how you resolved it?",
                "category": "technical",
                "skill": "Go"
            },
            {
                "idx": 2,
                "content": "How do you enforce PostgreSQL tenant isolation and transaction safety when handling asynchronous background tasks?",
                "category": "technical",
                "skill": "PostgreSQL"
            }
        ],
        "answers": [
            {
                "idx": 1,
                "content": "In our event processing pipeline, we experienced silent goroutine leaks and data races during high-throughput shard rebalancing. We utilized Go sync.RWMutex with atomic state pointers and context cancellation propagation. We ran race detector in CI and load tested with 50,000 concurrent socket events.",
                "answered_at": "2026-08-15T10:04:15Z"
            },
            {
                "idx": 2,
                "content": "We enforce PostgreSQL Row-Level Security (RLS) with FORCE ROW LEVEL SECURITY on all tenant tables. In background workers, every task execution opens an isolated transaction executing SET LOCAL app.org_id before any queries run, guaranteeing strict multi-tenant isolation.",
                "answered_at": "2026-08-15T10:11:30Z"
            }
        ]
    }'::jsonb,
    '{
        "overall_score": 92.0,
        "dimensions": {
            "technical": {"score": 95.0, "weight": 0.40},
            "communication": {"score": 91.0, "weight": 0.20},
            "problem_solving": {"score": 90.0, "weight": 0.25},
            "culture_fit": {"score": 88.0, "weight": 0.15}
        },
        "per_question": [
            {
                "question_idx": 1,
                "category": "technical",
                "score": 95.0,
                "rationale": "Demonstrated deep mastery of Go concurrency primitives, atomic pointers, and race detector tooling in CI.",
                "quotes": [
                    "utilized Go sync.RWMutex with atomic state pointers and context cancellation propagation",
                    "ran race detector in CI and load tested with 50,000 concurrent socket events"
                ],
                "strengths": [
                    "Deep knowledge of Go sync primitives",
                    "Production-scale race debugging"
                ],
                "weaknesses": []
            },
            {
                "question_idx": 2,
                "category": "technical",
                "score": 92.0,
                "rationale": "Clear and rigorous explanation of PostgreSQL Row-Level Security and transaction-scoped context setting in asynchronous workers.",
                "quotes": [
                    "enforce PostgreSQL Row-Level Security (RLS) with FORCE ROW LEVEL SECURITY",
                    "every task execution opens an isolated transaction executing SET LOCAL app.org_id"
                ],
                "strengths": [
                    "Strong multi-tenant RLS understanding",
                    "Transaction safety awareness"
                ],
                "weaknesses": [
                    "Could expand on cross-region replication trade-offs"
                ]
            }
        ],
        "strengths": [
            "Exceptional understanding of Go memory model, goroutines, and channels",
            "Production experience enforcing database-level Row-Level Security",
            "Structured and quantified explanations under technical probing"
        ],
        "weaknesses": [
            "Could expand on cross-region consensus mechanisms (e.g. Raft vs Paxos trade-offs)"
        ],
        "recommendation": "proceed"
    }'::jsonb,
    '[
        {
            "type": "focus_lost",
            "timestamp": "2026-08-15T10:06:12Z",
            "question_idx": 2
        },
        {
            "type": "focus_regained",
            "timestamp": "2026-08-15T10:06:16Z",
            "question_idx": 2
        },
        {
            "type": "tab_switch",
            "timestamp": "2026-08-15T10:06:16Z",
            "question_idx": 2
        },
        {
            "type": "clipboard_paste",
            "timestamp": "2026-08-15T10:10:05Z",
            "question_idx": 2,
            "details": {
                "pasted_text_length": 42
            }
        }
    ]'::jsonb,
    '{
        "integrity_score": 92,
        "risk_level": "low",
        "tab_switch_count": 1,
        "total_away_duration_sec": 4,
        "paste_event_count": 1,
        "suspicious_paste_count": 0,
        "audio_anomaly_count": 0,
        "flags": [
            "Candidate switched tabs or blurred browser 1 time(s)",
            "Detected 1 minor clipboard paste event(s)"
        ]
    }'::jsonb,
    '[
        {
            "question_idx": 2,
            "language": "go",
            "code": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n)\n\nfunc ExecuteTask(ctx context.Context, orgID string) error {\n\t// Set tenant context for safe RLS\n\tfmt.Printf(\"Executing isolated task for org: %s\\n\", orgID)\n\treturn nil\n}\n\nfunc main() {\n\t_ = ExecuteTask(context.Background(), \"demo-org\")\n}\n",
            "submitted_at": "2026-08-15T10:11:00Z",
            "final_result": {
                "stdout": "Executing isolated task for org: demo-org\n",
                "stderr": "",
                "exit_code": 0,
                "duration_ms": 142,
                "all_passed": true
            },
            "ai_code_review": {
                "time_complexity": "O(1)",
                "space_complexity": "O(1)",
                "quality_score": 95,
                "summary": "Clean, idiomatic Go implementation with correct context propagation.",
                "strengths": ["Proper context propagation", "Idiomatic error handling"],
                "improvements": []
            }
        }
    ]'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    transcript = EXCLUDED.transcript,
    evaluation = EXCLUDED.evaluation,
    proctoring_events = EXCLUDED.proctoring_events,
    proctoring_summary = EXCLUDED.proctoring_summary,
    coding_sessions = EXCLUDED.coding_sessions,
    updated_at = NOW();

-- Interview 2: David Chen (Invited to Live AI Interview with Active Token)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx, context_version,
    created_at, updated_at, transcript
)
VALUES (
    'c5d6e7f8-a4b5-4c5d-8e6f-6e7f8a4b5c5d',
    'f1a2b3c4-d5e6-4f1a-8b2c-3c4d5e6f1a2b',
    'chat',
    'pending',
    false,
    0,
    1,
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days',
    '{
        "questions": [
            {
                "idx": 1,
                "content": "How do you optimize Whisper speech-to-text inference latency for real-time WebRTC audio streams?",
                "category": "technical",
                "skill": "Whisper"
            },
            {
                "idx": 2,
                "content": "Explain how you structure HNSW vector index parameters in pgvector for sub-50ms cosine similarity recall at scale.",
                "category": "technical",
                "skill": "pgvector"
            },
            {
                "idx": 3,
                "content": "How do you defend LLM evaluation pipelines against prompt injection attacks embedded in candidate resumes or code submissions?",
                "category": "problem_solving",
                "skill": "LLM Security"
            }
        ],
        "answers": []
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    transcript = EXCLUDED.transcript,
    updated_at = NOW();

-- Interview Tokens:
-- Token 1 for Alex Rivera (used/completed interview)
INSERT INTO interview_tokens (
    id, org_id, interview_id, token, expires_at, used_at, created_at
)
VALUES (
    'e7f8a4b5-c6d7-4e7f-8a1b-8a1b2c3d4e5f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c',
    'demo-invitation-token-alex-rivera-2026',
    NOW() + INTERVAL '4 days',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '4 days'
)
ON CONFLICT (id) DO UPDATE SET
    used_at = EXCLUDED.used_at,
    expires_at = EXCLUDED.expires_at;

-- Token 2 for David Chen (active pending invitation)
INSERT INTO interview_tokens (
    id, org_id, interview_id, token, expires_at, used_at, created_at
)
VALUES (
    'd6e7f8a4-b5c6-4d6e-8f7a-7f8a4b5c6d6e',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'c5d6e7f8-a4b5-4c5d-8e6f-6e7f8a4b5c5d',
    'demo-invitation-token-david-chen-2026',
    NOW() + INTERVAL '7 days',
    NULL,
    NOW() - INTERVAL '2 days'
)
ON CONFLICT (id) DO UPDATE SET
    expires_at = EXCLUDED.expires_at;

-- Interview 3: Chloe Dubois (Completed SDET Automation Interview)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx, context_version,
    started_at, completed_at, expires_at, created_at, updated_at,
    transcript, evaluation, proctoring_events, proctoring_summary, coding_sessions
)
VALUES (
    'cccccccc-1111-2222-3333-444444444406',
    'bbbbbbbb-1111-2222-3333-444444444406',
    'chat',
    'completed',
    true,
    2,
    1,
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day' + INTERVAL '18 minutes',
    NOW() - INTERVAL '1 day' + INTERVAL '30 minutes',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day',
    '{
        "questions": [
            {
                "idx": 1,
                "content": "How do you design hermetic Playwright test suites that prevent flaky test runs and isolate shared database state across parallel test workers?",
                "category": "technical",
                "skill": "Playwright"
            },
            {
                "idx": 2,
                "content": "How do you integrate automated k6 performance tests into GitHub Actions CI to prevent latency regressions?",
                "category": "technical",
                "skill": "Performance Testing"
            }
        ],
        "answers": [
            {
                "idx": 1,
                "content": "We use dynamic test database schemas per worker or transaction rollback fixtures, along with deterministic seed harnesses and custom locator retry thresholds.",
                "answered_at": "2026-08-19T14:10:00Z"
            },
            {
                "idx": 2,
                "content": "We run threshold-gated k6 scripts checking p95 latency under 100 concurrent virtual users, automatically blocking PR merge if SLA exceeds 150ms.",
                "answered_at": "2026-08-19T14:18:00Z"
            }
        ]
    }'::jsonb,
    '{
        "overall_score": 94.0,
        "dimensions": {
            "technical": {"score": 96.0, "weight": 0.40},
            "communication": {"score": 92.0, "weight": 0.20},
            "problem_solving": {"score": 94.0, "weight": 0.25},
            "culture_fit": {"score": 92.0, "weight": 0.15}
        },
        "per_question": [
            {
                "question_idx": 1,
                "category": "technical",
                "score": 96.0,
                "rationale": "Clear mastery of test isolation patterns and browser automation best practices.",
                "quotes": ["dynamic test database schemas per worker", "transaction rollback fixtures"],
                "strengths": ["Deep Playwright knowledge", "Flakiness mitigation expertise"],
                "weaknesses": []
            }
        ],
        "strengths": [
            "Comprehensive understanding of browser automation and parallel worker state isolation",
            "Strong grasp of CI/CD gate automation"
        ],
        "weaknesses": [],
        "recommendation": "proceed"
    }'::jsonb,
    '[]'::jsonb,
    '{
        "integrity_score": 98,
        "risk_level": "low",
        "tab_switch_count": 0,
        "total_away_duration_sec": 0,
        "paste_event_count": 0,
        "suspicious_paste_count": 0,
        "audio_anomaly_count": 0,
        "flags": []
    }'::jsonb,
    '[]'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    transcript = EXCLUDED.transcript,
    evaluation = EXCLUDED.evaluation,
    updated_at = NOW();

-- Interview 4: Liam O'Connor (Pending SRE Interview)
INSERT INTO interviews (
    id, application_id, type, status, consent_given, last_question_idx, context_version,
    started_at, completed_at, expires_at, created_at, updated_at,
    transcript
)
VALUES (
    'cccccccc-1111-2222-3333-444444444401',
    'bbbbbbbb-1111-2222-3333-444444444401',
    'chat',
    'pending',
    false,
    0,
    1,
    NULL,
    NULL,
    NOW() + INTERVAL '5 days',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '4 days',
    '{
        "questions": [
            {
                "idx": 1,
                "content": "How do you configure Kubernetes PodDisruptionBudgets, topology spread constraints, and cluster autoscaling to ensure zero downtime during node upgrades?",
                "category": "technical",
                "skill": "Kubernetes"
            }
        ],
        "answers": []
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    transcript = EXCLUDED.transcript,
    updated_at = NOW();

-- Token 3 for Chloe Dubois (used)
INSERT INTO interview_tokens (
    id, org_id, interview_id, token, expires_at, used_at, created_at
)
VALUES (
    'dddddddd-1111-2222-3333-444444444406',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'cccccccc-1111-2222-3333-444444444406',
    'demo-invitation-token-chloe-dubois-2026',
    NOW() + INTERVAL '4 days',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO UPDATE SET
    used_at = EXCLUDED.used_at,
    expires_at = EXCLUDED.expires_at;

-- Token 4 for Liam O'Connor (active)
INSERT INTO interview_tokens (
    id, org_id, interview_id, token, expires_at, used_at, created_at
)
VALUES (
    'dddddddd-1111-2222-3333-444444444401',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'cccccccc-1111-2222-3333-444444444401',
    'demo-invitation-token-liam-oconnor-2026',
    NOW() + INTERVAL '5 days',
    NULL,
    NOW() - INTERVAL '4 days'
)
ON CONFLICT (id) DO UPDATE SET
    expires_at = EXCLUDED.expires_at;

