#!/usr/bin/env bash
# check-env-sync.sh — gate: the local dev .env must not quietly drift from
# what the app/compose actually consume, and must not carry dead vars.
#
# Checks (run from repo root; pass ENV_FILE= to point at another file):
#   1. DEAD VAR: a var in .env that NEITHER pkg/config reads NOR
#      docker-compose.yml(dev) relays → it does nothing; someone added it
#      expecting an effect. (Real case: INTIVAI_LLM_TIMEOUT_SECONDS was in
#      .env but compose never passed it through → app silently used 60s.)
#   2. MISSING CONSUMED VAR: a var the app reads AND compose relays in the
#      DEV stack (base+dev overlay) that is absent from .env AND has no
#      compose default → the stack boots with an unset value.
#   3. DEVIATION FROM .env.example: helpful diff hint, not a failure —
#      the local .env legitimately differs (own key, provider choice).
#
# NOT a secret scanner: .env is git-ignored; this script is an ops guard.
set -euo pipefail
cd "$(dirname "$0")/.."

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE=(docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f docker-compose.dev.yml)

fail=0
err() { echo "ENV SYNC: $1"; fail=1; }

# ---- var sets ---------------------------------------------------------------
# App-consumed (INTIVAI_ prefixed) — config.go getString/getInt/bool + viper.
APP_CONSUMED="$(grep -oE '(get(String|Int|Bool)|v\.Get(String|Int|Bool))\("[A-Z0-9_]+"' backend/pkg/config/config.go \
  | grep -oE '"[A-Z0-9_]+"' | tr -d '"' | sort -u)"
# Sanity: config uses SetEnvPrefix("INTIVAI"), so keys prefix to INTIVAI_*.
APP_CONSUMED="$(for k in $APP_CONSUMED; do echo "INTIVAI_$k"; done | sort -u)"

# Compose-relayed dev vars: every INTIVAI_* referenced in base+dev compose
# (both interpolation ${..} and literal `INTIVAI_X:` entries).
COMPOSE_VARS="$(grep -hEo 'INTIVAI_[A-Z0-9_]+' docker-compose.yml docker-compose.dev.yml \
  | sort -u || true)"

ENV_VARS="$(grep -oE '^[A-Z_][A-Z0-9_]*=' "$ENV_FILE" 2>/dev/null \
  | sed 's/=$//' | sed 's/=//' | sort -u || true)"

# Compose vars that have a hardcoded default in the base compose (i.e. the
# stack is self-sufficient without .env providing them).
HAS_DEFAULT="$(
  grep -nE 'INTIVAI_[A-Z0-9_]+:' docker-compose.yml \
  | grep -oE 'INTIVAI_[A-Z0-9_]+' | sort -u || true
)"

# ---- 1. dead vars -----------------------------------------------------------
echo "checking $ENV_FILE against app consumption + compose relay..."
for v in $ENV_VARS; do
  case "$v" in
    INTIVAI_*)
      if ! printf '%s\n' "$APP_CONSUMED" | grep -qx "$v" \
         && ! printf '%s\n' "$COMPOSE_VARS" | grep -qx "$v"; then
        err "dead var: $v is neither read by pkg/config nor relayed by compose — remove it or wire it"
      fi
      ;;
  esac
done

# ---- 2. missing consumed vars ----------------------------------------------
# Relayed IN compose (they would break/boot wrong if empty) but absent from
# .env with no hardcoded default → tell the user to add.
for v in $COMPOSE_VARS; do
  printf '%s\n' "$HAS_DEFAULT" | grep -qx "$v" && continue
  printf '%s\n' "$ENV_VARS" | grep -qx "$v" && continue
  if printf '%s\n' "$APP_CONSUMED" | grep -qx "$v"; then
    err "missing: $v is consumed by the app and relayed with NO compose default — add it to $ENV_FILE (see .env.example)"
  fi
done

# ---- 3. example diff hint ----------------------------------------------------
if [ -f .env.example ]; then
  EXAMPLE="$(sort .env.example | sed 's/=.*/=/')"
  ACTUAL="$(sort "$ENV_FILE" | sed 's/=.*/=/')"
  if [ "$EXAMPLE" != "$ACTUAL" ]; then
    echo "note: $ENV_FILE deviates from .env.example (expected — local keys/providers differ)"
  fi
fi

if [ "$fail" -ne 0 ]; then
  echo "env-sync gate FAILED"
  exit 1
fi
echo "env-sync ok"
