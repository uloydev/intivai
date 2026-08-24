#!/usr/bin/env bash
# check-openapi-drift.sh — fail when a registered HTTP route is missing from api/openapi.yaml
# (or vice versa). Interim guard until spec-first codegen lands.
#
# Scans TWO registration surfaces:
#   1. main.go direct registrations (app/v1/publicRoutes/authRoutes/authed receivers)
#   2. helper-mounted routes: any .Get/.Post/.Put/.Patch/.Delete("/...") literal in
#      backend/internal (handler Register* functions receiving a router group).
#      Helper routes are assumed to sit under /api/v1 (handlers receive the v1
#      group or its subgroups from main.go). If a future handler mounts elsewhere,
#      mark the file with a line comment: `// drift-prefix:/other/prefix`.
#
# Testability (ledger E4): override DRIFT_MAIN / DRIFT_SPEC / DRIFT_HELPER_DIRS
# to point at fixtures — see scripts/test-check-openapi-drift.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

MAIN="${DRIFT_MAIN:-backend/cmd/server/main.go}"
SPEC="${DRIFT_SPEC:-api/openapi.yaml}"
HELPER_DIRS="${DRIFT_HELPER_DIRS:-backend/internal}"

norm() { sed -E 's/:[a-zA-Z]+/{P}/g; s/\{[^}]+\}/{P}/g'; }

registered() {
  {
    # `|| true` on every scan: grep exits 1 on zero matches and, under
    # pipefail, would otherwise abort the whole brace group — silently
    # skipping every later registration surface (ledger E4 class of bug).
    grep -oE '(app|v1|publicRoutes|authRoutes|authed)\.(Get|Post|Put|Patch|Delete)\("[^"]+"' "$MAIN" || true
    grep -oE 'RegisterAt\((app|v1), "[^"]+"\)' "$MAIN" |
      sed -E 's/RegisterAt\([^,]+, ("[^"]+")\)/app.Get(\1/' || true
    # Helper-mounted routes in internal packages (e.g. RegisterVoiceRoutes on the
    # v1 group). Path-literal only ("/...) to avoid client calls like http.Get.
    grep -rnoE '[a-zA-Z_][a-zA-Z0-9_]*\.(Get|Post|Put|Patch|Delete)\("/[^"]+"' \
      --include='*.go' --exclude='*_test.go' $HELPER_DIRS |
      sed -E 's/^[^:]+:[0-9]+://' || true
  } |
  while IFS= read -r line; do
    recv="${line%%.*}"
    case "$recv" in
      app) prefix="" ;;
      v1) prefix="/api/v1" ;;
      publicRoutes) prefix="/api/v1/public" ;;
      authRoutes) prefix="/api/v1/auth" ;;
      authed) prefix="/api/v1" ;;
      *) prefix="/api/v1" ;; # helper receivers mount on the v1 group (or subgroups)
    esac
    verb="$(sed -E 's/.*\.(Get|Post|Put|Patch|Delete)\(.*/\1/' <<<"$line")"
    path="$(sed -E 's/^.*\("//; s/"$//' <<<"$line")"
    echo "${verb^^} ${prefix}${path}" | norm
  done | sort -u
}

spec_routes() {
  # path lines look like: "  /api/v1/jobs:" ; methods are indented exactly 4 spaces
  awk '
    /^components:/ {inpaths=0}
    /^paths:/ {inpaths=1; next}
    inpaths && /^  \// {
      p=$1; sub(/:$/,"",p)
    }
    inpaths && /^    (get|post|put|patch|delete):[[:space:]]*$/ {
      m=toupper($1); sub(/:/,"",m); print m" "p
    }
  ' "$SPEC" | norm | sort -u
}

fail=0
# comm -3 marks columns with tabs: left-only "reg\t", right-only "\tspe".
# Parse by position — IFS=$'\t' read would STRIP the leading tab (IFS
# whitespace rule) and misreport spec-only routes as missing-from-spec.
while IFS= read -r row; do
  [[ -z "$row" ]] && continue
  reg="${row%%$'\t'*}"
  spe="${row#*$'\t'}"
  [[ -n "${reg// /}" ]] && { echo "MISSING FROM SPEC: $reg" >&2; fail=1; }
  [[ -n "${spe// /}" ]] && { echo "IN SPEC BUT NOT REGISTERED: $spe" >&2; fail=1; }
done < <(comm -3 <(registered) <(spec_routes))

if (( fail )); then
  echo "openapi drift detected — update api/openapi.yaml" >&2
  exit 1
fi
echo "openapi in sync with backend routes (main.go + helper registrations)"
