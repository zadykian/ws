#!/bin/sh
# A second server, lab, on the Mac where ws has its tunnel. Its stand-in is standin.sh's second, at
# 127.0.0.1:22023. With no terminal a key with a default needs its variable. --check must show
# WS_SLOT=1 and WS_PORT=51821, and write no file. The wizard asks for no WS_USER, so lab's file holds
# that line before the run. The run must set lab's host_vars to 10.99.1.1/24 and 51821. It must give
# lab a daemon with a handshake and one login, and leave ws's files. The next run prints nothing.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws client macos wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
tunnel=/usr/local/libexec/ws-tunnel/ws
conf=$HOME/.config/ws/mac.conf
lab=$RUNNER_TEMP/lab
labconf=${conf%/*}/servers/lab/server.conf
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard ARG...: a run of the wizard for lab with no terminal, and the answers in the words of
# $answers. It prints the output, and sets out and status.
wizard() {
    # shellcheck disable=SC2086 # answers is a list of KEY=value words
    out=$(env $answers "$wsbin" client macos wizard --server lab "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# handshake NAME: the daemon of the server NAME has a handshake.
handshake() {
    sudo "$tunnel" client macos tunnel status --server "$1" 2>/dev/null | grep -q '^latest handshake: [0-9]'
}

# The account's keys go to the user's file on both stand-ins, as ws's installer imports them.
install -d -m 700 ~/.ssh
install -m 600 "$RUNNER_TEMP/gh/keys" ~/.ssh/authorized_keys
conf_print "$conf" >"$RUNNER_TEMP/ws.before"
answers="WS_REMOTE=localhost WS_REMOTE_PORT=22023 WS_TARGET=lan WS_LAN= WS_TUNNEL_NETWORKS= WS_RESOLVERS= WS_CABLE_SERVICE= WS_HOST_KEY=$(cat "$lab/fingerprint")"

wizard --check
shown=$(printf '%s\n' "$out" | tr -d '\r')
[ "$status" = 0 ] && printf '%s\n' "$shown" | grep -qx '+WS_SLOT=1' &&
    printf '%s\n' "$shown" | grep -qx '+WS_PORT=51821' &&
    printf '%s\n' "$shown" | grep -qx '+WS_REMOTE_PORT=22023' && [ ! -e "${labconf%/*}" ] ||
    error "--check for lab exited $status, showed no WS_SLOT=1, WS_PORT=51821 and WS_REMOTE_PORT=22023, or wrote lab's file"

mkdir -p "${labconf%/*}" && echo "WS_USER=$(id -un)" >"$labconf"
wizard
[ "$status" = 0 ] || error "ws client macos wizard --server lab exited $status"

for line in WS_SLOT=1 WS_PORT=51821 WS_REMOTE_PORT=22023 "WS_PUBLIC_KEY=$(cat "$lab/wg0" 2>/dev/null)"; do
    grep -qxF -e "$line" "$labconf" || error "lab's file lacks $line"
done
vars="$lab/root"/etc/ws/host_vars.yml
sudo grep -Eqx 'wireguard_address: "?10\.99\.1\.1/24"?' "$vars" ||
    error "lab's host_vars names no wireguard_address 10.99.1.1/24"
sudo grep -qx 'wireguard_port: 51821' "$vars" || error "lab's host_vars names no wireguard_port 51821"
mac=$(sudo sh -c '"$1" pubkey <"$2"' sh /opt/homebrew/bin/wg /etc/ws-tunnel/private.key)
[ "$(sudo cat "$lab/root"/etc/wireguard/mac.pub)" = "$mac" ] ||
    error "lab's stand-in has no mac.pub of the Mac's key"
grep -qx -- '--only wireguard' "$lab/server.log" || error "lab's stand-in ran no ws server ubuntu apply --only wireguard"
launchctl print system/com.github.zadykian.ws-tunnel.lab >/dev/null 2>&1 || error "the run loaded no daemon for lab"
handshake lab || error "lab's daemon has no handshake with its stand-in"
[ "$(wc -l <"$lab/tunnel.log")" -eq 1 ] || error "the wizard logged in to lab through the tunnel other than once"
handshake ws || error "ws's daemon lost its handshake beside lab's"
grep -qx '    HostName 10.99.1.1' ~/.ssh/config.d/ws || error "the run wrote no Host lab in config.d/ws for the tunnel"
grep -q '^lab ' ~/.ssh/known_hosts || error "the run added no line for lab's host key to known_hosts"
conf_print "$conf" | cmp -s - "$RUNNER_TEMP/ws.before" || error "the run for lab changed mac.conf or ws's file"

answers=
wizard
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "the next run for lab exited $status, or printed what it should not"
exit "$fail"
