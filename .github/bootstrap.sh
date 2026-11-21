#!/bin/sh
# CI's run of install.sh, as on a new machine, in the server job. It runs on the tree in /opt/ws
# in place, as the checkout is not a clone for install.sh to make. Run as
# `bootstrap.sh DIR`, DIR being .tools/serve-build's. Its serve-releases serves the tree's build on
# loopback in the release's place while install.sh runs. The server stops with the script, by its
# own pid.
set -eu
dir=$1
port=$(mktemp)
"$dir/serve-releases" "$dir" 8765 "$port" &
server=$!
trap 'kill "$server"; rm -f "$port"' EXIT
for _ in $(seq 50); do
    [ -s "$port" ] && break
    sleep 0.1
done
if [ ! -s "$port" ]; then
    echo '::error::serve-releases did not start.'
    exit 1
fi
WS_IN_PLACE=1 sh /opt/ws/install.sh
