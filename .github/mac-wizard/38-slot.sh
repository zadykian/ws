#!/bin/sh
# A server at slot 1 and port 51821: --check must plan ws's run after naming wireguard_address and
# wireguard_port in ws's host_vars, which lacks both. It must write nothing, neither a pair nor a
# change to host_vars. The conf goes back as it stood.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
vars="$ws/root"/etc/ws/host_vars.yml
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# written prints a sum of host_vars and of the log of values' pairs, as it stands.
written() {
    sudo cat "$vars" "$ws/values.log" 2>/dev/null | cksum
}

conf_print "$conf" >"$RUNNER_TEMP/slot.conf"
before=$(written)
conf_set "$conf" WS_SLOT=1 WS_PORT=51821
out=$("$wsbin" client macos wizard --check --plain </dev/null 2>&1)
status=$?
conf_print "$RUNNER_TEMP/slot.conf" | conf_write "$conf"
printf '%s\n' "$out"
[ "$status" = 0 ] || error "ws client macos wizard --check at slot 1 exited $status"
printf '%s\n' "$out" | grep -qF -e "after naming wireguard_address: 10.99.1.1/24 and wireguard_port: 51821 in ws's host_vars" ||
    error "ws client macos wizard --check at slot 1 named no wireguard pairs before ws's run"
[ "$(written)" = "$before" ] || error "ws client macos wizard --check at slot 1 wrote host_vars or a pair"
exit "$fail"
