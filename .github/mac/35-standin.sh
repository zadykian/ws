#!/bin/bash
# A stand-in ws.
# A second wireguard-go, with its own key, listening on 51820 with the Mac's key, as setup.sh
# printed it, as its peer. It has no address in the tunnel, so the daemon's pings go
# unanswered and the steps below read handshakes instead.

set -eo pipefail

wg=$(brew --prefix wireguard-tools)/bin/wg
go=$(brew --prefix wireguard-go)/bin/wireguard-go
(umask 077 && "$wg" genkey >"$RUNNER_TEMP/ws.key")
"$wg" pubkey <"$RUNNER_TEMP/ws.key" >"$RUNNER_TEMP/ws.pub"
sudo mkdir -p /var/run/wireguard
sudo env WG_TUN_NAME_FILE=/var/run/wireguard/standin.name "$go" utun ||
    {
        echo '::error::wireguard-go made no utun, as the daemon would make none'
        exit 1
    }
utun=$(sudo cat /var/run/wireguard/standin.name)
sudo "$wg" set "$utun" private-key "$RUNNER_TEMP/ws.key" listen-port 51820 \
    peer "$(cat "$RUNNER_TEMP/mac.pub")" allowed-ips 10.99.0.2/32
sudo ifconfig "$utun" up
sudo "$wg" show "$utun"
