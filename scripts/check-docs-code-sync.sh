#!/usr/bin/env bash
# check-docs-code-sync.sh — gate: LIVE documentation must reference REAL code
# paths, the CURRENT migration range, and EXISTING env vars. Complements
# check-doc-links.sh (link targets) and markdownlint (style): this one stops
# fact drift — a doc pointing at a renamed/moved file, a stale "001–025"
# migration claim, or an env var the code never reads is a lie in the repo.
#
# Scan scope: docs/** EXCEPT docs/plans/archive/** and docs/reviews/**
# (frozen historical context — not claims about today) and docs/FINDINGS.md
# (the findings LEDGER: it must record past states — "was 001–009, fixed to
# 001–025" is history, not a live claim; the evidence column is exempt).
#
# Checks:
#   1. Path refs: every backticked path-like ref resolves under one of
#      [root, docs/, backend/, frontend/, backend/internal/, frontend/src/].
#      Fenced code blocks are stripped first (pseudocode examples are not
#      claims).
#   2. Migration ranges: "001–NNN" in live docs must show the REAL max
#      migration number.
#   3. Env vars: backticked INTIVAI_* vars must be read somewhere in
#      backend/, frontend/, scripts/, Makefiles, or compose files.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Overrides (self-test only): DOCS_ROOT_DOCS / DOCS_ROOT_BACKEND let the test
# spin a fixture tree instead of the real repo.
DOCS_SCAN="${DOCS_SCAN_OVERRIDE:-$ROOT/docs}"
MIGR_DIR="${MIGR_DIR_OVERRIDE:-$ROOT/backend/pkg/db/migrations}"
CODE_DIRS="${CODE_DIRS_OVERRIDE:-}"
[ -n "$CODE_DIRS" ] || CODE_DIRS="$ROOT"

# Candidate roots the refs may resolve under (real repo: root, backend,
# frontend, backend/internal, frontend/src). Self-test: CODE_DIRS only.
roots=("$CODE_DIRS")
[ "$CODE_DIRS" = "$ROOT" ] && roots=("$ROOT" "$ROOT/backend" "$ROOT/frontend" \
  "$ROOT/backend/internal" "$ROOT/frontend/src")

fail=0
err() { echo "DOC-CODE SYNC: $1"; fail=1; }

# Strip fenced code blocks (```...```): pseudocode/path-comment examples are
# not claims about the tree.
strip_fences() {
  awk '/^```/ { inblock = !inblock; next } !inblock { print }' "$1"
}

STAGING="$(mktemp)"
trap 'rm -f "$STAGING"' EXIT

# ---- scan scope -------------------------------------------------------------
mapfile -t DOCS < <(find "$DOCS_SCAN" -name '*.md' \
  ! -path '*/plans/archive/*' ! -path '*/reviews/*' \
  ! -path '*/FINDINGS.md' | sort)
[ "${#DOCS[@]}" -gt 0 ] || { echo "docs-code sync: no live docs found"; exit 1; }

# ---- 1. path refs -----------------------------------------------------------
: > "$STAGING"
for f in "${DOCS[@]}"; do
  { strip_fences "$f" | grep -oE '`[^`]+`' | tr -d '`' || true; } \
    | while IFS= read -r ref; do printf '%s\t%s\n' "$f" "$ref"; done >> "$STAGING"
done

