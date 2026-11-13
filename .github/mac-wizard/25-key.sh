#!/bin/sh
# The Mac key stage on a fresh Mac. The case has a mac.conf of its own that holds every answer but
# names no key for ws. With it and no terminal, a run must stop at the stage, naming
# WS_IDENTITY_PUB_KEY, before ws client macos apply's. A --check must say that a run at a terminal
# would ask for ws's login key, with a new key the default, as the runner's agent lists none. With a
# copy that names a missing key, a run must stop at the stage, naming it, as the wizard makes no key
# in a file. With one that names standin.sh's key, --check must say nothing of the stage. None of
# these may change that mac.conf, or leave a lock or file of the wizard's in $TMPDIR.

set -u

wsbin=$HOME/.local/bin/ws
conf=$RUNNER_TEMP/key.conf
named=$RUNNER_TEMP/named.conf
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0
applied='^ws client macos wizard: (ws client macos apply: would apply|wrote [^ ]*/\.ssh/config\.d/ws|ran launchctl |made the Mac.s WireGuard key)'

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

printf '%s\n' "WS_USER=$(id -un)" WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_TARGET=lan WS_LAN= WS_GIT_NAME= \
    WS_TUNNEL_NETWORKS= WS_RESOLVERS= >"$conf"
cp "$conf" "$RUNNER_TEMP/key.conf.before"
out=$(WS_CONF=$conf "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
none="no terminal to ask for ws's login key, and Secretive's agent lists no P-256 key: "
printf '%s\n' "$out" | grep -q "^ws client macos wizard: Mac key: $none.*WS_IDENTITY_PUB_KEY" &&
    [ "$status" = 1 ] || error "ws client macos wizard, with no terminal, exited $status, naming no WS_IDENTITY_PUB_KEY"
! printf '%s\n' "$out" | grep -qE "$applied" || error 'ws client macos wizard ran ws client macos apply after the Mac key'
out=$(WS_CONF=$conf "$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
want="ws client macos wizard: Mac key: would ask for ws's login key, in Secretive: make a new one (the default)"
printf '%s\n' "$out" | grep -qxF "$want" && [ "$status" = 0 ] ||
    error "ws client macos wizard --check exited $status, saying no '$want'"
{ cat "$conf" && echo "WS_IDENTITY_PUB_KEY=$RUNNER_TEMP/none.pub"; } >"$named"
out=$(WS_CONF=$named "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
stop="ws client macos wizard: Mac key: $RUNNER_TEMP/none.pub, named in mac.conf's WS_IDENTITY_PUB_KEY, is missing"
printf '%s\n' "$out" | grep -qF "$stop" && [ "$status" = 1 ] ||
    error "ws client macos wizard exited $status, naming no missing key"
! printf '%s\n' "$out" | grep -qE "$applied" || error 'ws client macos wizard ran ws client macos apply, with no key'
{ cat "$conf" && echo 'WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub'; } >"$named"
out=$(WS_CONF=$named "$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
! printf '%s\n' "$out" | grep -q '^ws client macos wizard: Mac key: ' && [ "$status" = 0 ] ||
    error "ws client macos wizard --check exited $status, or printed a line of the Mac key stage, with the file key"
rm -f "$named" "$RUNNER_TEMP/none.pub"
for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "ws client macos wizard left $left"
done
cmp -s "$conf" "$RUNNER_TEMP/key.conf.before" || error "ws client macos wizard changed $conf"
rm -f "$conf" "$RUNNER_TEMP/key.conf.before"
exit "$fail"
