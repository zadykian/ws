#!/bin/sh
# The wizard's remote script on ws, which the Mac's wizard sends to root there over ssh. So it is
# the Mac's clone's, whatever ws's clone holds, or where ws has none:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote.sh
#
# - status prints what the wizard reads of ws, a line each, as NAME VALUE, with - for a value that
#   is missing. run is the commit and status of the last full run, and last the unit and status of
#   the last run. active is the run's unit that runs now. mac.pub is the Mac's key that wg0's peer
#   takes, and wg0 is wg0's own key.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, and the Mac's key for wg0. Each is a line NAME=PATH, which CI's stand-in for
# ws roots in a directory of its own.
state=/var/lib/ws-wizard
peer=/etc/wireguard/mac.pub

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# first prints the first line of its stdin, or - where it has none.
first() {
    awk 'NR == 1 { print; found = 1 } END { if (!found) print "-" }'
}

# stamp FILE prints the two words of the stamp FILE, or - where it has none.
stamp() {
    awk 'NF == 2 { print; exit }' "$1" 2>/dev/null | first
}

# active prints the run's unit that is active or activating, where there is one.
active() {
    systemctl list-units --type=service --state=active,activating --no-legend --plain \
        'ws-wizard-*' 2>/dev/null | awk '{ sub(/\.service$/, "", $1); print $1; exit }'
}

# status prints what the wizard reads of ws, as the header says.
status() {
    echo "run $(stamp "$state/run")"
    echo "last $(stamp "$state/last")"
    echo "active $(active | first)"
    echo "mac.pub $(head -n 1 "$peer" 2>/dev/null | first)"
    echo "wg0 $(wg show wg0 public-key 2>/dev/null | first)"
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    status) status ;;
    *) die "unknown op '${1-}'; usage: wizard-remote.sh status" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
