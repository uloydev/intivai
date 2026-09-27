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

COV_FILE="$(mktemp)"
trap 'rm -f "$COV_FILE"' EXIT

go test -count=1 -cover ./... > "$COV_FILE" 2>&1 || true

while IFS= read -r line; do
  case "$line" in
    *"no test files"*)
      pkg=$(echo "$line" | awk '{print $2}')
      pct=0
      ;;
    $'\t'*coverage:*)
      pkg=$(echo "$line" | awk '{print $1}')
      pct=$(echo "$line" | grep -oE '[0-9.]+%' | head -1 | tr -d '%' || true)
      if [ -z "$pct" ]; then pct=0; fi
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
    pkg/*|*/pkg/*|*/shared/*|*/api|cmd/server|*/infrastructure/auth|*/memory/domain|*/notification/domain|github.com/intivai/backend/cmd/server) continue ;;
    # Tool/model-gated: tests skip without tesseract/poppler (ocr) or the
    # downloaded embedding model; covered when run inside the app image.
    */infrastructure/ocr|github.com/intivai/backend/internal/embedding|github.com/intivai/backend/cmd/loadcheck) continue ;;
    # ---- Explicit exemptions (CD11: rationale + owner + review date) ----
    # Each row: package -> rationale, owner, review date. PENDING TESTS are
    # booked in FINDINGS (I18); the review date is the test/remove-deadline.
    # Owner: EM. Review date: 2026-08-28.
    github.com/intivai/backend/cmd/sandboxd) continue ;; # bootstrap main; smoke covers boot path
    github.com/intivai/backend/internal/context/application) continue ;; # production-critical — tests booked 08-28
    github.com/intivai/backend/internal/context/infrastructure/persistence) continue ;; # RLS adapter — tests booked 08-28
    github.com/intivai/backend/internal/integration/application) continue ;; # webhook adapter — tests booked 08-28
    github.com/intivai/backend/internal/integration/infrastructure/persistence) continue ;; # webhook repo — tests booked 08-28
    github.com/intivai/backend/internal/interview/infrastructure/stt|github.com/intivai/backend/internal/interview/infrastructure/tts|github.com/intivai/backend/internal/interview/infrastructure/webrtc) continue ;; # voice demo-only (ADR-0006/0008)
    github.com/intivai/backend/internal/llm) continue ;; # provider adapter — client tests cover spans; provider tests booked 08-28
    github.com/intivai/backend/internal/memory/application) continue ;; # memory stub — 0 logic; sandbox-gated
    github.com/intivai/backend/internal/memory/infrastructure/native) continue ;; # native memory — 0 logic; sandbox-gated
    github.com/intivai/backend/internal/sandbox/application|github.com/intivai/backend/internal/sandbox/infrastructure/sidecarclient|github.com/intivai/backend/internal/sandbox/proto) continue ;; # execution adapter — covered by e2e/exec-image smoke
    github.com/intivai/backend/internal/screening/infrastructure/persistence) continue ;; # portal repo — tests booked 08-28
  esac

  floor=$FLOOR_ALL
  case "$pkg" in */domain) floor=$FLOOR_DOMAIN ;; esac

  if awk "BEGIN{exit !($pct < $floor)}"; then
    echo "LOW: $pkg ${pct}% < ${floor}%"
    FAIL=1
  fi
done < "$COV_FILE"

if [ "$FAIL" -eq 0 ]; then
  echo "coverage gate OK"
else
  echo "coverage gate FAILED — add tests or adjust exemptions"
  exit 1
fi
