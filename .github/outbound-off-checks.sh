#!/bin/bash
# The server job's last cases, as root on the runner that server-checks.sh and secrets-checks.sh
# set up (.github/workflows/server.md). They turn the outbound table off, as a cloud VM has it
# (roles/server/ubuntu/security/README.md#outbound). They come last, as no later case may need the
# table they take off. Each refusal has a control that holds while the table is on.
set -euo pipefail

# The runner's sudo keeps the runner's XDG_CONFIG_HOME, which would send root's settings to the
# runner's home.
unset XDG_CONFIG_HOME
export HOME=/root DEBIAN_FRONTEND=noninteractive

fail() {
    echo "::error::$*"
    exit 1
}

must() {
    local name=$1
    shift
    "$@" || fail "$name"
    echo "ok: $name"
}

refused() {
    local name=$1
    shift
    ! "$@" || fail "$name"
    echo "ok: $name"
}

check=$(dirname "$0")/warnings-check.sh
setup() { "$check" -- "$(dirname "$0")/bootstrap.sh" --in-place "$@" </dev/null; }
exits() {
    local want=$1 status=0
    shift
    "$@" || status=$?
    [ "$status" = "$want" ]
}

# The table's units, and the files that the security role drops beside their own: its scripts and
# the fill's record of what it set.
units=(security-resolvers.path security-resolvers.timer security-resolvers.service
    security-outbound.service)
files=(/usr/local/lib/ws/security /run/security-resolvers.conf)
table() { nft list table inet ws >/dev/null 2>&1; }
# gone holds where no file of the table's is left: no unit file, no link that enables one, and
# none of $files. It prints each one left.
gone() {
    local left
    left=$(
        find /etc/systemd/system -name 'security-*'
        find "${files[@]}" -maxdepth 0 2>/dev/null || true
    )
    [ -z "$left" ] || {
        printf '%s\n' "$left"
        return 1
    }
}

# The controls: each holds while the table is on, and its opposite once the table is off.
must "the table is in force while it is on" table
must "the table's unit runs while it is on" systemctl is-active --quiet security-outbound.service
refused "no file of the table's is left while it is on" gone

# As on a cloud VM, the value goes in the values file. The plan shows the removals. Setup runs
# twice, and its plan must exit 0 after each run: a stopped unit plans no stop again.
echo 'WS_SECURITY_OUTBOUND: false' >>/etc/ws/values.yml
must "the plan with the table off has changes to make" exits 2 setup --plan
must "the run with the table off runs" setup
must "the run with the table off leaves nothing to do" setup --plan
must "the second run with the table off runs" setup
must "the second run with the table off leaves nothing to do" setup --plan

refused "the table is in force with it off" table
refused "a unit of the table's is active with it off" systemctl is-active --quiet "${units[@]}"
must "no file of the table's is left with it off" gone
