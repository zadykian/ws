#!/bin/bash
# A second mise bootstrap with the daemon calls sudo not once, and its plan exits 0. The checks of
# ws client tunnel conf, networks and cable, as their units run them, print nothing and
# exit 0.

set -eo pipefail
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

mac_serve
trap 'kill "$mac_server"' EXIT
fail=0
mac_mark
mac_bootstrap || {
    echo '::error::the second mise bootstrap with the daemon failed'
    exit 1
}
calls=$(mac_calls)
if [ "$calls" != 0 ]; then
    echo "::error::the second mise bootstrap with the daemon called sudo $calls times, not once"
    fail=1
fi
mac_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan with the daemon exited $?, not 0"
    fail=1
}
for part in 'tunnel conf' networks cable; do
    # shellcheck disable=SC2086 # the part's words are the command's
    if ! out=$(ws client $part --check 2>&1) || [ -n "$out" ]; then
        echo "::error::ws client $part --check is not in sync:"
        printf '%s\n' "$out"
        fail=1
    fi
done
exit "$fail"
