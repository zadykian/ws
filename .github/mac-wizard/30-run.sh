#!/bin/sh
# The first run, by sh, with standin.sh's PATH, which lacks brew. The wizard must still find brew,
# so that setup.sh installs wireguard-tools, root's copies and the Mac's key, and writes ssh's
# files. Without WS_PUBLIC_KEY, setup.sh prints the Mac's key, which the wizard drops. It must make
# the ssh key that mac.conf names, ~/.ssh/id_ed25519, beside its .pub, with no passphrase, as no
# terminal asked one. It must add the .pub to the stand-in gh's login, titled with the Mac's name.
# It must add the environment's answers at the end of mac.conf, and leave no lock or file of its
# own in $TMPDIR. It must warn that ws, reached through WS_REMOTE, faces the internet, and add ws's
# host key to known_hosts under ws, as WS_HOST_KEY gives its fingerprint. Root on the stand-in ws
# takes no key yet, so that run must stop at Root's key, and add none to root's authorized_keys. The
# case then puts the account's keys there, as Ubuntu's installer imports them for its user and the
# README's step copies them to root, and the wizard runs again. It must run ws's playbook once,
# bootstrap.sh without arguments in a unit with root's HOME, and stamp its status 0. Then the Mac's
# key must go to the stand-in ws's mac.pub, and bootstrap.sh --tags wireguard run there. wg0's key
# must end mac.conf as WS_PUBLIC_KEY, and setup.sh, run again, load the daemon, which must have a
# handshake with the stand-in ws. The wizard must log in to ws through the tunnel once. The closing
# list must name no step of the tunnel.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0
# mac.conf's answers that standin.sh leaves out, from the environment, each empty for no value.
export WS_TARGET=lan WS_LAN='' WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS='' WS_HOOK=''

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

host=$(cat "$RUNNER_TEMP/ws/fingerprint")
WS_HOST_KEY=$host PATH=$(cat "$RUNNER_TEMP/path") sh "$wizard" </dev/null \
    >"$RUNNER_TEMP/stop.out" 2>"$RUNNER_TEMP/stop.err"
status=$?
cat "$RUNNER_TEMP/stop.err" "$RUNNER_TEMP/stop.out"
stop="wizard.sh: Root's key: root on ws doesn't log in by the Mac's key, $HOME/.ssh/id_ed25519.pub,"
if [ "$status" != 1 ] || ! grep -qF "$stop" "$RUNNER_TEMP/stop.err"; then
    error "wizard.sh exited $status, or did not stop at Root's key, where root takes no key"
fi
brew list --formula wireguard-tools >/dev/null 2>&1 ||
    error 'wizard.sh installed no wireguard-tools'
for each in /usr/local/libexec/ws-tunnel/wg /etc/ws-tunnel/private.key ~/.ssh/config.d/ws; do
    sudo test -f "$each" || error "wizard.sh made no $each"
done
[ "$(ssh-keygen -y -P '' -f ~/.ssh/id_ed25519 | cut -d ' ' -f 1-2)" = \
    "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ] ||
    error 'wizard.sh made no ~/.ssh/id_ed25519 with no passphrase, beside its .pub'
added="ssh-key add $HOME/.ssh/id_ed25519.pub --title $(hostname -s)"
if ! grep -qxF "$added" "$RUNNER_TEMP/gh/log" ||
    ! grep -qxF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" "$RUNNER_TEMP/gh/keys"; then
    error "wizard.sh added no ~/.ssh/id_ed25519.pub to gh's login, titled with the Mac's name"
fi
grep -q "^wizard\.sh: Host key: warning: through the router's forward, " "$RUNNER_TEMP/stop.err" ||
    error "wizard.sh gave no warning that ws faces the internet through WS_REMOTE"
grep -qxF "ws $(cut -d ' ' -f 1-2 "$RUNNER_TEMP/ws/host_key.pub")" ~/.ssh/known_hosts ||
    error "wizard.sh added no line for ws's host key to ~/.ssh/known_hosts"
if sudo grep -qF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ~root/.ssh/authorized_keys \
    2>/dev/null; then
    error "wizard.sh added ~/.ssh/id_ed25519.pub to root's authorized_keys"
fi
sudo install -d -m 700 ~root/.ssh
sudo install -m 600 "$RUNNER_TEMP/gh/keys" ~root/.ssh/authorized_keys
PATH=$(cat "$RUNNER_TEMP/path") sh "$wizard" >"$RUNNER_TEMP/run.out" 2>"$RUNNER_TEMP/run.err"
status=$?
cat "$RUNNER_TEMP/run.err" "$RUNNER_TEMP/run.out"
[ "$status" = 0 ] || error "wizard.sh exited $status, with the account's keys in root's file"
if ! printf '\n%s\n' '--tags wireguard' | cmp -s - "$RUNNER_TEMP/ws/bootstrap.log" ||
    ! grep -q -- '--setenv=HOME=/root ' "$RUNNER_TEMP/ws/run.log" ||
    ! grep -q ' 0 on$' "$RUNNER_TEMP/ws/root/var/lib/ws-wizard/run"; then
    error "wizard.sh ran no bootstrap.sh on ws, once, then --tags wireguard, with root's HOME"
fi
mac=$(sudo sh -c '"$1" pubkey <"$2"' sh /usr/local/libexec/ws-tunnel/wg /etc/ws-tunnel/private.key)
[ "$(sudo cat "$RUNNER_TEMP/ws/root/etc/wireguard/mac.pub")" = "$mac" ] ||
    error "wizard.sh gave the stand-in ws no mac.pub of the Mac's key, $mac"
launchctl print system/com.github.zadykian.ws-tunnel >/dev/null 2>&1 ||
    error "wizard.sh loaded no daemon with wg0's key"
sudo /usr/local/libexec/ws-tunnel/ws-tunnel status | grep -q '^latest handshake: [0-9]' &&
    [ "$(cat "$RUNNER_TEMP/ws/tunnel.log" 2>/dev/null | wc -l)" -eq 1 ] ||
    error "wizard.sh had no handshake, or logged in to ws through the tunnel other than once"
! grep -q 'WireGuard public key' "$RUNNER_TEMP/run.out" || error "wizard.sh printed setup.sh's key"
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY=~/.ssh/id_ed25519.pub' \
    WS_TARGET=lan WS_LAN= WS_GIT_NAME= WS_TUNNEL_NETWORKS= WS_RESOLVERS= WS_HOOK= \
    "WS_PUBLIC_KEY=$(cat "$RUNNER_TEMP/ws/wg0")" |
    cmp -s - ~/.config/ws/mac.conf ||
    error "wizard.sh wrote another mac.conf than standin.sh's lines, the answers and wg0's key"
if [ "$(sed -n 2p "$RUNNER_TEMP/run.out")" != 'What stays by hand:' ] ||
    grep -qi 'tunnel' "$RUNNER_TEMP/run.out"; then
    error "wizard.sh listed nothing by hand, or a step of the tunnel"
fi
for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
exit "$fail"
