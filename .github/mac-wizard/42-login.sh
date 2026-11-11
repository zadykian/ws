#!/bin/sh
# claude's login on the stand-in ws, on the Mac that 41-remote.sh leaves. standin.sh logged it in,
# so no run of the wizard may have opened a session on ws for a login. Without it, a run with no
# terminal must stop at claude on ws, naming claude auth login, and --check must say it would run
# it. The next run must print nothing.

set -u

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws mac wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STAGE STATUS WANT ARG...: ws mac wizard ARGs, with no terminal, must exit STATUS, with WANT in
# its lines of STAGE.
expect() {
    stage=$1
    want_status=$2
    want=$3
    shift 3
    out=$("$wsbin" mac wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep "^ws mac wizard: $stage: " | grep -qF -e "$want" &&
        [ "$status" = "$want_status" ] || error "ws mac wizard $* exited $status, with no '$want'"
}

# sessions: no session on ws may have run a login.
sessions() {
    ! grep -q "^$(id -un): exec bash -lc \"claude auth login\"\$" "$ws/log" ||
        error "a run opened a session on ws for a login"
}

sessions
rm "$ws/claude"
expect 'claude on ws' 1 'no terminal is there for claude auth login'
expect 'claude on ws' 0 'would run claude auth login as WS_USER on ws' --check
touch "$ws/claude"
sessions
out=$("$wsbin" mac wizard </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "ws mac wizard after it exited $status: '$out'"
exit "$fail"
