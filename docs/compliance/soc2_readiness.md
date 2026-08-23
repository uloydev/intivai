# SOC 2 Type I — Controls Inventory

> Status: corrected 2026-08-24 · Owner: EM
> **Readiness inventory — NOT a certification claim.** Intivai is not SOC 2
> certified; certification is a post-beta goal. Statuses reflect code reality
> as of the correction date; gaps are tracked in `docs/FINDINGS.md`.

## Trust Service Criteria

| TSC | Control | Implementation | Status |
|-----|---------|---------------|--------|
| CC6.1 | Logical access controls | JWT + RBAC + Postgres RLS (ENABLE + FORCE, least-privilege app role) | Implemented |
| CC6.2 | Authentication | bcrypt (users), SHA256 OTP (candidates), magic links | Implemented |
| CC6.3 | Access removal | Candidate self-service erasure (`DELETE /candidate/portal/me`) | Partial (candidate path only) |
| CC6.3a | User deactivation endpoint + JWT invalidation | Not built | **Target Q4 2026** |
| CC7.1 | Monitoring | Structured logging (zerolog), audit middleware | Partial |
| CC7.2 | Anomaly detection | Rate limiting (fail-closed on auth), failed login tracking | Partial |
| CC8.1 | Change management | Git version control, CI/CD via GitHub Actions | Implemented |
| A1.1 | Availability | Health probes (`/health`, `/live`, `/ready`), graceful shutdown | Implemented |

## Data Classification

Encryption-at-rest is **NOT yet implemented** — see the PRD NFR table and FINDINGS F12.

| Data Type | Classification | Encryption in transit | Encryption at rest | Retention |
|-----------|---------------|----------------------|--------------------|-----------|
| Candidate CVs | Confidential | TLS 1.3 | **Target Q4 2026** (MinIO SSE-S3) | 90 days after last activity |
| Interview transcripts | Confidential | TLS 1.3 | **Target Q4 2026** (pgcrypto) | 90 days after interview completion |
| Evaluation scores | Confidential | TLS 1.3 | **Target Q4 2026** (pgcrypto) | 90 days after interview completion |
| User accounts | Internal | TLS 1.3 | n/a (hashes only) | Until deleted |
| Webhook payloads | Internal | TLS 1.3 | n/a | 30 days |

## Implementation Details

### Access Control (CC6.1, CC6.2, CC6.3)
- PostgreSQL Row-Level Security (FORCE RLS) on all tenant tables
- JWT tokens: `type=auth` for API, `ws_ticket` for WebSocket
- Role hierarchy: admin > recruiter > interviewer/member
- `intivai_app` DB role: least-privilege, no DDL
- Candidate auth: OTP-based with 7-day TTL, magic link single-use

### Monitoring (CC7.1)
- Zerolog structured logging on all API requests
- `pkg/db` transaction tracing with RLS context
- Asynq task queue monitoring (pending, processing, failed)
- MinIO access logging for object storage

### Availability (A1.1)
- Health endpoints: `GET /health`, `GET /live`, `GET /ready` (DB/Redis/MinIO checks)
- Graceful shutdown: SIGTERM handler with connection drain
- Database connection pooling via GORM
- Worker concurrency limits on Asynq
