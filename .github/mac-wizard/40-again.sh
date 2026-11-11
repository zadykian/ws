#!/bin/sh
# A second run and --check after the first, each: each must print nothing, as
# ws mac apply's own second run does.

set -u

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws mac wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
fail=0

for args in '' --check; do
    if ! out=$("$wsbin" mac wizard ${args:+"$args"} 2>&1); then
        echo "::error::ws mac wizard${args:+ $args} failed:"
    elif [ -n "${out#"$known"}" ]; then
        echo "::error::ws mac wizard${args:+ $args} printed what it should not:"
    else
        continue
    fi
    printf '%s\n' "$out"
    fail=1
done
exit "$fail"
