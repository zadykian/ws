#!/bin/sh
# A second run and --check after the first, each: each must print nothing, as
# ws mac apply's own second run does.

set -u

wsbin=$HOME/.local/bin/ws
fail=0

for args in '' --check; do
    if ! out=$("$wsbin" mac wizard ${args:+"$args"} 2>&1); then
        echo "::error::ws mac wizard${args:+ $args} failed:"
    elif [ -n "$out" ]; then
        echo "::error::ws mac wizard${args:+ $args} printed what it should not:"
    else
        continue
    fi
    printf '%s\n' "$out"
    fail=1
done
exit "$fail"
