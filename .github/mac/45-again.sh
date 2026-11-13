#!/bin/bash
# A second run and --check print nothing.

set -eo pipefail

fail=0
for args in '' --check; do
    if ! out=$(ws client macos apply ${args:+"$args"} 2>&1); then
        echo "::error::ws client macos apply${args:+ $args} failed:"
    elif [ -n "$out" ]; then
        echo "::error::ws client macos apply${args:+ $args} printed what it should not:"
    else
        continue
    fi
    printf '%s\n' "$out"
    fail=1
done
exit "$fail"
