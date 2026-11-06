#!/bin/bash
# At a terminal, ws mac apply --check draws the run screen, lists the Mac's steps and leaves it.
# script gives ws that terminal, and no window size but stty's.

set -eo pipefail

status=0
TERM=xterm script -q /dev/null sh -c 'stty cols 100 rows 30; ws mac apply --check' \
    </dev/null >"$RUNNER_TEMP/screen.log" 2>&1 || status=$?
fail=0
if [ "$status" != 0 ]; then
    echo "::error::ws mac apply --check exited $status at a terminal"
    fail=1
fi
for mark in $'\e[?1049h' $'\e[?1049l' networks; do
    grep -qF "$mark" "$RUNNER_TEMP/screen.log" ||
        {
            echo "::error::ws mac apply --check drew no ${mark#$'\e'} at a terminal"
            fail=1
        }
done
exit "$fail"
