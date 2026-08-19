#!/usr/bin/env bash
# scripts/seed.sh
# Seeds the database with scenario-grouped SQL queries.
# Usage:
#   bash scripts/seed.sh [scenario_name]   (default: demo)
#   bash scripts/seed.sh --list
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEEDS_BASE_DIR="${SCRIPT_DIR}/seeds"
DB_URL="${DATABASE_URL:-postgres://intivai:intivai@localhost:5433/intivai?sslmode=disable}"

say() { printf '\n== %s ==\n' "$1"; }

# 1. Check for --list flag
if [ "${1:-}" = "--list" ] || [ "${1:-}" = "-l" ]; then
    echo "Available seed scenarios in ${SEEDS_BASE_DIR}:"
    for dir in "${SEEDS_BASE_DIR}"/*; do
        if [ -d "$dir" ]; then
            echo "  • $(basename "$dir")"
        fi
    done
    exit 0
fi

# 2. Determine target scenario
SCENARIO="${1:-${SCENARIO:-demo}}"
SCENARIO_DIR="${SEEDS_BASE_DIR}/${SCENARIO}"

if [ ! -d "$SCENARIO_DIR" ]; then
    echo "❌ Error: Seed scenario '${SCENARIO}' not found at ${SCENARIO_DIR}"
    echo "Available scenarios:"
    for dir in "${SEEDS_BASE_DIR}"/*; do
        [ -d "$dir" ] && echo "  • $(basename "$dir")"
    done
    exit 1
fi

say "Applying Seed Scenario: '${SCENARIO}' (${SCENARIO_DIR})"

run_sql() {
    local file="$1"
    local filename
    filename=$(basename "$file")
    printf "  ↳ [%s] %s... " "$SCENARIO" "$filename"

    if command -v psql >/dev/null 2>&1; then
        PSQL_ERR="$(mktemp)"
        if ! psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$file" >/dev/null 2>"$PSQL_ERR"; then
            # Fallback to docker compose if direct psql host port differs.
            # Surface the original error first — silent fallback hides drift.
            echo "direct psql failed — falling back to docker compose:"
            sed 's/^/    /' "$PSQL_ERR"
            rm -f "$PSQL_ERR"
            docker compose -f "${SCRIPT_DIR}/../docker-compose.yml" -f "${SCRIPT_DIR}/../docker-compose.dev.yml" \
                exec -T postgres psql -U intivai -d intivai -v ON_ERROR_STOP=1 < "$file" >/dev/null
        else
            rm -f "$PSQL_ERR"
        fi
    else
        docker compose -f "${SCRIPT_DIR}/../docker-compose.yml" -f "${SCRIPT_DIR}/../docker-compose.dev.yml" \
            exec -T postgres psql -U intivai -d intivai -v ON_ERROR_STOP=1 < "$file" >/dev/null
    fi
    echo "✓ Done"
}

for sql_file in "${SCENARIO_DIR}"/*.sql; do
    [ -f "$sql_file" ] || continue
    run_sql "$sql_file"
done

say "============================================================"
say "🎉 Scenario '${SCENARIO}' Applied Successfully!"
echo "• Recruiter Dashboard:  http://localhost:5173/dashboard"
echo "• Recruiter Login:      http://localhost:5173/login"
echo "  - Org Slug:           demo"
echo "  - Email:              admin@demo.io (or recruiter@demo.io)"
echo "  - Password:           password123"
echo "• Public Careers:       http://localhost:5173/careers"
echo "• Candidate Portal:     http://localhost:5173/candidate/portal"
echo "• Demo Candidate Link:  http://localhost:5173/interview/demo-invitation-token-david-chen-2026"
echo "• Mailpit Web UI:       http://localhost:8026"
say "============================================================"
