#!/bin/bash
# --server names one of several servers. With a second server's file beside ws's, ws client macos
# apply --check must refuse to pick one. It must refuse a name with no file or of a bad form. With
# --server ws it must go on as with one server. The second file goes with the case, even on a failure.

set -eo pipefail
# shellcheck source=.github/mac-conf.sh
. .github/mac-conf.sh

conf=$HOME/.config/ws/mac.conf
second=${conf%/*}/servers/lab
trap 'rm -rf "$second"' EXIT

# Lab's file, as a second server's: conf_server names it, and mac.conf keeps its lines.
conf_server=lab
{
    cat "$conf"
    echo WS_USER=lab
    echo WS_SLOT=1
} | conf_write "$conf"
conf_server=ws

# refused WHY [ARG...]: ws client macos apply --check ARG... must exit 1, with WHY in what it prints.
refused() {
    why=$1
    shift
    status=0
    out=$(ws client macos apply --check "$@" 2>&1) || status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$why"; then
        echo "::error::ws client macos apply --check $* exited $status, printing '$out', where 1 and '$why' were due"
        exit 1
    fi
}

refused 'several servers in'
refused 'lab and ws: name one by --server'
refused 'no server named box in' --server box
refused "--server Lab is not a server's name" --server Lab
ws client macos apply --check --server ws >"$RUNNER_TEMP/server.out" 2>&1 || {
    cat "$RUNNER_TEMP/server.out"
    echo '::error::ws client macos apply --check --server ws failed with a second server beside it'
    exit 1
}
