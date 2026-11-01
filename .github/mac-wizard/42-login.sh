#!/bin/sh
# claude's login on the stand-in ws, on the Mac that 41-remote.sh leaves. standin.sh logged it in,
# so no run of the wizard may have opened a session on ws for a login. Without it, a run with no
# terminal must stop at claude on ws, naming claude auth login, and --check must say it would run
# it. The next run must print nothing.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STAGE STATUS WANT ARG...: ws wizard ARGs, with no terminal, must exit STATUS, with WANT in
# its lines of STAGE.
expect() {
    stage=$1
    want_status=$2
    want=$3
    shift 3
    out=$("$wsbin" wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep "^ws wizard: $stage: " | grep -qF -e "$want" &&
        [ "$status" = "$want_status" ] || error "ws wizard $* exited $status, with no '$want'"
}

# sessions: no session on ws may have run a login.
sessions() {
    ! grep -q '^root: exec bash -lc "claude auth login"$' "$ws/log" ||
        error "a run opened a session on ws for a login"
}

sessions
rm "$ws/claude"
expect 'claude on ws' 1 'no terminal is there for claude auth login'
expect 'claude on ws' 0 'would run claude auth login as root on ws' --check
touch "$ws/claude"
sessions
out=$("$wsbin" wizard </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "ws wizard after it exited $status: '$out'"
exit "$fail"
