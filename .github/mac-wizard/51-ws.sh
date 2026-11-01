#!/bin/sh
# The ws that bootstrap.sh built into ~/.local/bin, in 50-bootstrap.sh's run: it must run, print
# its usage, and refuse an unknown command with status 2.

set -u

ws=$HOME/.local/bin/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

out=$("$ws" 2>&1) || error "$ws exited $?"
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -q '^usage: ws' || error "$ws printed no usage"
"$ws" nonesuch >/dev/null 2>&1
status=$?
[ "$status" = 2 ] || error "$ws nonesuch exited $status, not 2"
file "$ws"
exit "$fail"