while IFS=$'\t' read -r f ref; do
  [ -n "$ref" ] || continue
  # Path-like only: must contain a slash; skip globs, URLs, emails,
  # protocol/selectors, and prose with spaces/punctuation.
  case "$ref" in
    *" "*|*\**|*\?*|*"("*|*")"*|*"="*|*":"*|*"#"*|*"|"*|*"{"*|*"}"*) continue ;;
    *"//"*|localhost*|http*|git@*|ssh*|\.\.?/*\.\.*) continue ;;
  esac
  case "$ref" in *"/"*) ;; *) continue ;; esac
  # Non-code-path tokens: Go import specifiers, HTTP routes, markdown links,
  # path patterns (NNN_<...>), and dangling short dirs (layout prose).
  case "$ref" in
    github.com/*|gorm.io/*|os/exec|ai.*/*|pkg/*/|/*|*"<"*|*".md"|*").sql"*) continue ;;
    domain/|application/|infrastructure/|pages/|api/|llm/) continue ;;
    # Third-party library/module names (ledongthuc/pdf, unidoc/unioffice,
    # golang-migrate/migrate, pressly/goose, tecnativa/docker-socket-proxy):
    # they describe DEPENDENCIES, not repo files. Only treat as file refs when
    # they resolve under a KNOWN repo root (backend/, frontend/, docs/, pkg/)
    # or were already validated.
    ledongthuc/pdf|unidoc/unioffice|golang-migrate/migrate|pressly/goose|tecnativa/docker-socket-proxy) continue ;;
  esac
  # One-segment dirs like "frontend/pages/" are layout shorthand, not files.
  case "$ref" in */) continue ;; esac

  found=""
  # Call-form refs (pkg/telemetry.Init → pkg/telemetry) resolve against the
  # package path only: drop ONE trailing .Identifier segment. Real file
  # extensions (.go/.ts/.sql/...) stay untouched.
  cand_ref="$ref"
  case "$ref" in
    *.go|*.ts|*.tsx|*.sql|*.md|*.json|*.yaml|*.yml|*.sh|*.mjs|*.js|*.pdf|*.csv|*.xml|*.proto|*.pem|*.env) ;;
    *)
      if [[ "$ref" =~ ^[^[:space:]]+\/[^[:space:]]*\.([A-Za-z0-9_]+)$ ]]; then
        cand_ref="${ref%.*}"
      fi
      ;;
  esac
  for cand in "${roots[@]}"; do
    if [ -e "$cand/$cand_ref" ]; then found=1; break; fi
  done
  [ -n "$found" ] || err "$f: path '$ref' does not exist"
done < "$STAGING"

# ---- 2. migration ranges ----------------------------------------------------
MAX="$(ls "$MIGR_DIR"/*_*.up.sql 2>/dev/null \
  | grep -oE '[0-9]{3}' | sort -n | tail -1)"
if [ -n "$MAX" ]; then
  for f in "${DOCS[@]}"; do
    # "001–025", "001-025", "001 to 025" — compare the declared max.
    while IFS= read -r hit; do
      decl="$(echo "$hit" | grep -oE '[0-9]{3}[[:space:]]*$' | tr -d '[:space:]' || true)"
      [ -n "$decl" ] || continue
      # normalize: strip the leading "001–" and compare the trailing bound
      lo="$(echo "$hit" | grep -oE '^[0-9]{3}' | tr -d '[:space:]' || true)"
      [ "${lo:-}" = "001" ] || continue # only 001–NNN patterns are claims
      if [ "$decl" != "$MAX" ]; then
        err "$f: migration range '001–$decl' — live max is 001–$MAX"
      fi
    done < <(strip_fences "$f" | grep -oE '001[–-][0-9]{3}|001[[:space:]]+to[[:space:]]+[0-9]{3}' || true)
  done
fi

# ---- 3. env vars ------------------------------------------------------------
ENVS="$(mktemp)"
trap 'rm -f "$ENVS" "$STAGING"' EXIT
for f in "${DOCS[@]}"; do
  strip_fences "$f" | grep -oE 'INTIVAI_[A-Z0-9_]+' || true
done | sort -u > "$ENVS"

while IFS= read -r v; do
  [ -n "$v" ] || continue
  if ! grep -rq -- "$v" "$CODE_DIRS" 2>/dev/null; then
    err "env '$v' referenced in docs but never read by code/compose" 
  fi
done < "$ENVS"

if [ "$fail" -ne 0 ]; then
  echo "docs-code sync gate FAILED"
  exit 1
fi
echo "docs-code sync ok"
