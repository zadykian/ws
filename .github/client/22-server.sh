#!/bin/bash
# --server names one of several servers. With a second server's file beside ws's, ws macos
# values print must refuse to pick one. It must refuse a name with no file or of a bad form. With
# --server ws it must go on as with one server. The second file goes with the case, even on a failure.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh

conf=/etc/ws
second=$conf/servers/lab
trap 'sudo rm -rf "$second"' EXIT

# Lab's file, as a second server's: conf_server names it.
conf_server=lab
{
    echo 'WS_SLOT: 1'
} | conf_write "$conf"
conf_server=ws

# refused WHY [ARG...]: ws macos values print ARG... must exit 1, with WHY in what it prints.
refused() {
    why=$1
    shift
    status=0
    out=$(ws macos values print "$@" 2>&1) || status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$why"; then
        echo "::error::ws macos values print $* exited $status, printing '$out', where 1 and '$why' were due"
        exit 1
    fi
}

refused 'several servers in'
refused 'lab and ws: name one by --server'
refused 'no server named box in' --server box
refused "--server Lab is not a server's name" --server Lab
ws macos values print --server ws >"$RUNNER_TEMP/server.out" 2>&1 || {
    cat "$RUNNER_TEMP/server.out"
    echo '::error::ws macos values print --server ws failed with a second server beside it'
    exit 1
}
