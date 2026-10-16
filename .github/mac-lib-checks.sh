#!/bin/sh
# The mac job's checks of the functions that mac/lib.sh offers a private repository's own script.
# The job runs it from the repository's root once setup.sh has loaded the tunnel's daemon
# (.github/workflows/mac.yml). It takes ws-tunnel's path from the job's TUNNEL, and stops without
# it. A network that ws_tunnel_routes adds must go into the daemon's utun and out again, with the
# daemon reading routes.d in place. A domain that ws_resolver adds must show in scutil --dns. Each
# failure prints an ::error:: line, and any fails the script at its end.
#
# shellcheck source-path=SCRIPTDIR/..

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL, ws-tunnel's path, is unset: mac.yml's job sets it"
    exit 1
fi
# shellcheck source=mac/lib.sh
. mac/lib.sh
log=/var/log/ws-tunnel.log
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# must COMMAND... runs a function of lib.sh's in a subshell, as one exits on an error, so that the
# failure is an ::error:: too.
must() {
    ("$@") || error "$* failed, as above"
}

# until_ok SECONDS COMMAND... polls COMMAND every second.
until_ok() {
    n=$1
    shift
    while ! "$@" >/dev/null 2>&1; do
        n=$((n - 1))
        [ "$n" -gt 0 ] || return 1
        sleep 1
    done
}

# pid prints the daemon's pid as launchd has it, or none.
pid() {
    sudo launchctl print "system/$ws_label" 2>/dev/null |
        awk '$1 == "pid" && p == "" { p = $3 } END { print (p == "" ? "none" : p) }'
}

ws_tunnel_loaded || error 'ws_tunnel_loaded fails with the daemon loaded'
for cidr in 0.0.0.0/0 10.99.0.0/16 10.77.0.0/29 198.51.100.5/24; do
    if (ws_tunnel_routes test "$cidr") 2>/dev/null; then
        error "ws_tunnel_routes took $cidr"
    fi
done
if ! out=$(WS_CHECK=1 ws_tunnel_routes test 198.51.100.0/24 2>&1); then
    error "WS_CHECK=1 ws_tunnel_routes test 198.51.100.0/24 failed: $out"
else
    case $out in *'+198.51.100.0/24'*) ;; *) error "WS_CHECK showed no diff: $out" ;; esac
fi
! sudo test -e "$ws_etc/routes.d/test" || error 'WS_CHECK wrote routes.d/test'

# The daemon takes each SIGHUP of ws_tunnel_routes in place: it keeps its pid and logs a reload of
# routes.d. A daemon that died of a SIGHUP, and that launchd started again, would log a start.
was=$(pid)
lines=$(sudo awk 'END { print NR }' "$log")
must ws_tunnel_routes test 198.51.100.0/24
utun=$(sudo "$TUNNEL" status | awk -F': ' '$1 == "interface" { print $2 }')
# shellcheck disable=SC2329 # called through until_ok
routed() { route -n get 198.51.100.1 | grep -q "interface: $utun\$"; }
until_ok 20 routed || error "198.51.100.1 is not routed into the daemon's $utun"
route -n get 198.51.100.1 || true
must ws_tunnel_routes test
until_ok 20 eval '! routed' || error "198.51.100.1 is still routed into $utun"
! sudo test -e "$ws_etc/routes.d/test" || error 'routes.d/test outlived its removal'
now=$(pid)
if [ "$was" = none ] || [ "$now" != "$was" ]; then
    error "the daemon was pid $was before ws_tunnel_routes's SIGHUPs, and is pid $now"
fi
new=$(sudo tail -n "+$((lines + 1))" "$log")
case $new in
*'read its settings again'*) ;;
*) error 'the daemon logged no reload of routes.d' ;;
esac
case $new in *' started on '*) error 'the daemon started again on a SIGHUP' ;; esac

# shellcheck disable=SC2329 # called through until_ok
shown() {
    scutil --dns | awk '/^resolver #/ { d = 0 } /domain *: example\.test$/ { d = 1 }
        d && /nameserver\[0\] *: 198\.51\.100\.53$/ { found = 1 } END { exit !found }'
}
must ws_resolver example.test 198.51.100.53
until_ok 20 shown || error 'scutil --dns shows no resolver for example.test'
scutil --dns | grep -B 1 -A 3 'example.test' || true
must ws_resolver example.test
until_ok 20 eval '! shown' || error 'scutil --dns still shows example.test'
exit "$fail"
