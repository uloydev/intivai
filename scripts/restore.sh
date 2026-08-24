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

# Resolve the postgres container from the running prod stack (no hardcoded name).
POSTGRES_CT=$(docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml ps -q postgres)
[ -n "$POSTGRES_CT" ] || { echo "postgres container not found — is the prod stack up?"; exit 1; }

set -a; . ./.env.prod; set +a

# A11: DR roles must not be bootstrapped with hardcoded passwords. Both are
# required in .env.prod (see .env.prod.example) and MUST be rotated after the
# restore completes (reminder printed at script end).
: "${POSTGRES_APP_PASSWORD:?set POSTGRES_APP_PASSWORD in .env.prod}"
: "${POSTGRES_RLS_BYPASS_PASSWORD:?set POSTGRES_RLS_BYPASS_PASSWORD in .env.prod}"

if [[ "$DUMP" == /* ]]; then
  cp "$DUMP" "$WORK/restore.sql.gz"
else
  docker run --rm --network "$NETWORK" \
    -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
    -v "$WORK:/restore" \
    minio/mc cp "intivai/intivai-backups/$DUMP" /restore/restore.sql.gz >/dev/null
fi

echo "restoring $DUMP — this OVERWRITES the intivai database"
docker exec -i "$POSTGRES_CT" dropdb -U intivai --if-exists intivai_restore
docker exec -i "$POSTGRES_CT" createdb -U intivai intivai_restore

echo "bootstrapping roles..."
# A11: passwords come from .env.prod, never hardcoded. psql :'var' quoting is
# used because interpolation inside single-quoted SQL literals is unsafe;
# note it only works at TOP-LEVEL statements, not inside a DO $$ block (psql
# skips substitution in dollar-quoted strings) — hence shell-side checks.
PSQL="docker exec -i $POSTGRES_CT psql -U intivai -d postgres -v ON_ERROR_STOP=1"
APP_EXISTS=$($PSQL -tAc "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'intivai_app'" || true)
if [ "$APP_EXISTS" != "1" ]; then
  $PSQL -v app_pw="$POSTGRES_APP_PASSWORD" -c "CREATE ROLE intivai_app LOGIN PASSWORD :'app_pw'"
else
  echo "role intivai_app already exists — leaving its password alone"
fi
BYPASS_EXISTS=$($PSQL -tAc "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'intivai_rls_bypass'" || true)
if [ "$BYPASS_EXISTS" != "1" ]; then
  $PSQL -v bypass_pw="$POSTGRES_RLS_BYPASS_PASSWORD" -c "CREATE ROLE intivai_rls_bypass LOGIN PASSWORD :'bypass_pw' BYPASSRLS"
else
  echo "role intivai_rls_bypass already exists — leaving its password alone"
fi

gzip -dc "$WORK/restore.sql.gz" | docker exec -i "$POSTGRES_CT" psql -U intivai -d intivai_restore -v ON_ERROR_STOP=1 >/dev/null

echo "restore complete into database 'intivai_restore'"

echo "restoring object storage (cvs/) into the intivai bucket — OVERWRITES"
docker run --rm --network "$NETWORK" \
  -e "MC_HOST_intivai=http://${MINIO_ROOT_USER:-intivai}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  minio/mc mirror --overwrite "intivai/intivai-backups/cvs" "intivai/intivai" >/dev/null
echo "object storage restored"
echo "review it, then promote:"
echo "  docker exec -i $POSTGRES_CT psql -U intivai -c 'ALTER DATABASE intivai RENAME TO intivai_old'"
echo "  docker exec -i $POSTGRES_CT psql -U intivai -c 'ALTER DATABASE intivai_restore RENAME TO intivai'"

echo ""
echo "=== MANDATORY POST-RESTORE ROTATION ==="
echo "The restored dump contains historical password hashes for the DR roles."
echo "Rotate BOTH role passwords now, then update .env.prod to match:"
echo "  docker exec -i $POSTGRES_CT psql -U intivai -c \"ALTER ROLE intivai_app PASSWORD '<new-strong>'\""
echo "  docker exec -i $POSTGRES_CT psql -U intivai -c \"ALTER ROLE intivai_rls_bypass PASSWORD '<new-strong>'\""
echo "Then restart the app: docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d app"
