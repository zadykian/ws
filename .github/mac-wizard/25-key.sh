#!/bin/sh
# The Mac key stage on a fresh Mac, which none of these runs may change. With a mac.conf of its own
# that holds every answer but names no key for ws, and no terminal, a run must stop at the stage,
# naming WS_IDENTITY, before setup.sh's. --check must say that a run at a terminal would ask whether
# Secretive keeps the key, and, with a stand-in for Secretive's app, that it would open the app.
# None of these may make a key, change that mac.conf, or leave a lock or file of the wizard's in
# $TMPDIR. A last run makes a file key in $RUNNER_TEMP, where ssh-add can't take it: it must stop at
# the stage, naming the command to run by hand, as Root's key logs in with no prompt, and leave the
# key it made, before setup.sh.

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

printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_TARGET=lan WS_LAN= WS_GIT_NAME= \
    WS_TUNNEL_NETWORKS= WS_RESOLVERS= WS_HOOK= >"$conf"
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
mkdir -p ~/Applications/Secretive.app
out=$(WS_CONF=$conf sh "$wizard" --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -q '^wizard\.sh: Mac key: would open Secretive, and wait for ' &&
    [ "$status" = 0 ] || error "wizard.sh --check exited $status, with no Secretive's app to open"
rmdir ~/Applications/Secretive.app
# ssh-add fails where SSH_AUTH_SOCK names no socket. WS_IDENTITY names the key, so that no terminal
# is asked, and keeps it from ~/.ssh.
made=$RUNNER_TEMP/key-add
mkdir "$made"
{
    cat "$conf"
    echo "WS_IDENTITY=$made/id_ed25519.pub"
} >"$made/mac.conf"
out=$(SSH_AUTH_SOCK=$made/none WS_CONF=$made/mac.conf sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
stop="wizard.sh: Mac key: $made/id_ed25519 is made, but ssh-add cannot give it to the agent: run"
stop="$stop ssh-add --apple-use-keychain $made/id_ed25519, which asks its passphrase, then run the"
stop="$stop wizard again"
printf '%s\n' "$out" | grep -qF "$stop" && [ "$status" = 1 ] ||
    error "wizard.sh, where ssh-add fails, exited $status, naming no ssh-add to run"
[ -f "$made/id_ed25519" ] && [ -f "$made/id_ed25519.pub" ] ||
    error "wizard.sh, where ssh-add fails, left no key in $made"
! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error 'wizard.sh ran setup.sh, ssh-add failed'
rm -rf "$made"
for left in ~/.ssh/id_ed25519 ~/.ssh/id_ed25519.pub "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
cmp -s "$conf" "$RUNNER_TEMP/key.conf.before" || error "wizard.sh changed $conf"
rm -f "$conf" "$RUNNER_TEMP/key.conf.before"
exit "$fail"
