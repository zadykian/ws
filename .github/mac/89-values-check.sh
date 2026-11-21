#!/bin/bash
# The frame's check of the values. With the server's WS_LAN_ADDRESS dropped, which WS_LAN needs
# beside it, mise bootstrap's plan must fail and name the key and its file, as ws client values
# check does. The key goes back with the case, even on a failure, and the plan must pass again.
# Its detailed exit code is not asked: 76-subcommands.sh wrote the tunnel's routes, so the plan
# holds the daemon's reload until 96-converged.sh's bootstrap.

set -eo pipefail
# shellcheck source=.github/mac-conf.sh
. .github/mac-conf.sh
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

conf=/etc/ws/values.yml
line=$(conf_print "$conf" | grep -x 'WS_LAN_ADDRESS: .*')
trap 'conf_set "$conf" "$line"' EXIT

conf_set "$conf" WS_LAN_ADDRESS
status=0
out=$(mac_bootstrap plan 2>&1) || status=$?
if [ "$status" = 0 ] || ! printf '%s\n' "$out" | grep -qF 'servers/ws/values.yml: WS_LAN and WS_LAN_ADDRESS go together'; then
    echo "::error::mise bootstrap's plan exited $status, printing '$out', where a failure and the pair's refusal were due"
    exit 1
fi

conf_set "$conf" "$line"
out=$(mac_bootstrap plan 2>&1) || {
    echo "::error::the plan with the key back failed: $out"
    exit 1
}
