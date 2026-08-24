# Data Retention Policy

> Status: corrected 2026-08-24 · Owner: EM
> Deletion methods marked **Target** are policy intent, not running jobs yet;
> they must ship (or the row change) before the first paying customer.
> Legal retention for hiring records may exceed these technical windows —
> confirm with counsel before enabling auto-delete in production.

## Retention Schedule

| Data Type | Retention Period | Deletion Method | Status |
|-----------|-----------------|-----------------|--------|
| Candidate CVs (files) | 90 days after last activity | MinIO lifecycle policy | **Target** (no lifecycle rule configured) |
| Candidate CVs (metadata) | 90 days after last activity | Cascade delete from `candidates` table | Target (candidate erasure path implemented; age-based purge not) |
| Interview transcripts | 90 days after interview completion | Cascade delete from `interviews` table | Target (same as above) |
| Evaluation reports | 90 days after interview completion | Cascade delete from `interviews` table | Target |
| Proctoring events | Raw events capped at 500/interview; summary retained with report | App-level cap + cascade | Implemented (cap) / Target (age purge) |
| Webhook deliveries | 30 days | Background purge job | **Target** |
| User accounts | Until manual deletion | Soft delete via deactivation endpoint | **Target** (endpoint not built — see access_control_policy) |
| OTP tokens | 7 days | `PurgeExpired` repository call on portal access | Implemented |
| Invitation tokens | 7 days validity; revoked rows purgeable | `PurgeExpired` job | Implemented (validity); Target (purge job) |
| Data request logs | 1 year | Annual purge | **Target** |
| Erased candidate data (in backups) | Until backup expiry (≤14 days after erasure) | Backup rotation ages erased rows/files out of archives | Accepted interim (legal OK'd; erasure is not immediate inside the 14-day backup window) — **Target**: crypto-erase / key-rotation on backups |

## GDPR Compliance

### Right to Access (Article 15)
- `GET /candidate/portal/export` returns all candidate data as JSON
- Export includes: applications, interviews, transcripts, scores

### Right to Erasure (Article 17)
- `DELETE /candidate/portal/me` triggers `candidate_erase()` function
- Cascade: candidates → applications → interviews → questions → scores
- Data request logged to `data_requests` table for audit trail

### Consent (Article 6/7)
- Explicit consent captured before interview start (`consent_given` boolean)
- Consent text displayed on invitation page
- Consent record includes timestamp and consent version

## Backup Policy
- Nightly pg_dump + MinIO mirror via `scripts/backup.sh` — **Implemented (on-server copy only)**
- Offsite copy: **Target — hard gate 2026-09-30 or first paying customer** (`docs/adr/0007-backup-offsite-deferral.md`)
- 14-day retention for backup archives
- Backup bucket separate from production storage; monthly restore drill required (`docs/runbooks/restore-drill.md`)
