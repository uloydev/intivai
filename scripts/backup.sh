#!/usr/bin/env bash
# Backup: pg_dump → MinIO (bucket intivai-backups), 14-day retention,
# optional offsite copy via OFFSITE_DEST (rclone remote:path or s3://bucket/path).
# Run nightly on the VPS, e.g. cron:
#   0 2 * * * root OFFSITE_DEST='b2:intivai-backups/prod/' /opt/intivai/scripts/backup.sh >> /var/log/intivai-backup.log 2>&1
# Restore: scripts/restore.sh <dump-file>
set -euo pipefail

cd "$(dirname "$0")/.."

NETWORK="intivai_default"
BACKUP_DIR="${BACKUP_DIR:-/tmp/intivai-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
STAMP="$(date +%Y%m%d-%H%M%S)"
DUMP="intivai-${STAMP}.sql.gz"
OFFSITE_DEST="${OFFSITE_DEST:-}"

# Resolve the postgres container from the running prod stack (no hardcoded name).
POSTGRES_CT=$(docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml ps -q postgres)
[ -n "$POSTGRES_CT" ] || { echo "postgres container not found — is the prod stack up?"; exit 1; }

mkdir -p "$BACKUP_DIR"

# 1. Dump postgres (admin user, plain format, compressed).
docker exec "$POSTGRES_CT" pg_dump -U intivai -d intivai --no-owner | gzip > "$BACKUP_DIR/$DUMP"
echo "dumped: $BACKUP_DIR/$DUMP ($(du -h "$BACKUP_DIR/$DUMP" | cut -f1))"

# 2. Push to MinIO (internal network, no published ports needed).
# MinIO creds come from .env.prod via the compose env file.
set -a; . ./.env.prod; set +a
docker run --rm --network "$NETWORK" \
  -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  -v "$BACKUP_DIR:/backups:ro" \
  minio/mc mirror --overwrite /backups "intivai/intivai-backups" >/dev/null
echo "mirrored to minio://intivai-backups"

# 3. Object storage (CVs, company contexts) — same bucket, cvs/ prefix.
# CV files are candidate data; losing MinIO = losing all candidates.
docker run --rm --network "$NETWORK" \
  -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  minio/mc mirror --overwrite "intivai/intivai" "intivai/intivai-backups/cvs" >/dev/null
echo "mirrored object storage to minio://intivai-backups/cvs"

# 4. Retention: drop dumps older than RETENTION_DAYS (local + remote).
find "$BACKUP_DIR" -name 'intivai-*.sql.gz' -mtime "+${RETENTION_DAYS}" -delete
for REMOTE in "intivai/intivai-backups" "intivai/intivai-backups/cvs"; do
  docker run --rm --network "$NETWORK" \
    -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
    minio/mc rm --recursive --older-than "${RETENTION_DAYS}d" --force "$REMOTE" >/dev/null || true
done
echo "retention: ${RETENTION_DAYS}d applied"

# 5. Offsite copy (DR hard gate 2026-09-30). Configure via OFFSITE_DEST, e.g.
#    OFFSITE_DEST='b2:intivai-backups/prod/' (rclone remote:path) or
#    OFFSITE_DEST='s3://bucket/intivai-backups/' (aws s3). Needs rclone/aws
#    installed and configured on the host. No-op when unset.
if [ -n "$OFFSITE_DEST" ]; then
  if command -v rclone >/dev/null 2>&1; then
    rclone copy "$BACKUP_DIR/$DUMP" "$OFFSITE_DEST" >/dev/null
    echo "offsite copy pushed to $OFFSITE_DEST"
  elif command -v aws >/dev/null 2>&1; then
    aws s3 cp "$BACKUP_DIR/$DUMP" "$OFFSITE_DEST" >/dev/null
    echo "offsite copy pushed to $OFFSITE_DEST"
  else
    echo "OFFSITE_DEST set ($OFFSITE_DEST) but neither rclone nor aws installed — offsite copy SKIPPED"
  fi
else
  echo "offsite copy skipped (OFFSITE_DEST unset)"
fi
