#!/bin/sh
# The Mac key stage on a fresh Mac, which none of these runs may change. With a mac.conf of its own
# that holds every answer but names no key for ws, the key is the YubiKey's, whose .pub setup.sh
# writes after the stage. So a run with no terminal, and --check, must stop at the stage, naming
# ws's agent and the .pub, before setup.sh's.
# None may make a key, change that mac.conf, or leave a lock or file of the wizard's in $TMPDIR.

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
want="wizard.sh: Mac key: the agent at $HOME/.ssh/ws-agent.sock lists no key of $HOME/.config/ws"
printf '%s\n' "$out" | grep -qF "$want/keys/ws.pub" &&
    [ "$status" = 1 ] || error "wizard.sh, with no terminal, exited $status, naming no ws's agent"
! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error 'wizard.sh ran setup.sh after the Mac key'
out=$(WS_CONF=$conf sh "$wizard" --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qF "$want/keys/ws.pub" &&
    [ "$status" = 1 ] || error "wizard.sh --check exited $status, naming no ws's agent"
for left in ~/.ssh/id_ed25519 ~/.ssh/id_ed25519.pub "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
cmp -s "$conf" "$RUNNER_TEMP/key.conf.before" || error "wizard.sh changed $conf"
rm -f "$conf" "$RUNNER_TEMP/key.conf.before"
exit "$fail"
