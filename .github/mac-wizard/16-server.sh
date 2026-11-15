#!/bin/sh
# The wizard's server: --server names it. A bad name must stop a run. On a values.yml with no server, a
# run with no terminal and no name must stop, and ask for the flag. There --check must show the new
# server's file, headed by WS_SLOT: 0, and write none. None of these runs may change a file.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
conf=/etc/ws/values.yml
fail=0

# error MESSAGE... prints an ::error::, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# refused WHY COMMAND...: COMMAND must exit 1, with WHY in what it prints.
refused() {
    why=$1
    shift
    out=$("$@" 2>&1 </dev/null)
    status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$why"; then
        error "$* exited $status, printing '$out', where 1 and '$why' were due"
    fi
}

conf_print "$conf" >"$RUNNER_TEMP/values.yml"
refused "--server Lab is not a server's name" "$wsbin" client macos wizard --server Lab --check
conf_print "$conf" | cmp -s - "$RUNNER_TEMP/values.yml" || error 'a refused run changed values.yml'
[ ! -e "${conf%/*}/servers/lab" ] || error 'a refused run made the file of the server lab'

# A values.yml of the Mac's keys alone, so with no server: the wizard asks the name, and the answers
# of a cloud VM make box's file.
box=$RUNNER_TEMP/confs/server/values.yml
printf '%s\n' 'WS_IDENTITY_PUB_KEY: ~/.ssh/id_ed25519.pub' \
    'WS_GIT_SIGNING_PUB_KEY: ~/.ssh/id_ed25519.pub' | conf_write "$box"
refused "no terminal to ask the server's name: pass --server NAME" \
    env WS_VALUES="$box" "$wsbin" client macos wizard --check
env WS_VALUES="$box" WS_TARGET=cloud WS_REMOTE=ws.example.test WS_REMOTE_PORT=22 \
    WS_TUNNEL_NETWORKS= WS_RESOLVERS= "$wsbin" client macos wizard --server box --check </dev/null >"$RUNNER_TEMP/box" 2>&1
status=$?
cat "$RUNNER_TEMP/box"
[ "$status" = 0 ] || error "ws client macos wizard --server box --check exited $status"
tr -d '\r' <"$RUNNER_TEMP/box" |
    awk -v f="+++ ${box%/*}/servers/box/values.yml" '$0 == f { on = 1; next } on && /^\+/ { print; exit }' |
    grep -qx '+WS_SLOT: 0' || error "ws client macos wizard --server box --check did not show box's file headed by WS_SLOT: 0"
[ ! -e "${box%/*}/servers" ] || error 'ws client macos wizard --server box --check made the servers directory'
rm -r "${box%/*}" "$RUNNER_TEMP/box"
exit "$fail"
