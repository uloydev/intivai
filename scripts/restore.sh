#!/usr/bin/env bash
# Restore a postgres dump from the MinIO backup bucket.
# Usage: scripts/restore.sh <dump-file>        (inside the bucket)
#    or: scripts/restore.sh /local/path.sql.gz (local file)
set -euo pipefail

cd "$(dirname "$0")/.."

DUMP="${1:?usage: restore.sh <dump-file-or-bucket-object>}"
NETWORK="intivai_default"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

set -a; . ./.env.prod; set +a

if [[ "$DUMP" == /* ]]; then
  cp "$DUMP" "$WORK/restore.sql.gz"
else
  docker run --rm --network "$NETWORK" \
    -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
    -v "$WORK:/restore" \
    minio/mc cp "intivai/intivai-backups/$DUMP" /restore/restore.sql.gz >/dev/null
fi

echo "restoring $DUMP — this OVERWRITES the intivai database"
docker exec -i intivai-postgres-1 dropdb -U intivai --if-exists intivai_restore
docker exec -i intivai-postgres-1 createdb -U intivai intivai_restore
gzip -dc "$WORK/restore.sql.gz" | docker exec -i intivai-postgres-1 psql -U intivai -d intivai_restore >/dev/null

echo "restore complete into database 'intivai_restore'"

echo "restoring object storage (cvs/) into the intivai bucket — OVERWRITES"
docker run --rm --network "$NETWORK" \
  -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  minio/mc mirror --overwrite "intivai/intivai-backups/cvs" "intivai/intivai" >/dev/null
echo "object storage restored"
echo "review it, then promote:"
echo "  docker exec -i intivai-postgres-1 psql -U intivai -c 'ALTER DATABASE intivai RENAME TO intivai_old'"
echo "  docker exec -i intivai-postgres-1 psql -U intivai -c 'ALTER DATABASE intivai_restore RENAME TO intivai'"
