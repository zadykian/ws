#!/bin/sh
# The first run, by sh, with standin.sh's PATH, which lacks brew. The wizard must still find brew,
# so that setup.sh installs wireguard-tools, root's copies and the Mac's key, and writes ssh's
# files. Without WS_PUBLIC_KEY, setup.sh loads no daemon, and prints the Mac's key, which the
# wizard drops: it must end with the tunnel's step by hand instead. It must leave no lock or file
# of its own in $TMPDIR.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0
# mac.conf's answers that standin.sh leaves out, from the environment, each empty for no value.
export WS_LAN='' WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS=''

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

PATH=$(cat "$RUNNER_TEMP/path") sh "$wizard" >"$RUNNER_TEMP/run.out" 2>"$RUNNER_TEMP/run.err"
status=$?
cat "$RUNNER_TEMP/run.err" "$RUNNER_TEMP/run.out"
[ "$status" = 0 ] || error "wizard.sh exited $status"
brew list --formula wireguard-tools >/dev/null 2>&1 ||
    error 'wizard.sh installed no wireguard-tools'
for each in /usr/local/libexec/ws-tunnel/wg /etc/ws-tunnel/private.key ~/.ssh/config.d/ws; do
    sudo test -f "$each" || error "wizard.sh made no $each"
done
! launchctl print system/com.github.zadykian.ws-tunnel >/dev/null 2>&1 ||
    error "wizard.sh loaded the daemon without ws's key"
! grep -q 'WireGuard public key' "$RUNNER_TEMP/run.out" || error "wizard.sh printed setup.sh's key"
if [ "$(sed -n 2p "$RUNNER_TEMP/run.out")" != 'What stays by hand:' ] ||
    ! grep -q "^- The Mac's tunnel: run mac/setup.sh, " "$RUNNER_TEMP/run.out"; then
    error "wizard.sh listed no tunnel's step by hand"
fi
for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
exit "$fail"
