#!/bin/bash
# .tools/build's build of ws into ~/.local/bin, from this clone, with the Go that .tools/run pins.
# The binary must run and refuse an unknown command with status 2.

set -eo pipefail

ws=$HOME/.local/bin/ws
rm -f "$ws"
sh .tools/build darwin arm64 "$ws"
out=$("$ws")
echo "$out"
grep -q '^Usage:' <<<"$out"
status=0
"$ws" nonesuch || status=$?
[ "$status" = 2 ]
