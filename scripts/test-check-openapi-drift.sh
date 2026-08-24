#!/usr/bin/env bash
# test-check-openapi-drift.sh — self-test for scripts/check-openapi-drift.sh.
#
# Covers ledger finding E4 (helper-mounted routes escaping the drift guard):
#   1. route registered inside an internal handler Register* fn, absent from
#      the spec -> exit 1 with MISSING FROM SPEC on stderr
#   2. same fixture WITH the route present in the spec -> exit 0
#   3. main.go-registered route absent from the spec -> still detected
#      (regression of the pre-E4 behavior)
#   4. fully synced fixtures -> exit 0
#   5. spec-only route -> reported as IN SPEC BUT NOT REGISTERED (direction
#      guard for the comm -3 leading-tab parsing)
#
# Fixtures are built in mktemp -d trees and wired into the guard via the
# DRIFT_MAIN / DRIFT_SPEC / DRIFT_HELPER_DIRS overrides. The guard's norm()
# maps both :param and {param} to {P}, so ":id" in code matches "{id}" in spec.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="$SCRIPT_DIR/check-openapi-drift.sh"

if [[ ! -f "$GUARD" ]]; then
	echo "FATAL: guard script not found: $GUARD" >&2
	exit 2
fi

FIXTURE_PATHS=()

