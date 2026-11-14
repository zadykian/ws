#!/bin/bash
# Two servers' files that clash are refused. With a second and a third server beside ws, each pair
# below clashes in one key: a network, a domain, the cable, a slot or a port. ws client macos apply
# --check --server lab must stop with the refusal and exit 1. Beside a file ws refuses, a run for
# ws stops too. The files go with the case, even on a failure.

set -eo pipefail
# shellcheck source=.github/mac-conf.sh
. .github/mac-conf.sh

conf=$HOME/.config/ws/mac.conf
trap 'rm -rf "${conf%/*}/servers/lab" "${conf%/*}/servers/box"' EXIT

# ws's own ports, which the slot and port cases clash with.
for line in WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_PORT=51820; do
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

# refused WHY NAME: ws client macos apply --check --server NAME must exit 1, with WHY in what it prints.
refused() {
    status=0
    out=$(ws client macos apply --check --server "$2" 2>&1) || status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$1"; then
        echo "::error::ws client macos apply --check --server $2 exited $status, printing '$out', where 1 and '$1' were due"
        exit 1
    fi
}

server box WS_SLOT=2 WS_TUNNEL_NETWORKS=198.51.100.0/24
server lab WS_SLOT=1 WS_TUNNEL_NETWORKS=198.51.100.0/25
refused "WS_TUNNEL_NETWORKS's 198.51.100.0/25 overlaps server box's 198.51.100.0/24" lab
refused 'fix its file, or ws client macos remove' ws

server box WS_SLOT=2 WS_RESOLVERS=example.test=198.51.100.53
server lab WS_SLOT=1 WS_RESOLVERS=example.test=198.51.100.54
refused "WS_RESOLVERS's example.test is server box's too" lab

server box WS_SLOT=2 WS_CABLE_SERVICE=Box
server lab WS_SLOT=1 WS_CABLE_SERVICE=Lab
refused "WS_CABLE_SERVICE names a cable, as server box's does" lab

rm -rf "${conf%/*}/servers/box"
server lab WS_SLOT=0
refused "WS_SLOT=0 is server ws's slot too" lab

server lab WS_SLOT=1 WS_REMOTE=localhost WS_REMOTE_PORT=22023 WS_PORT=51820
refused "WS_PORT=51820 is server ws's too, behind WS_REMOTE localhost" lab
server lab WS_SLOT=1 WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_PORT=51821
refused "WS_REMOTE_PORT=22022 is server ws's too, behind WS_REMOTE localhost" lab
