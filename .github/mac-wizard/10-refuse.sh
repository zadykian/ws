#!/bin/sh
# The wizard's refusals, which leave the runner as they found it: an unknown argument, and root.
# -h prints the usage.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# refused WHY COMMAND...: COMMAND must exit 1, with WHY in what it prints.
refused() {
    why=$1
    shift
    out=$("$@" 2>&1)
    status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF "$why"; then
        error "$* exited $status, printing '$out', where 1 and '$why' were due"
    fi
}

[ "$(sh "$wizard" -h)" = 'usage: wizard.sh [--check] [--update]' ] || error 'no usage from -h'
refused "unknown argument '--bogus'" sh "$wizard" --bogus
refused 'run it as yourself, not root' sudo sh "$wizard" --check
exit "$fail"
