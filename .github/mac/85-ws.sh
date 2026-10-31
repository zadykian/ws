#!/bin/bash
# bootstrap.sh's build of ws into ~/.local/bin, from this clone: its functions, all but its last
# line, which runs main, then build. The binary must run and refuse an unknown command with
# status 2. Its Go is the one tools/run pins.

set -eo pipefail

ws=$HOME/.local/bin/ws
rm -f "$ws"
# shellcheck disable=SC2016 # the sh that reads them expands them
{
    sed '$d' mac/bootstrap.sh
    echo 'check=0 dir=$PWD'
    echo build
} | sh
out=$("$ws")
echo "$out"
grep -q '^usage: ws' <<<"$out"
status=0
"$ws" nonesuch || status=$?
[ "$status" = 2 ]
