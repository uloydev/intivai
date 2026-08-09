#!/usr/bin/env bash
# Coverage gate: every non-exempt package must meet its floor.
# Floors: domain packages >= 70%, all others >= 50%.
# Exempt (thin glue / no logic or covered by smoke): pkg/*, shared/*,
# */api handlers, cmd/server, iam auth adapter, memory/domain (interface).
# Requires TEST_DATABASE_URL + TEST_REDIS_ADDR so integration-gated tests
# count (use make coverage).
set -euo pipefail

cd "$(dirname "$0")/../backend"

FLOOR_ALL=50
FLOOR_DOMAIN=70
FAIL=0

go test -cover ./... > /tmp/intivai-coverage.txt 2>&1 || true

while IFS= read -r line; do
  case "$line" in
    *"no test files"*)
      pkg=$(echo "$line" | awk '{print $2}')
      pct=0
      ;;
    ok\ *|FAIL\ *)
      pkg=$(echo "$line" | awk '{print $2}')
      pct=$(echo "$line" | grep -oE '[0-9.]+%' | head -1 | tr -d '%' || true)
      if [ -z "$pct" ]; then
        case "$line" in
          *\[no\ statements\]*) continue ;; # test-only package, nothing to cover
        esac
        # e.g. "FAIL pkg [build failed]" — treat as gate failure.
        echo "BROKEN: $line"
        FAIL=1
        continue
      fi
      ;;
    *) continue ;;
  esac

  case "$pkg" in
    pkg/*|*/shared/*|*/api|cmd/server|*/infrastructure/auth|*/memory/domain|*/pkg/db/migrations|github.com/intivai/backend/cmd/server) continue ;;
  esac

  floor=$FLOOR_ALL
  case "$pkg" in */domain) floor=$FLOOR_DOMAIN ;; esac

  if awk "BEGIN{exit !($pct < $floor)}"; then
    echo "LOW: $pkg ${pct}% < ${floor}%"
    FAIL=1
  fi
done < /tmp/intivai-coverage.txt

if [ "$FAIL" -eq 0 ]; then
  echo "coverage gate OK"
else
  echo "coverage gate FAILED — add tests or adjust exemptions"
  exit 1
fi
