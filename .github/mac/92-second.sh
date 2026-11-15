#!/bin/bash
# A second server, lab, at slot 1. A third wireguard-go stands in for it on 51821. The Mac's key is
# its peer, as in 35-standin.sh. lab shares ws's WS_REMOTE, so its ports differ from ws's. ws client
# macos apply --server lab must give lab a daemon and a utun of its own, and ssh's host at its slot.
# ws's daemon and host stay as they were. ws client macos servers must list both. A second run for
# lab must print nothing.

set -eo pipefail
# shellcheck source=.github/mac-conf.sh
. .github/mac-conf.sh

conf=$HOME/.config/ws/mac.conf
lab=${LABEL%.ws}.lab
wg=$(brew --prefix wireguard-tools)/bin/wg
go=$(brew --prefix wireguard-go)/bin/wireguard-go

(umask 077 && "$wg" genkey >"$RUNNER_TEMP/lab.key")
"$wg" pubkey <"$RUNNER_TEMP/lab.key" >"$RUNNER_TEMP/lab.pub"
sudo env WG_TUN_NAME_FILE=/var/run/wireguard/lab.name "$go" utun || {
    echo "::error::wireguard-go made no utun for lab's stand-in"
    exit 1
}
utun=$(sudo cat /var/run/wireguard/lab.name)
sudo "$wg" set "$utun" private-key "$RUNNER_TEMP/lab.key" listen-port 51821 \
    peer "$(cat "$RUNNER_TEMP/mac.pub")" allowed-ips 10.99.1.2/32
sudo ifconfig "$utun" up

conf_server=lab
{
    cat "$conf"
    printf '%s\n' WS_USER=lab WS_SLOT=1 WS_REMOTE=localhost WS_REMOTE_PORT=22023 WS_PORT=51821 \
        "WS_PUBLIC_KEY=$(cat "$RUNNER_TEMP/lab.pub")"
} | conf_write "$conf"
conf_server=ws

ws client macos apply --server lab || {
    echo '::error::ws client macos apply --server lab failed'
    exit 1
}
launchctl print "system/$lab" >/dev/null || {
    echo "::error::lab's daemon is not loaded"
    exit 1
}
[ "$(stat -f '%Su:%Sg %Lp' "/Library/LaunchDaemons/$lab.plist")" = "root:wheel 644" ] || {
    echo "::error::lab's plist is not root:wheel 644"
    exit 1
}

# status NAME prints the daemon's status of the server NAME.
status() { sudo "$TUNNEL" client macos tunnel status --server "$1" 2>&1 || true; }
# shaken NAME PORT: the daemon of NAME has a handshake on the remote tier at 127.0.0.1:PORT.
shaken() {
    out=$(status "$1")
    printf '%s\n' "$out" | grep -qx 'tier: remote' &&
        printf '%s\n' "$out" | grep -qx "endpoint: 127.0.0.1:$2" &&
        printf '%s\n' "$out" | grep -Eq '^latest handshake: [0-9]+ s ago$'
}
# shakes NAME PORT gives the daemon of NAME 60 s for it. A stand-in answers no ping
# (35-standin.sh), so a daemon tries its paths again every 36 s or so. Its peer is new for each
# try, and has no handshake for the 5 s that nothing answers at WS_LAN_ADDRESS.
shakes() {
    for _ in $(seq 1 60); do
        shaken "$1" "$2" && return
        sleep 1
    done
    return 1
}
shakes lab 51821 || {
    echo "::error::no handshake on lab's remote tier in 60 s; its status gives:"
    printf '%s\n' "$out"
    exit 1
}
shakes ws 51820 || {
    echo "::error::ws's daemon has no handshake beside lab's in 60 s; its status gives:"
    printf '%s\n' "$out"
    exit 1
}
[ "$(status lab | sed -n 's/^interface: //p')" != "$(status ws | sed -n 's/^interface: //p')" ] || {
    echo '::error::lab and ws share a utun'
    exit 1
}

# tunnel_end NAME prints the HostName that ssh gives NAME.
tunnel_end() { ssh -G "$1" | awk '$1 == "hostname" { print $2 }'; }
[ "$(tunnel_end lab)" = 10.99.1.1 ] && [ "$(tunnel_end ws)" = 10.99.0.1 ] || {
    echo "::error::ssh -G gives lab '$(tunnel_end lab)' and ws '$(tunnel_end ws)', not 10.99.1.1 and 10.99.0.1"
    exit 1
}

# listed NAME prints the row of the server NAME in ws client macos servers, its blanks as one.
listed() { ws client macos servers | awk -v n="$1" '$1 == n { $1 = $1; print }'; }
[ "$(listed ws)" = 'ws lan 0 remote 127.0.0.1:51820' ] && [ "$(listed lab)" = 'lab lan 1 remote 127.0.0.1:51821' ] || {
    echo "::error::ws client macos servers lists '$(listed ws)' and '$(listed lab)'; it gives:"
    ws client macos servers
    exit 1
}

if ! out=$(ws client macos apply --check --server lab 2>&1) || [ -n "$out" ]; then
    echo '::error::ws client macos apply --check --server lab printed what it should not:'
    printf '%s\n' "$out"
    exit 1
fi
