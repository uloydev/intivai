#!/usr/bin/env bash
# check-doc-links.sh — verify relative markdown links resolve (internal only).
# External http(s) links are not fetched here; lychee can be layered in CI later.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
while IFS= read -r -d '' f; do
  dir=$(dirname "$f")
  # extract markdown link targets that look like paths (skip urls, anchors, mailto)
  targets=$(grep -oE '\]\(([^)#]+)(#[^)]*)?\)' "$f" 2>/dev/null |
    sed -E 's/\]\(([^)#]+)(#[^)]*)?\)/\1/') || true
  [ -z "$targets" ] && continue
  while IFS= read -r target; do
    case "$target" in
      http://*|https://*|mailto:*|"") continue ;;
    esac
    if [ ! -e "$dir/$target" ]; then
      echo "BROKEN LINK in $f -> $target" >&2
      fail=1
    fi
  done <<< "$targets"
done < <(find . -name '*.md' \
  -not -path './.kilo/*' \
  -not -path './.kilocode/*' \
  -not -path './.opencode/*' \
  -not -path './node_modules/*' \
  -not -path '*/node_modules/*' \
  -not -path './docs/plans/archive/*' \
  -not -path './docs/reviews/*' \
  -print0)

if (( fail )); then echo "doc link check failed" >&2; exit 1; fi
echo "doc internal links OK"
