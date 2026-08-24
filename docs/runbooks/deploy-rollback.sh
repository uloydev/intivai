#!/usr/bin/env bash
# deploy-rollback.sh — redeploy a previous app image tag on the VPS (A13).
# Executable form of docs/runbooks/rollback.md (policy lives there:
# migrations are forward-only, so this NEVER runs .down.sql).
# Usage (on the VPS, repo checked out at /opt/intivai):
#   bash docs/runbooks/deploy-rollback.sh <previous-git-sha-tag>
# Discover candidate tags (CI keeps the last N):
#   docker images "ghcr.io/$IMAGE" --format '{{.Tag}}'   # newest first
set -euo pipefail

cd "$(dirname "$0")/.."

TAG="${1:-${TAG:-}}"
if [ -z "$TAG" ]; then
  echo "usage: $0 <previous-tag>" >&2
  echo "candidates:" >&2
  set -a; . ./.env.prod; set +a
  : "${IMAGE:?set IMAGE in .env.prod}"
  docker images "ghcr.io/${IMAGE}" --format '  {{.Tag}}' | grep -v latest >&2 || true
  exit 2
fi

set -a; . ./.env.prod; set +a
: "${IMAGE:?set IMAGE in .env.prod}"
: "${INTIVAI_APP_PUBLIC_URL:=}"

COMPOSE="docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml"

echo "== rolling back to ghcr.io/${IMAGE}:${TAG} =="
docker pull "ghcr.io/${IMAGE}:${TAG}"
export TAG IMAGE

# Recreates only the changed containers; deliberately does NOT run the
# migrate service (forward-only policy — see docs/runbooks/rollback.md).
$COMPOSE up -d app

echo "== waiting for /health then /ready (max 5 min) =="
BASE="${INTIVAI_APP_PUBLIC_URL%/}"
if [ -z "$BASE" ]; then
  echo "INTIVAI_APP_PUBLIC_URL unset in .env.prod — cannot poll; verify manually." >&2
  exit 1
fi

deadline=$((SECONDS + 300))
until curl -fsS "$BASE/health" >/dev/null 2>&1 && curl -fsS "$BASE/ready" >/dev/null 2>&1; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "FAIL: app did not become healthy within 300s after rollback." >&2
    echo "Preserve logs first:  docker compose logs app > /tmp/app-\$(date +%s).log" >&2
    echo "Then retry with an OLDER tag:  $0 <older-tag>" >&2
    exit 3
  fi
  sleep 5
done

echo "== rollback OK: $TAG serving traffic =="
echo "Next: run make smoke BASE=$BASE (with CV_PDF + LLM key) before standing down."
