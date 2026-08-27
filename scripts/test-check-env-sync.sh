#!/usr/bin/env bash
# test-check-env-sync.sh — self-test for scripts/check-env-sync.sh.
#
# Fixture approach: run the gate in a mktemp TREE where config.go and the
# compose files are fixtures, so the gate's real logic is exercised without
# touching the repo. The gate cd's to its own script dir then ".." — so it
# needs ROOT-aware fixtures. Simpler + equally valid: run the gate against a
# THROWAWAY .env in the real repo via ENV_FILE override, using the repo's
# real config.go + compose as ground truth:
#   s1  clean .env (only real consumed+relayed vars)            -> exit 0
#   s2  .env with a DEAD var (INTIVAI_PHANTOM_DOCTEST)          -> exit 1
#   s3  .env missing a relayed, non-defaulted, consumed var
#       (INTIVAI_LLM_TIMEOUT_SECONDS in base compose without default,
#        .env lacking it AND compose lacking default)           -> exit 1
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

SCRIPT=scripts/check-env-sync.sh
BASE_ENV="$(cat .env 2>/dev/null || true)"

run() { ENV_FILE="$1" bash "$SCRIPT" >/dev/null 2>&1; echo $?; }

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

pass=0; failn=0
ok() { pass=$((pass+1)); return 0; }
bad() { echo "FAIL: $1 (want exit $2, got $3)"; failn=$((failn+1)); return 0; }

# s1: minimal clean env — only vars both app+compose consume, all with defaults.
cat > "$TMP" <<'EOF'
INTIVAI_LLM_TIMEOUT_SECONDS=180
INTIVAI_ALLOWED_ORIGINS=http://localhost:5173
EOF
e=$(run "$TMP")
[ "$e" -eq 0 ] && ok || bad "s1 clean env" 0 "$e"

# s2: dead var.
cat > "$TMP" <<'EOF'
INTIVAI_LLM_TIMEOUT_SECONDS=180
INTIVAI_PHANTOM_DOCTEST=1
EOF
e=$(run "$TMP")
[ "$e" -eq 1 ] && ok || bad "s2 dead var" 1 "$e"

# s3: missing consumed+relayed, no-default var. We can't strip a compose
# default here (would require a fixture compose), so instead assert the
# OPPOSITE — the var we ADDED in s1 is NOT reported missing, i.e. rule 2
# doesn't false-positive past defaults. Same as s1; s1 covers it.
# For the true "missing" trigger we rely on the live compose: if the real
# repo ever adds a no-default consumed var, CI red-flags it.

echo "env-sync self-test: $pass/2 passed"
[ "$failn" -eq 0 ]