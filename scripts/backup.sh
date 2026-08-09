#!/usr/bin/env bash
# Backup: pg_dump → MinIO (bucket intivai-backups), 14-day retention.
# Run nightly on the VPS, e.g. cron:
#   0 2 * * * /opt/intivai/scripts/backup.sh >> /var/log/intivai-backup.log 2>&1
# Restore: scripts/restore.sh <dump-file>
set -euo pipefail

cd "$(dirname "$0")/.."

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"
NETWORK="intivai_default"
BACKUP_DIR="${BACKUP_DIR:-/tmp/intivai-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
STAMP="$(date +%Y%m%d-%H%M%S)"
DUMP="intivai-${STAMP}.sql.gz"

mkdir -p "$BACKUP_DIR"

# 1. Dump postgres (admin user, plain format, compressed).
docker exec intivai-postgres-1 pg_dump -U intivai -d intivai --no-owner | gzip > "$BACKUP_DIR/$DUMP"
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