cleanup() {
	if ((${#FIXTURE_PATHS[@]})); then
		rm -rf "${FIXTURE_PATHS[@]}"
	fi
	return 0
}
trap cleanup EXIT

MAIN_FIXTURE=""
SPEC_FIXTURE=""
HELPERS_FIXTURE=""

new_fixture() {
	local d
	d="$(mktemp -d "${TMPDIR:-/tmp}/kilo-drift-test.XXXXXX")" || return 1
	FIXTURE_PATHS+=("$d")
	mkdir -p "$d/internal/chat"
	MAIN_FIXTURE="$d/main.go"
	SPEC_FIXTURE="$d/spec.yaml"
	HELPERS_FIXTURE="$d/internal"
}

write_main_no_routes() {
	cat >"$MAIN_FIXTURE" <<'EOF'
package main

func setup(app *fiber.App) {}
EOF
}

write_main_livez_jobs() {
	cat >"$MAIN_FIXTURE" <<'EOF'
package main

func setup(app *fiber.App, v1 fiber.Router) {
	app.Get("/livez", handleLive)
	v1.Get("/jobs", handleListJobs)
}
EOF
}

write_helper_fake_get() {
	cat >"$HELPERS_FIXTURE/chat/fake.go" <<'EOF'
package chat

import "github.com/gofiber/fiber/v2"

func (h *FooHandler) RegisterFakeRoutes(router fiber.Router) {
	router.Get("/interviews/:id/fake", h.handler)
}
EOF
}

write_helper_voice_post() {
	cat >"$HELPERS_FIXTURE/chat/voice.go" <<'EOF'
package chat

import "github.com/gofiber/fiber/v2"

func (h *chatHandler) RegisterVoiceRoutes(router fiber.Router) {
	router.Post("/interviews/:id/voice", h.handleVoice)
}
EOF
}

# write_spec PATH|METHOD ... — emits an OpenAPI skeleton with those operations,
# e.g. write_spec '/api/v1/jobs|get'. Paths use {param}; guard norm() aligns
# them with :param in Go code.
write_spec() {
	local entry path method
	{
		echo "openapi: 3.0.3"
		echo "info:"
		echo "  title: drift-fixture"
		echo '  version: "0.0.0"'
		echo "paths:"
		for entry in "$@"; do
			path="${entry%|*}"
			method="${entry##*|}"
			echo "  $path:"
			echo "    $method:"
			echo "      summary: fixture operation"
		done
	} >"$SPEC_FIXTURE"
}

GUARD_RC=0
GUARD_STDOUT=""
GUARD_STDERR=""

run_guard() {
	local out err rc=0
	out="$(mktemp "${TMPDIR:-/tmp}/kilo-drift-out.XXXXXX")"
	err="$(mktemp "${TMPDIR:-/tmp}/kilo-drift-err.XXXXXX")"
	FIXTURE_PATHS+=("$out" "$err")
	DRIFT_MAIN="$MAIN_FIXTURE" \
		DRIFT_SPEC="$SPEC_FIXTURE" \
		DRIFT_HELPER_DIRS="$HELPERS_FIXTURE" \
		bash "$GUARD" >"$out" 2>"$err" || rc=$?
	GUARD_RC=$rc
	GUARD_STDOUT="$out"
	GUARD_STDERR="$err"
}

PASSED=0
FAILED=0

# assert_case NAME WANT_EXIT [STDERR_REGEX] [STDOUT_REGEX]
assert_case() {
	local name="$1" want_rc="$2" err_re="${3:-}" out_re="${4:-}"
	local problem=""
	if ((GUARD_RC != want_rc)); then
		problem="exit=${GUARD_RC}, want ${want_rc}"
	fi
	if [[ -n "$err_re" ]] && ! grep -Eq -- "$err_re" "$GUARD_STDERR"; then
		problem+="${problem:+; }stderr missing /${err_re}/"
	fi
	if [[ -n "$out_re" ]] && ! grep -Eq -- "$out_re" "$GUARD_STDOUT"; then
		problem+="${problem:+; }stdout missing /${out_re}/"
	fi
	if [[ -z "$problem" ]]; then
		printf 'PASS %s\n' "$name"
		PASSED=$((PASSED + 1))
		return 0
	fi
	printf 'FAIL %s (%s)\n' "$name" "$problem"
	printf '  --- guard stdout ---\n'
	sed 's/^/  | /' "$GUARD_STDOUT"
	printf '  --- guard stderr ---\n'
	sed 's/^/  | /' "$GUARD_STDERR"
	FAILED=$((FAILED + 1))
}

# --- Case 1 (E4): helper-mounted route missing from spec --------------------
new_fixture
write_main_no_routes
write_helper_fake_get
write_spec
run_guard
assert_case \
	"helper-mounted route missing from spec -> exit 1 + MISSING FROM SPEC" 1 \
	'^MISSING FROM SPEC: GET /api/v1/interviews/\{P\}/fake$'

# --- Case 2: same helper fixture WITH the route in the spec -----------------
new_fixture
write_main_no_routes
write_helper_fake_get
write_spec '/api/v1/interviews/{id}/fake|get'
run_guard
assert_case \
	"helper route present in spec -> exit 0 (no false positive)" 0 \
	'' 'in sync'

# --- Case 3: main.go registration missing from spec (pre-E4 behavior) -------
new_fixture
write_main_livez_jobs
write_spec
run_guard
assert_case \
	"main.go-registered route missing from spec -> exit 1" 1 \
	'^MISSING FROM SPEC: GET /livez$'

# --- Case 4: everything in sync ---------------------------------------------
new_fixture
write_main_livez_jobs
write_helper_voice_post
write_spec '/livez|get' '/api/v1/jobs|get' '/api/v1/interviews/{id}/voice|post'
run_guard
assert_case \
	"synced fixtures -> exit 0" 0 \
	'' 'in sync'

# --- Case 5: spec-only route reported with the correct direction ------------
new_fixture
write_main_no_routes
write_spec '/api/v1/ghosts|delete'
run_guard
assert_case \
	"spec-only route -> exit 1 + IN SPEC BUT NOT REGISTERED" 1 \
	'^IN SPEC BUT NOT REGISTERED: DELETE /api/v1/ghosts$'

printf '\n%d/%d checks passed\n' "$PASSED" "$((PASSED + FAILED))"
if ((FAILED > 0)); then
	exit 1
fi
exit 0
