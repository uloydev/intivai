#!/usr/bin/env bash
# test-check-docs-code-sync.sh — self-test for scripts/check-docs-code-sync.sh.
#
# Scenarios (mirrors the pattern used by test-check-openapi-drift.sh):
#   1. live doc referencing an EXISTING path -> exit 0
#   2. live doc referencing a DEAD path -> exit 1 (path does not exist)
#   3. live doc claiming "001–009" when the fixture's real max is 032 -> exit 1
#   4. live doc referencing a REAL env var -> exit 0
#   5. live doc referencing a PHANTOM env var -> exit 1
#   6. findings ledger (FINDINGS.md) is EXEMPT: historical "001–009" text
#      inside it must NOT fail the gate
#
# Fixtures are mktemp trees; gate overrides point at them.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/check-docs-code-sync.sh"

pass=0; failn=0
check_case() { # name expected_exit actual_exit
  local name="$1" want="$2" got="$3"
  if [ "$want" -eq "$got" ]; then ((pass++)); else ((failn++)); echo "FAIL: $name (want exit $want, got $got)"; fi
}

# fixture root
FX="$(mktemp -d)"
trap 'rm -rf "$FX"' EXIT
DOCS="$FX/docs"; MIG="$FX/migrations"; CODE="$FX/code"
mkdir -p "$DOCS/engineering" "$MIG" "$CODE" "$CODE/backend/internal/x"

# fake but real-looking tree
: > "$MIG/001_init.up.sql"
: > "$MIG/032_latest.up.sql"
printf '%s\n' 'TYPE=db' > "$CODE/backend/internal/x/thing.go"
grep -rq nothing "$CODE" 2>/dev/null || true

# doc that uses only real refs
cat > "$DOCS/engineering/ok.md" <<'EOF'
# Live doc
- path ref: `backend/internal/x/thing.go` exists
- env: reads `INTIVAI_DB_DOCTEST` in code below
EOF
mkdir -p "$CODE/backend/internal/x"
printf 'var x = "INTIVAI_DB_DOCTEST"\n' > "$CODE/backend/internal/x/settings.go"

# scenario 1 — clean
run() { DOCS_SCAN_OVERRIDE="$DOCS" MIGR_DIR_OVERRIDE="$MIG" CODE_DIRS_OVERRIDE="$CODE" bash "$GATE" >/dev/null 2>&1; echo $?; }
check_case "scenario 1: clean doc" 0 "$(run)"

# scenario 2 — dead path
cat > "$DOCS/engineering/dead.md" <<'EOF'
- `backend/internal/x/ghost.go` — missing in fixture
EOF
check_case "scenario 2: dead path" 1 "$(run)"
rm "$DOCS/engineering/dead.md"

# scenario 3 — stale migration range
cat > "$DOCS/engineering/range.md" <<'EOF'
Migrations 001–009 are applied.
EOF
check_case "scenario 3: stale range 001-009 vs 032" 1 "$(run)"
rm "$DOCS/engineering/range.md"

# scenario 4 — real env
cat > "$DOCS/engineering/env.md" <<'EOF'
Uses `INTIVAI_DB_DOCTEST` at runtime.
EOF
check_case "scenario 4: real env ok" 0 "$(run)"
rm "$DOCS/engineering/env.md"

# scenario 5 — phantom env
cat > "$DOCS/engineering/env.md" <<'EOF'
Uses `INTIVAI_PHANTOM_DOCTEST` at runtime.
EOF
check_case "scenario 5: phantom env" 1 "$(run)"
rm "$DOCS/engineering/env.md"

# scenario 6 — FINDINGS.md ledger exemption
cat > "$DOCS/FINDINGS.md" <<'EOF'
| H4 | stale range 001–009 fixed to 001–025 (history, not a live claim) |
EOF
check_case "scenario 6: FINDINGS ledger exempt" 0 "$(run)"

echo "docs-code-sync self-test: $pass/6 passed"
[ "$failn" -eq 0 ]
