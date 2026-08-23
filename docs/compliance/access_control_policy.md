# Access Control Policy

> Status: corrected 2026-08-24 · Owner: EM
> Rule: this policy states what IS, not what is planned. Planned controls are
> marked **Target-\<quarter\>** and tracked in `docs/FINDINGS.md`.

## User Roles

| Role | Permissions | Rank |
|------|------------|------|
| admin | Full CRUD on all org resources, user management, billing | 0 (highest) |
| recruiter | CRUD on jobs, candidates, interviews, webhooks | 1 |
| interviewer/member | Read-only on most resources | 2 (lowest) |

## Role Assignment Rules

- Users can only create roles at or below their own rank
- A user cannot promote another user above their own role
- Role changes are logged with before/after values

## Authentication

### Internal Users
- Password: bcrypt with min 8 characters — **Implemented**
- Session: JWT (24h expiry intent; `type=auth` tokens only on API routes) — **Implemented**. Refresh-token rotation — **Target Q4 2026 (not implemented)**
- MFA — **Target 2027 (roadmap)**

### Candidates
- Passwordless: SHA256 hashed OTP sent via email — **Implemented**
- Magic link: single-use token in invitation/portal emails — **Implemented**
- No password requirement (candidate portal only) — **Implemented**

## Authorization

### API Layer
- All authenticated routes use `RequireActor` middleware
- RBAC enforced in use-case layer via `Authorize(actor, roles...)`
- No route bypasses authorization

### Database Layer
- PostgreSQL RLS (ENABLE + FORCE) on all tenant-scoped tables — **Implemented** (migration 002)
- App connects as `intivai_app` (non-superuser, no BYPASSRLS)
- Migrations connect as `INTIVAI_MIGRATE_URL` (separate role with DDL)
- Tenant isolation: `SET app.org_id = '<uuid>'` per transaction

## Access Removal

- User deactivation: `DELETE /api/v1/users/:id` soft delete + JWT invalidation — **Target Q4 2026 (endpoint not implemented)**. Until shipped, offboarding = password reset + role downgrade to member by an admin.
- Candidates: GDPR data erasure via `DELETE /candidate/portal/me` — **Implemented** (migration 024 audit trail)
