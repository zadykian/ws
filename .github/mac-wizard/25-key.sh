#!/bin/sh
# The Mac key stage on a fresh Mac, which none of these runs may change. With a mac.conf of its own
# that holds every answer but names no key for ws, and no terminal, a run must stop at the stage,
# naming WS_IDENTITY, before setup.sh's. --check must say that a run at a terminal would ask whether
# Secretive keeps the key. Neither may make a key, change that mac.conf, or leave a lock or file of
# the wizard's in $TMPDIR.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$RUNNER_TEMP/key.conf
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_LAN= WS_GIT_NAME= WS_TUNNEL_NETWORKS= \
    WS_RESOLVERS= >"$conf"
cp "$conf" "$RUNNER_TEMP/key.conf.before"
out=$(WS_CONF=$conf sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -q '^wizard\.sh: Mac key: no terminal .*: set WS_IDENTITY=' &&
    [ "$status" = 1 ] || error "wizard.sh, with no terminal, exited $status, naming no WS_IDENTITY"
! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error 'wizard.sh ran setup.sh after the Mac key'
out=$(WS_CONF=$conf sh "$wizard" --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -q "^wizard\.sh: Mac key: would ask whether Secretive keeps " &&
    [ "$status" = 0 ] || error "wizard.sh --check exited $status, with no question on Secretive"
for left in ~/.ssh/id_ed25519 ~/.ssh/id_ed25519.pub "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
cmp -s "$conf" "$RUNNER_TEMP/key.conf.before" || error "wizard.sh changed $conf"
rm -f "$conf" "$RUNNER_TEMP/key.conf.before"
exit "$fail"
