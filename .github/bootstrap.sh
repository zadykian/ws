#!/bin/sh
# CI's run of ws server setup, as on a new machine, in the server job. It runs on the tree in
# /opt/ws in place, as the checkout is not a tag for setup to fetch. Run as
# `bootstrap.sh DIR [ARG]...`, DIR being .tools/serve-build's, whose build of ws runs setup, and
# ARG setup's flags, --in-place and --plan. DIR's serve-releases serves that build on loopback in
# the release's place while setup runs. The server stops with the script, by its own pid.
set -eu
dir=$1
shift
set -- "$dir"/download/v*/ws-linux-amd64 server setup "$@"
if [ ! -x "$1" ]; then
    echo "::error::$dir holds no build of ws for linux amd64."
    exit 1
fi
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
"$@"
