#!/bin/bash
# The build of ws that .tools/install-build put in /usr/local/bin, from this clone, with the Go
# that .tools/run pins. The binary must run and refuse an unknown command with status 2.

set -eo pipefail

ws=/usr/local/bin/ws
out=$("$ws")
echo "$out"
grep -q '^Usage:' <<<"$out"
status=0
"$ws" nonesuch || status=$?
[ "$status" = 2 ]
