#!/bin/bash

# 1. Tests & Coverage
git add scripts/check-coverage.sh
git add backend/internal/screening/application/update_decision_test.go
git add backend/internal/iam/
git add backend/internal/notification/
git commit -m "test: add coverage checks, update decision tests, and OTP lockout"

# 2. CI & Sandbox & Docker
git add .github/workflows/ci.yml docker-compose.prod.yml scripts/smoke.sh backend/Dockerfile Makefile backend/Makefile backend/internal/sandbox/
git commit -m "ci: deploy sandbox to production and harden pipeline"

# 3. Mailer
git add backend/pkg/mailer/
git commit -m "fix: replace mailpit with SMTP in prod and fix email formatting"

# 4. is_published Gate
git add backend/internal/job/ backend/pkg/db/migrations/017* frontend/src/pages/Jobs.tsx frontend/src/pages/Careers.tsx
git commit -m "feat: enforce strict is_published gating and FE toggle"

# 5. Remove Fabricated Values
git add frontend/src/pages/Candidates.tsx frontend/src/components/candidates/Candidate360Drawer.tsx frontend/src/pages/InterviewResult.tsx backend/internal/evaluation/application/pdf.go
git commit -m "refactor: remove fabricated candidate data and mock UI states"

# 6. Cost Rails & Token Ledger
git add backend/internal/llm/ backend/internal/evaluation/application/evaluation_worker.go backend/internal/cv/application/extract_worker.go backend/internal/cv/application/extract_worker_failure_test.go backend/internal/screening/application/score_worker.go backend/pkg/db/errors.go
git commit -m "feat: implement LLM max retries and token ledger bounding"

# 7. HR Fairness & Scoring Calibration
git add backend/internal/screening/domain/scoring.go backend/internal/evaluation/domain/report.go backend/internal/evaluation/infrastructure/llm/evaluator.go frontend/src/pages/Invite.tsx backend/internal/screening/domain/stage.go backend/internal/screening/application/screening_service.go
git commit -m "feat: calibrate HR fairness dimensions, prompt constraints, and 30-QA window"

# 8. Disaster Recovery
git add scripts/backup.cron scripts/restore.sh
git commit -m "chore: add database disaster recovery scripts"

# 9. Tech Lead Nits & Bug Fixes
git add frontend/tsconfig.app.json frontend/src/App.tsx frontend/src/pages/CandidateReview.tsx frontend/src/components/sandbox/CodeEditor.tsx frontend/src/components/sandbox/CodingSandbox.tsx backend/internal/memory/infrastructure/native/native_memory.go backend/internal/context/application/context_service.go
git commit -m "fix: address tech lead nits (strict TS, dynamic imports, modal focus, and lock contention)"

# 10. Agent Rules & Docs
git add .agents/ AGENTS.md AI_Interviewer_*.md P4_Plan.md REVIEW_2026-08-19.md
git commit -m "docs: update agent rules and tech lead review documentation"

# 11. Catch-all for remaining migrations and files
git add .
git commit -m "chore: sync remaining migrations and internal refactors"

