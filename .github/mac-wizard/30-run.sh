#!/bin/sh
# The first run, with standin.sh's PATH, which lacks brew. The wizard must still find brew,
# so that ws mac apply copies ws for root, makes the Mac's key, and writes ssh's files.
# Without WS_PUBLIC_KEY, ws mac apply prints the Mac's key, which the wizard drops. It must make
# the ssh key that mac.conf names, ~/.ssh/id_ed25519, beside its .pub, with no passphrase, as no
# terminal asked one. It must add the .pub to the stand-in gh's login, titled with the Mac's name.
# It must add the environment's answers at the end of mac.conf, and leave no lock or file of its
# own in $TMPDIR. It must warn that ws, reached through WS_REMOTE, faces the internet, and add ws's
# host key to known_hosts under ws, as WS_HOST_KEY gives its fingerprint. The runner on the stand-in ws
# takes no key yet, so that run must stop at Login, and add none to the user's authorized_keys. The
# case then puts the account's keys there, as Ubuntu's installer imports them for its user, and the
# wizard runs again. It must run ws server
# apply once, with --json alone, in a unit with root's HOME, stamp its status 0, and print no event. Then the Mac's
# key must go to the stand-in ws's mac.pub, and ws server apply --only wireguard run there. wg0's key
# must end mac.conf as WS_PUBLIC_KEY, and ws mac apply, run again, load the daemon, which must have a
# handshake with the stand-in ws. The wizard must log in to ws through the tunnel once. The closing
# list must name no step of the tunnel.

set -u

wsbin=$HOME/.local/bin/ws
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0
# mac.conf's answers that standin.sh leaves out, from the environment, each empty for no value.
export WS_TARGET=lan WS_LAN='' WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS=''

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

host=$(cat "$RUNNER_TEMP/ws/fingerprint")
WS_HOST_KEY=$host PATH=$(cat "$RUNNER_TEMP/path") "$wsbin" mac wizard </dev/null \
    >"$RUNNER_TEMP/stop.out" 2>"$RUNNER_TEMP/stop.err"
status=$?
cat "$RUNNER_TEMP/stop.err" "$RUNNER_TEMP/stop.out"
stop="ws mac wizard: Login: $(id -un) on ws doesn't log in by the Mac's key, $HOME/.ssh/id_ed25519.pub,"
if [ "$status" != 1 ] || ! grep -qF "$stop" "$RUNNER_TEMP/stop.err"; then
    error "ws mac wizard exited $status, or did not stop at Login, where the user takes no key"
fi
for each in /usr/local/libexec/ws-tunnel/ws /etc/ws-tunnel/private.key ~/.ssh/config.d/ws; do
    sudo test -f "$each" || error "ws mac wizard made no $each"
done
[ "$(ssh-keygen -y -P '' -f ~/.ssh/id_ed25519 | cut -d ' ' -f 1-2)" = \
    "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ] ||
    error 'ws mac wizard made no ~/.ssh/id_ed25519 with no passphrase, beside its .pub'
added="ssh-key add $HOME/.ssh/id_ed25519.pub --title $(hostname -s)"
if ! grep -qxF "$added" "$RUNNER_TEMP/gh/log" ||
    ! grep -qxF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" "$RUNNER_TEMP/gh/keys"; then
    error "ws mac wizard added no ~/.ssh/id_ed25519.pub to gh's login, titled with the Mac's name"
fi
grep -q '^ws mac wizard: Host key: warning: through WS_REMOTE, ' "$RUNNER_TEMP/stop.err" ||
    error "ws mac wizard gave no warning that ws faces the internet through WS_REMOTE"
grep -qxF "ws $(cut -d ' ' -f 1-2 "$RUNNER_TEMP/ws/host_key.pub")" ~/.ssh/known_hosts ||
    error "ws mac wizard added no line for ws's host key to ~/.ssh/known_hosts"
if grep -qF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ~/.ssh/authorized_keys \
    2>/dev/null; then
    error "ws mac wizard added ~/.ssh/id_ed25519.pub to the user's authorized_keys"
fi
install -d -m 700 ~/.ssh
install -m 600 "$RUNNER_TEMP/gh/keys" ~/.ssh/authorized_keys
PATH=$(cat "$RUNNER_TEMP/path") "$wsbin" mac wizard >"$RUNNER_TEMP/run.out" 2>"$RUNNER_TEMP/run.err"
status=$?
cat "$RUNNER_TEMP/run.err" "$RUNNER_TEMP/run.out"
[ "$status" = 0 ] || error "ws mac wizard exited $status, with the account's keys in the user's file"
if ! printf '\n%s\n' '--only wireguard' | cmp -s - "$RUNNER_TEMP/ws/server.log" ||
    ! grep -q -- '--setenv=HOME=/root ' "$RUNNER_TEMP/ws/run.log" ||
    ! grep -q ' 0 on -$' "$RUNNER_TEMP/ws/root/var/lib/ws-wizard/run"; then
    error "ws mac wizard ran no ws server apply on ws, once, then with --only wireguard, with root's HOME"
fi
! grep -q '"event"' "$RUNNER_TEMP/run.err" || error "ws mac wizard printed an event line of the run's log"
mac=$(sudo sh -c '"$1" pubkey <"$2"' sh /opt/homebrew/bin/wg /etc/ws-tunnel/private.key)
[ "$(sudo cat "$RUNNER_TEMP/ws/root/etc/wireguard/mac.pub")" = "$mac" ] ||
    error "ws mac wizard gave the stand-in ws no mac.pub of the Mac's key, $mac"
launchctl print system/com.github.zadykian.ws-tunnel >/dev/null 2>&1 ||
    error "ws mac wizard loaded no daemon with wg0's key"
sudo /usr/local/libexec/ws-tunnel/ws mac tunnel status | grep -q '^latest handshake: [0-9]' &&
    [ "$(cat "$RUNNER_TEMP/ws/tunnel.log" 2>/dev/null | wc -l)" -eq 1 ] ||
    error "ws mac wizard had no handshake, or logged in to ws through the tunnel other than once"
! grep -q 'WireGuard public key' "$RUNNER_TEMP/run.out" || error "ws mac wizard printed ws mac apply's key"
printf '%s\n' "WS_USER=$(id -un)" WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY=~/.ssh/id_ed25519.pub' \
    WS_TARGET=lan WS_LAN= WS_GIT_NAME= WS_TUNNEL_NETWORKS= WS_RESOLVERS= \
    "WS_PUBLIC_KEY=$(cat "$RUNNER_TEMP/ws/wg0")" |
    cmp -s - ~/.config/ws/mac.conf ||
    error "ws mac wizard wrote another mac.conf than standin.sh's lines, the answers and wg0's key"
if [ "$(sed -n 2p "$RUNNER_TEMP/run.out")" != 'What stays by hand:' ] ||
    grep -qi 'tunnel' "$RUNNER_TEMP/run.out"; then
    error "ws mac wizard listed nothing by hand, or a step of the tunnel"
fi
for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "ws mac wizard left $left"
done
exit "$fail"
