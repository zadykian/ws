#!/bin/sh
# A second run and --check after the first, each by sh and by dash: each must print nothing, as
# setup.sh's own second run does.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
dash=$(command -v dash)
fail=0

for shell in sh "$dash"; do
    for args in '' --check; do
        if ! out=$("$shell" "$wizard" ${args:+"$args"} 2>&1); then
            echo "::error::wizard.sh${args:+ $args} by $shell failed:"
        elif [ -n "$out" ]; then
            echo "::error::wizard.sh${args:+ $args} by $shell printed what it should not:"
        else
            continue
        fi
        printf '%s\n' "$out"
        fail=1
    done
done
exit "$fail"
