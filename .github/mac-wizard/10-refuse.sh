#!/bin/sh
# The wizard's refusals, which leave the runner as they found it: an unknown argument, root, and a
# run while another holds the lock, which it must leave. So must the removed key WS_HOOK in mac.conf,
# an unknown one, and a removed op of the files script, run as root as on ws. -h prints the usage.

set -u

wsbin=$HOME/.local/bin/ws
lock=${TMPDIR:-/tmp}
lock=${lock%/}/ws-wizard.lock
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

[ "$("$wsbin" mac wizard -h)" = 'usage: ws mac wizard [--check] [--update]' ] || error 'no usage from -h'
refused "unknown argument '--bogus'" "$wsbin" mac wizard --bogus
refused 'run it as yourself, not root' sudo "$wsbin" mac wizard --check
mkdir "$lock"
refused "another run holds $lock" "$wsbin" mac wizard --check
[ -d "$lock" ] || error 'a refused run removed the lock it found'
rmdir "$lock"
echo WS_HOOK=example/Ws >"$RUNNER_TEMP/refuse.conf"
refused 'line 1: unknown key WS_HOOK' \
    env WS_CONF="$RUNNER_TEMP/refuse.conf" "$wsbin" mac wizard --check
rm "$RUNNER_TEMP/refuse.conf"
refused "unknown op 'stage'" \
    sudo sh "$HOME/repository/ws/mac/wizard-remote-files.sh" stage
exit "$fail"
