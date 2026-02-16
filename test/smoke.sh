#!/usr/bin/env bash
#
# zburn pre-release smoke test
#
# Builds the binary, then exercises every CLI subcommand end-to-end.
# Uses ZBURN_PASSWORD env var to supply passwords non-interactively.
#
# Usage:
#   ./test/smoke.sh                     # build with default version ("dev")
#   ./test/smoke.sh 0.5.0               # build with version via ldflags
#
# Exit non-zero on any failure.

set -euo pipefail

PASS_COUNT=0
FAIL_COUNT=0
SMOKE_TMPDIR=""
BINARY=""
PASSWORD="smokeTestPass42!"
VERSION="${1:-}"

# --- helpers ----------------------------------------------------------------

cleanup() {
  if [[ -n "$SMOKE_TMPDIR" && -d "$SMOKE_TMPDIR" ]]; then
    rm -rf "$SMOKE_TMPDIR"
  fi
}
trap cleanup EXIT

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf "  \033[32mPASS\033[0m  %s\n" "$1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf "  \033[31mFAIL\033[0m  %s\n" "$1"
  if [[ -n "${2:-}" ]]; then
    printf "        %s\n" "$2"
  fi
}

# --- setup ------------------------------------------------------------------

printf "\n=== zburn smoke test ===\n\n"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SMOKE_TMPDIR=$(mktemp -d)
DATADIR="$SMOKE_TMPDIR/data"
export XDG_DATA_HOME="$DATADIR"

BINARY="$SMOKE_TMPDIR/zburn"

printf "building binary...\n"
if [[ -n "$VERSION" ]]; then
  go build -ldflags "-X main.version=$VERSION" -o "$BINARY" "$REPO_DIR/cmd/zburn"
  EXPECT_VERSION="$VERSION"
else
  go build -o "$BINARY" "$REPO_DIR/cmd/zburn"
  EXPECT_VERSION="dev"
fi
printf "binary built: %s (expect version: %s)\n\n" "$BINARY" "$EXPECT_VERSION"

# --- test 1: version --------------------------------------------------------

printf "test: version\n"
VERSION_OUT=$("$BINARY" version 2>&1)
if echo "$VERSION_OUT" | grep -q "$EXPECT_VERSION"; then
  pass "version output contains '$EXPECT_VERSION'"
else
  fail "version output" "expected '$EXPECT_VERSION', got: $VERSION_OUT"
fi

# --- test 2: email ----------------------------------------------------------

printf "test: email\n"
EMAIL_OUT=$("$BINARY" email 2>&1)
if echo "$EMAIL_OUT" | grep -qE '^[a-z]+[a-z]+[0-9]{4}@zburn\.id$'; then
  pass "email matches pattern *@zburn.id"
else
  fail "email pattern" "got: $EMAIL_OUT"
fi

# --- test 3: identity --json (no save, no password needed) ------------------

printf "test: identity --json\n"
IDENTITY_OUT=$("$BINARY" identity --json 2>&1)

FIELDS_OK=true
for field in id first_name last_name email phone street city state zip dob; do
  if ! echo "$IDENTITY_OUT" | python3 -c "import sys,json; d=json.load(sys.stdin); assert '$field' in d" 2>/dev/null; then
    FIELDS_OK=false
    fail "identity --json missing field '$field'"
  fi
done

if $FIELDS_OK; then
  pass "identity --json contains all expected fields"
fi

# --- test 4: identity --json --save (first run) ----------------------------

printf "test: identity --json --save\n"
SAVE_OUT=$(ZBURN_PASSWORD="$PASSWORD" "$BINARY" identity --json --save 2>/dev/null) || {
  fail "identity --json --save exited non-zero"
  SAVE_OUT=""
}

if [[ -n "$SAVE_OUT" ]]; then
  if [[ -f "$DATADIR/zburn/salt" ]]; then
    pass "store salt file created"
  else
    fail "store salt file not found" "ls: $(ls "$DATADIR/zburn/" 2>&1)"
  fi

  SAVED_ID=$(echo "$SAVE_OUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null) || SAVED_ID=""

  if [[ -n "$SAVED_ID" ]]; then
    pass "extracted saved identity ID: $SAVED_ID"
  else
    fail "could not extract identity ID from save output"
  fi
fi

# --- test 5: list --json ----------------------------------------------------

printf "test: list --json\n"
LIST_OUT=$(ZBURN_PASSWORD="$PASSWORD" "$BINARY" list --json 2>/dev/null) || {
  fail "list --json exited non-zero"
  LIST_OUT=""
}

if [[ -n "$LIST_OUT" ]]; then
  LIST_HAS_ID=$(echo "$LIST_OUT" | python3 -c "import sys,json; print(' '.join(i['id'] for i in json.loads(sys.stdin.read())))" 2>/dev/null) || LIST_HAS_ID=""

  if [[ -n "${SAVED_ID:-}" ]] && echo "$LIST_HAS_ID" | grep -q "$SAVED_ID"; then
    pass "list --json contains saved identity $SAVED_ID"
  else
    fail "list --json does not contain saved identity" "got IDs: ${LIST_HAS_ID:-<empty>}"
  fi
fi

# --- test 6: forget <id> ----------------------------------------------------

if [[ -n "${SAVED_ID:-}" ]]; then
  printf "test: forget %s\n" "$SAVED_ID"
  FORGET_OUT=$(ZBURN_PASSWORD="$PASSWORD" "$BINARY" forget "$SAVED_ID" 2>/dev/null) || {
    fail "forget exited non-zero"
    FORGET_OUT=""
  }

  if echo "$FORGET_OUT" | grep -q "deleted"; then
    pass "forget output contains 'deleted'"
  else
    fail "forget output" "got: ${FORGET_OUT:-<empty>}"
  fi
else
  printf "test: forget (skipped — no saved ID)\n"
  fail "forget skipped due to earlier failure"
fi

# --- test 7: list --json (should be empty after forget) ---------------------

printf "test: list (after forget)\n"
LIST2_OUT=$(ZBURN_PASSWORD="$PASSWORD" "$BINARY" list 2>/dev/null) || {
  fail "list (after forget) exited non-zero"
  LIST2_OUT=""
}

if [[ -n "$LIST2_OUT" ]]; then
  if echo "$LIST2_OUT" | grep -q "no saved identities"; then
    pass "list reports no saved identities after forget"
  else
    fail "list after forget" "got: $LIST2_OUT"
  fi
fi

# --- summary ----------------------------------------------------------------

printf "\n=== results: %d passed, %d failed ===\n\n" "$PASS_COUNT" "$FAIL_COUNT"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
