#!/bin/sh
# The mac job's checks of ws client macos tunnel loaded, of its routes and of ws client macos
# resolver. They replaced the functions of a script that another repository sourced. The job
# runs it from the repository's root, with ws's path in TUNNEL, once ws client macos apply has
# loaded the daemon (.github/workflows/mac.yml). A network that ws client macos tunnel
# routes adds must reach the daemon's utun; a domain that ws client macos resolver adds, scutil
# --dns.

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL, ws's path, is unset: mac.yml's job sets it"
    exit 1
fi
log=/var/log/ws-tunnel.ws.log
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# must COMMAND... runs a subcommand in a subshell, so that a failure is an ::error:: too.
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
    sudo launchctl print "system/$LABEL" 2>/dev/null |
        awk '$1 == "pid" && p == "" { p = $3 } END { print (p == "" ? "none" : p) }'
}

"$TUNNEL" client macos tunnel loaded || error 'ws client macos tunnel loaded fails with the daemon loaded'
for cidr in 0.0.0.0/0 10.99.0.0/16 10.77.0.0/29 198.51.100.5/24; do
    if ("$TUNNEL" client macos tunnel routes --server ws test "$cidr") 2>/dev/null; then
        error "ws client macos tunnel routes took $cidr"
    fi
done
if ! out=$("$TUNNEL" client macos tunnel routes --server ws --check test 198.51.100.0/24 2>&1); then
    error "ws client macos tunnel routes --server ws --check test 198.51.100.0/24 failed: $out"
else
    case $out in *'+198.51.100.0/24'*) ;; *) error "--check showed no diff: $out" ;; esac
fi
! sudo test -e /etc/ws-tunnel/ws/routes.d/test || error '--check wrote ws/routes.d/test'

# The daemon takes each SIGHUP of ws client macos tunnel routes in place. It keeps its pid and logs
# a reload of its routes.d. A daemon that died of a SIGHUP, and that launchd started again, would log a
# start.
was=$(pid)
lines=$(sudo awk 'END { print NR }' "$log")
must "$TUNNEL" client macos tunnel routes --server ws test 198.51.100.0/24
utun=$(sudo "$TUNNEL" client macos tunnel status | awk -F': ' '$1 == "interface" { print $2 }')
# shellcheck disable=SC2329 # called through until_ok
routed() { route -n get 198.51.100.1 | grep -q "interface: $utun\$"; }
until_ok 20 routed || error "198.51.100.1 is not routed into the daemon's $utun"
route -n get 198.51.100.1 || true
must "$TUNNEL" client macos tunnel routes --server ws test
until_ok 20 eval '! routed' || error "198.51.100.1 is still routed into $utun"
! sudo test -e /etc/ws-tunnel/ws/routes.d/test || error 'ws/routes.d/test outlived its removal'
now=$(pid)
if [ "$was" = none ] || [ "$now" != "$was" ]; then
    error "the daemon was pid $was before ws client macos tunnel routes's SIGHUPs, and is pid $now"
fi
new=$(sudo tail -n "+$((lines + 1))" "$log")
case $new in
*'read its settings again'*) ;;
*) error "the daemon logged no reload of its routes.d" ;;
esac
case $new in *' started on '*) error 'the daemon started again on a SIGHUP' ;; esac

# A /32's answer of route -n get may hold no mask line, as a host route's. The daemon must take it
# for the network's own route all the same, and delete it once routes.d drops it.
must "$TUNNEL" client macos tunnel routes --server ws host 192.0.2.77/32
# shellcheck disable=SC2329 # called through until_ok
hosted() { route -n get 192.0.2.77 | grep -q "interface: $utun\$"; }
until_ok 20 hosted || error "192.0.2.77 is not routed into the daemon's $utun"
route -n get -inet 192.0.2.77/32 || true
must "$TUNNEL" client macos tunnel routes --server ws host
until_ok 20 eval '! hosted' || error "192.0.2.77/32 is still routed into $utun"

# shellcheck disable=SC2329 # called through until_ok
shown() {
    scutil --dns | awk '/^resolver #/ { d = 0 } /domain *: example\.test$/ { d = 1 }
        d && /nameserver\[0\] *: 198\.51\.100\.53$/ { found = 1 } END { exit !found }'
}
must "$TUNNEL" client macos resolver example.test 198.51.100.53
until_ok 20 shown || error 'scutil --dns shows no resolver for example.test'
scutil --dns | grep -B 1 -A 3 'example.test' || true
must "$TUNNEL" client macos resolver example.test
until_ok 20 eval '! shown' || error 'scutil --dns still shows example.test'
exit "$fail"
