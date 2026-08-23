# Runbook — Restore Drill (monthly)

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> A backup never restored is not a backup. Run this drill monthly (beta gate
> #10; ADR-0007 for the offsite gap).

## Frequency & evidence

- **Monthly**, before the first of the month ends.
- Record in the drill log below: date, dump age, restore duration, verifier.

## Procedure

```bash
# 1. Pick the latest dump from the backup bucket
mc ls local/intivai-backups/ | tail -5

# 2. Restore into the isolated restore database (never over prod in place)
scripts/restore.sh <dump-file>
#    → restores DB into intivai_restore + mirrors objects to the restore bucket

# 3. Verify the least-privilege role + RLS came back
psql "$INTIVAI_RESTORE_URL" -c "\du intivai_app"          # role exists, NOSUPERUSER, no BYPASSRLS
psql "$INTIVAI_RESTORE_URL" -c "SELECT relname, relrowsecurity FROM pg_class WHERE relname IN ('candidates','jobs','interviews');"   # relrowsecurity = t

# 4. Boot a throwaway app stack against the restore DB and smoke it
INTIVAI_DATABASE_URL="$INTIVAI_RESTORE_URL" make up
make smoke BASE=http://localhost:8081 CV_PDF=/tmp/kilo/cv.pdf

# 5. Tear down the throwaway stack
make down
```

## Pass criteria

- `intivai_app` role recreated with correct privileges (`restore.sh` bootstrap step).
- RLS flags on all tenant tables.
- Smoke passes end-to-end against restored data.
- Total time recorded; RTO target ≤ 1h (design-decisions §4 backup block).

## Drill log

| Date | Dump age | Duration | Result | Notes |
|---|---|---|---|---|
| — | — | — | — | first drill pending (beta gate #10) |
