#!/bin/sh
# The real machine's check from another machine on the home LAN: ubuntu/checks/README.md, check 8.
# Run as
#
#   lan.sh HOST_ADDRESS [GUEST_ADDRESS...]
#
# HOST_ADDRESS is the host's address on the LAN, and GUEST_ADDRESS a guest's address on the bridge.
# The host must answer SSH and no other TCP port that the list holds. No guest may answer a ping or
# SSH, even where this machine routes its address through the host: for that it adds a host route
# by sudo, and removes it. It prints a line a check, ok or FAIL, and exits 1 where any fails.
set -u

fails=0
routed=

pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    fails=$((fails + 1))
}

# ports are the TCP ports a service of ws's or Docker's could open by mistake.
ports="53 80 443 2375 3301 4317 4318 8080 8443 9000 9100 13133 14317 14318"

open() { nc -z -w 3 "$1" "$2" >/dev/null 2>&1; }

pings() { ping -c 2 "$1" >/dev/null 2>&1; }

# route_to adds or deletes a host route to the guest through the host, as the OS has it.
route_to() {
    case $(uname) in
    Darwin) sudo route -n "$1" -host "$2" "$3" >/dev/null 2>&1 ;;
    *) sudo ip route "$1" "$2" via "$3" >/dev/null 2>&1 ;;
    esac
}

cleanup() {
    for g in $routed; do route_to delete "$g" "$host"; done
}

[ $# -ge 1 ] || {
    echo "usage: lan.sh HOST_ADDRESS [GUEST_ADDRESS...]" >&2
    exit 2
}
host=$1
shift
trap cleanup EXIT

if open "$host" 22; then pass "8 the host answers SSH"; else fail "8 the host answers SSH" "no connection to $host:22"; fi
for p in $ports; do
    if open "$host" "$p"; then fail "8 the host takes TCP $p" "$host:$p connects"; else pass "8 the host takes no TCP $p"; fi
done
for g in "$@"; do
    if route_to add "$g" "$host"; then routed="$routed $g"; fi
    if pings "$g"; then fail "8 the guest $g answers no ping" "it answered"; else pass "8 the guest $g answers no ping"; fi
    if open "$g" 22; then fail "8 the guest $g answers no SSH" "it connected"; else pass "8 the guest $g answers no SSH"; fi
done
[ "$fails" -eq 0 ]
