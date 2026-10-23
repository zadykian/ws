#!/bin/sh
# The mac job's checks of mac/setup.sh's own steps, each run with its key in the job's mac.conf, in
# a section per key. mac-conf-checks.sh runs it last, from the repository's root, once setup.sh has
# set up the Mac (.github/workflows/mac.yml). As it runs setup.sh, sudo included, and edits the
# job's mac.conf, it stops without the job's TUNNEL, and puts the file back as it exits, on a
# signal too. Each failure prints an ::error:: line, and any fails the script at its end.

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL is unset: this runs setup.sh, with sudo, in mac.yml's job alone"
    exit 1
fi
conf=$HOME/.config/ws/mac.conf
tmp=$(mktemp -d "${TMPDIR:-/tmp}/mac-setup-checks.XXXXXXXX")
trap '[ ! -f "$tmp/job.conf" ] || cp "$tmp/job.conf" "$conf"; rm -rf "$tmp"' EXIT
trap 'exit 1' HUP INT TERM
cp "$conf" "$tmp/job.conf"
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# with LINE... gives conf, the job's mac.conf, its own lines and then LINE..., a line each.
with() {
    cp "$tmp/job.conf" "$conf"
    [ $# = 0 ] || printf '%s\n' "$@" >>"$conf"
}

# setup [ARG] runs setup.sh, and sets out to what it printed.
setup() {
    out=$(sh mac/setup.sh "$@" 2>&1)
}

# prints TEXT [ARG] runs setup.sh, which must print TEXT alone.
prints() {
    text=$1
    shift
    setup "$@" || error "setup.sh${*:+ $*} failed: $out"
    [ "$out" = "$text" ] || error "setup.sh${*:+ $*} printed '$out', not '$text'"
}

# The cable's service. Empty, it must leave the network alone, and setup.sh print nothing.
with 'WS_CABLE_SERVICE='
prints ''
prints '' --check
exit "$fail"
