#!/usr/bin/env bash
# check-openapi-drift.sh — fail when a registered HTTP route is missing from api/openapi.yaml
# (or vice versa). Interim guard until spec-first codegen lands.
set -euo pipefail
cd "$(dirname "$0")/.."

MAIN="backend/cmd/server/main.go"
SPEC="api/openapi.yaml"

norm() { sed -E 's/:[a-zA-Z]+/{P}/g; s/\{[^}]+\}/{P}/g'; }

registered() {
  {
    grep -oE '(app|v1|publicRoutes|authRoutes|authed)\.(Get|Post|Put|Patch|Delete)\("[^"]+"' "$MAIN"
    grep -oE 'RegisterAt\((app|v1), "[^"]+"\)' "$MAIN" |
      sed -E 's/RegisterAt\([^,]+, ("[^"]+")\)/app.Get(\1/'
  } |
  while IFS= read -r line; do
    recv="${line%%.*}"
    case "$recv" in
      app) prefix="" ;;
      v1) prefix="/api/v1" ;;
      publicRoutes) prefix="/api/v1/public" ;;
      authRoutes) prefix="/api/v1/auth" ;;
      authed) prefix="/api/v1" ;;
    esac
    verb="$(sed -E 's/.*\.(Get|Post|Put|Patch|Delete)\(.*/\1/' <<<"$line")"
    path="$(sed -E 's/.*\("[^"]*"(.*)/\1/; s/^"//; s/".*$//' <<<"$line")"
    # path extraction: take the quoted literal
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
while IFS=$'\t' read -r reg spe; do
  [[ -n "${reg// /}" ]] && { echo "MISSING FROM SPEC: $reg" >&2; fail=1; }
  [[ -n "${spe// /}" ]] && { echo "IN SPEC BUT NOT REGISTERED: $spe" >&2; fail=1; }
done < <(comm -3 <(registered) <(spec_routes))

if (( fail )); then
  echo "openapi drift detected — update api/openapi.yaml" >&2
  exit 1
fi
echo "openapi in sync with backend/cmd/server/main.go routes"
