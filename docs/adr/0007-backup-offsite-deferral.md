# ADR-0007: Backups stay on-server until first paying customer / 2026-09-30

- **Status:** accepted (records P4_Plan D5; hard deadline unchanged)
- **Date:** 2026-08-10 (decision), recorded as ADR 2026-08-24
- **Deciders:** EM

## Context

The design docs mandate offsite backups ("1 server + 0 backups = losing everything"; RPO ≤ 24h, monthly restore tests). At beta scale, an offsite copy (Backblaze B2/S3 via rclone) was deferred for cost control — but the deferral needed a permanent, dated record instead of a line inside a plan file.

## Decision

Nightly `pg_dump` + MinIO mirror target the **local** backup bucket only. Offsite `rclone sync` to B2/S3 must be added **no later than the earlier of: first paying customer or 2026-09-30**. This is a customer gate, not a soft date. Restore drills (`scripts/restore.sh`) run monthly and must recreate the `intivai_app` role/privileges/RLS, not just data.

## Alternatives Considered

| Option | Why not |
|---|---|
| Offsite immediately | Beta cost control; single-copy risk accepted for a bounded window |
| No backup | Unacceptable — survival requirement |

## Consequences

- Server loss before the gate = total data loss; risk explicitly accepted and dated.
- Beta gate item #10 stays open until cron + first successful restore test.
- If the deadline passes without offsite copy, this ADR's status flips to violated — treat as a launch blocker.
