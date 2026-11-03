#!/bin/sh
# The wizard on the Mac that 40-again.sh leaves, with standin.sh's PATH, which lacks brew. Its Tools
# and Homebrew stages must find the runner's Command Line Tools and brew, so that a run and
# --check print nothing at all. Where WS_CONF names no file, --check must show the lines the wizard
# would write there, from the environment's answers, and write none.

set -u

wsbin=$HOME/.local/bin/ws
path=$(cat "$RUNNER_TEMP/path")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

for args in '' --check; do
    out=$(PATH=$path "$wsbin" mac wizard ${args:+"$args"} 2>&1)
    status=$?
    printf '%s\n' "$out"
    [ "$status" = 0 ] || error "ws mac wizard${args:+ $args} exited $status"
    [ -z "$out" ] || error "ws mac wizard${args:+ $args} printed what it should not"
done

conf=$RUNNER_TEMP/tools.conf
out=$(WS_CONF=$conf WS_TARGET=lan WS_LAN='' WS_REMOTE=localhost WS_REMOTE_PORT=22022 \
    WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS='' PATH=$path \
    "$wsbin" mac wizard --check 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qx '+WS_REMOTE=localhost' && [ "$status" = 0 ] ||
    error "ws mac wizard --check exited $status, with no line the wizard would write to $conf"
[ ! -e "$conf" ] || error "ws mac wizard --check wrote $conf"
rm -f "$conf"
exit "$fail"
