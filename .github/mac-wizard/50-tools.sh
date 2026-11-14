#!/bin/sh
# The wizard on the Mac that 40-again.sh leaves, with standin.sh's PATH, which lacks brew. Its Tools
# and Homebrew stages must find the runner's Command Line Tools and brew, so that a run and
# --check print nothing at all. Where WS_CONF names no file, --check must show the lines the wizard
# would write there, from the environment's answers, and write none.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws client macos wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
path=$(cat "$RUNNER_TEMP/path")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

for args in '' --check; do
    out=$(PATH=$path "$wsbin" client macos wizard ${args:+"$args"} 2>&1)
    status=$?
    printf '%s\n' "$out"
    [ "$status" = 0 ] || error "ws client macos wizard${args:+ $args} exited $status"
    [ -z "${out#"$known"}" ] || error "ws client macos wizard${args:+ $args} printed what it should not"
done

conf=$RUNNER_TEMP/confs/tools/mac.conf
user=$(id -un)
out=$(WS_CONF=$conf WS_USER=$user WS_TARGET=lan WS_LAN='' WS_REMOTE=localhost WS_REMOTE_PORT=22022 \
    WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS='' PATH=$path \
    "$wsbin" client macos wizard --server ws --check 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qx '+WS_REMOTE=localhost' && [ "$status" = 0 ] ||
    error "ws client macos wizard --check exited $status, with no line the wizard would write to $conf"
[ ! -e "$conf" ] && [ ! -e "$(conf_server_file "$conf")" ] || error "ws client macos wizard --check wrote $conf or the server's file"
rm -rf "${conf%/*}"
exit "$fail"
