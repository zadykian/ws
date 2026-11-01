#!/bin/bash
# The daemon on the LAN tier.

set -eo pipefail

for _ in $(seq 1 40); do
    status=$(sudo "$TUNNEL" tunnel status 2>&1) || true
    if printf '%s\n' "$status" | grep -qx 'tier: lan' &&
        printf '%s\n' "$status" | grep -Eq '^latest handshake: [0-9]+ s ago$'; then
        printf '%s\n' "$status"
        exit 0
    fi
    sleep 1
done
echo '::error::no handshake on the LAN tier in 40 s; ws tunnel status gives:'
printf '%s\n' "$status"
exit 1
