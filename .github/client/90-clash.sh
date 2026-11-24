#!/bin/bash
# Two servers' files that clash are refused. With a second and a third server beside ws, each pair
# below clashes in one key: a network, a domain, the cable, a slot or a port. mise bootstrap's plan
# must fail with the refusal of lab's file, which names the way out. A clash fails every unit's
# read, so mise stops before a change. The files go with the case, even on a failure.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

conf=/etc/ws/values.yml
trap 'sudo rm -rf "${conf%/*}/servers/lab" "${conf%/*}/servers/box"' EXIT

# ws's own ports, which the slot and port cases clash with.
for line in 'WS_REMOTE: localhost' 'WS_REMOTE_PORT: 22022' 'WS_TUNNEL_PORT: 51820'; do
    conf_print "$conf" | grep -qx "$line" || {
        echo "::error::ws's file lacks $line, which the cases below clash with"
        exit 1
    }
done

# server NAME LINE...: NAME's file holds the lines, with a slot where they give none.
server() {
    conf_server=$1
    shift
    {
        cat "$conf"
        printf '%s\n' "$@"
    } | conf_write "$conf"
    conf_server=ws
}

# refused WHY NAME: mise bootstrap's plan must fail, with WHY, and the way out of NAME's file.
refused() {
    status=0
    out=$(client_bootstrap plan 2>&1) || status=$?
    if [ "$status" = 0 ] || ! printf '%s\n' "$out" | grep -qF -e "$1"; then
        echo "::error::mise bootstrap's plan exited $status, printing '$out', where a failure and server $2's '$1' were due"
        exit 1
    fi
    printf '%s\n' "$out" | grep -qF "fix its file, or ws macos remove $2" || {
        echo "::error::mise bootstrap's plan named no way out of server $2's file: '$out'"
        exit 1
    }
}

server box 'WS_SLOT: 2' 'WS_TUNNEL_NETWORKS: 198.51.100.0/24'
server lab 'WS_SLOT: 1' 'WS_TUNNEL_NETWORKS: 198.51.100.0/25'
refused "WS_TUNNEL_NETWORKS's 198.51.100.0/25 overlaps server box's 198.51.100.0/24" lab

server box 'WS_SLOT: 2' 'WS_RESOLVERS: example.test=198.51.100.53'
server lab 'WS_SLOT: 1' 'WS_RESOLVERS: example.test=198.51.100.54'
refused "WS_RESOLVERS's example.test is server box's too" lab

server box 'WS_SLOT: 2' 'WS_CABLE_SERVICE: Box'
server lab 'WS_SLOT: 1' 'WS_CABLE_SERVICE: Lab'
refused "WS_CABLE_SERVICE names a cable, as server box's does" lab

sudo rm -rf "${conf%/*}/servers/box"
server lab 'WS_SLOT: 0'
refused "WS_SLOT=0 is server ws's slot too" lab

server lab 'WS_SLOT: 1' 'WS_REMOTE: localhost' 'WS_REMOTE_PORT: 22023' 'WS_TUNNEL_PORT: 51820'
refused "WS_TUNNEL_PORT=51820 is server ws's too, behind WS_REMOTE localhost" lab
server lab 'WS_SLOT: 1' 'WS_REMOTE: localhost' 'WS_REMOTE_PORT: 22022' 'WS_TUNNEL_PORT: 51821'
refused "WS_REMOTE_PORT=22022 is server ws's too, behind WS_REMOTE localhost" lab
