#!/usr/bin/env bash
#
# zburn pre-release smoke test
#
# Builds the binary, then exercises every CLI subcommand end-to-end.
# Uses a Python pty wrapper to feed passwords through a real tty fd,
# since term.ReadPassword requires an actual terminal.
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

# run_with_password executes a command with password(s) piped through a pty.
# $1 = number of password prompts to answer (1 or 2)
# remaining args = the command to run
# stdout from the child is captured and echoed.
run_with_password() {
  local num_prompts="$1"
  shift

  local output
  output=$(NUM_PROMPTS="$num_prompts" PASSWORD="$PASSWORD" python3 -c "
import pty, os, sys, select, time

def run():
    password = os.environ['PASSWORD']
    prompts_remaining = int(os.environ['NUM_PROMPTS'])

    (child_pid, fd) = pty.fork()
    if child_pid == 0:
        os.execvp(sys.argv[1], sys.argv[1:])
        os._exit(127)

    output = b''
    sent_all = (prompts_remaining <= 0)

    while True:
        try:
            rlist, _, _ = select.select([fd], [], [], 5.0)
        except (ValueError, OSError):
            break
        if not rlist:
            break
        try:
            data = os.read(fd, 4096)
        except OSError:
            break
        if not data:
            break
        output += data

        if not sent_all and b'password:' in data.lower():
            time.sleep(0.05)
            os.write(fd, (password + '\n').encode())
            prompts_remaining -= 1
            if prompts_remaining <= 0:
                sent_all = True

    _, status = os.waitpid(child_pid, 0)
    exit_code = os.waitstatus_to_exitcode(status)

    # strip terminal noise: password prompts and echoed password
    lines = output.decode('utf-8', errors='replace').splitlines()
    clean = []
    for line in lines:
        s = line.strip().replace('\r', '')
        if not s:
            continue
        low = s.lower()
        if 'password:' in low and len(s) < 80:
            continue
        if s == password:
            continue
        clean.append(s)
    sys.stdout.write('\n'.join(clean))
    sys.exit(exit_code)

run()
" "$@" 2>/dev/null) || {
    local rc=$?
    echo "$output"
    return $rc
  }

  echo "$output"
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
for field in id first_name last_name email phone street city state zip dob password; do
  if ! echo "$IDENTITY_OUT" | python3 -c "import sys,json; d=json.load(sys.stdin); assert '$field' in d" 2>/dev/null; then
    FIELDS_OK=false
    fail "identity --json missing field '$field'"
  fi
done

if $FIELDS_OK; then
  pass "identity --json contains all expected fields"
fi

# --- test 4: identity --json --save (first run = 2 password prompts) --------

printf "test: identity --json --save\n"
SAVE_OUT=$(run_with_password 2 "$BINARY" identity --json --save) || {
  fail "identity --json --save exited non-zero"
  SAVE_OUT=""
}

if [[ -n "$SAVE_OUT" ]]; then
  if [[ -f "$DATADIR/zburn/salt" ]]; then
    pass "store salt file created"
  else
    fail "store salt file not found" "ls: $(ls "$DATADIR/zburn/" 2>&1)"
  fi

  # extract identity ID from the JSON in the output
  SAVED_ID=$(echo "$SAVE_OUT" | python3 -c "
import sys, json

text = sys.stdin.read()
# find the JSON object — output may have terminal noise around it
depth = 0
start = -1
for i, ch in enumerate(text):
    if ch == '{':
        if depth == 0:
            start = i
        depth += 1
    elif ch == '}':
        depth -= 1
        if depth == 0 and start >= 0:
            try:
                d = json.loads(text[start:i+1])
                print(d['id'])
                sys.exit(0)
            except (json.JSONDecodeError, KeyError):
                start = -1
sys.exit(1)
" 2>/dev/null) || SAVED_ID=""

  if [[ -n "$SAVED_ID" ]]; then
    pass "extracted saved identity ID: $SAVED_ID"
  else
    fail "could not extract identity ID from save output"
  fi
fi

# --- test 5: list --json (subsequent run = 1 password prompt) ---------------

printf "test: list --json\n"
LIST_OUT=$(run_with_password 1 "$BINARY" list --json) || {
  fail "list --json exited non-zero"
  LIST_OUT=""
}

if [[ -n "$LIST_OUT" ]]; then
  LIST_HAS_ID=$(echo "$LIST_OUT" | python3 -c "
import sys, json

text = sys.stdin.read()
start = text.find('[')
end = text.rfind(']')
if start < 0 or end < 0:
    sys.exit(1)
arr = json.loads(text[start:end+1])
print(' '.join(item['id'] for item in arr))
" 2>/dev/null) || LIST_HAS_ID=""

  if [[ -n "${SAVED_ID:-}" ]] && echo "$LIST_HAS_ID" | grep -q "$SAVED_ID"; then
    pass "list --json contains saved identity $SAVED_ID"
  else
    fail "list --json does not contain saved identity" "got IDs: ${LIST_HAS_ID:-<empty>}"
  fi
fi

# --- test 6: forget <id> ----------------------------------------------------

if [[ -n "${SAVED_ID:-}" ]]; then
  printf "test: forget %s\n" "$SAVED_ID"
  FORGET_OUT=$(run_with_password 1 "$BINARY" forget "$SAVED_ID") || {
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

printf "test: list --json (after forget)\n"
LIST2_OUT=$(run_with_password 1 "$BINARY" list --json) || {
  fail "list --json (after forget) exited non-zero"
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
