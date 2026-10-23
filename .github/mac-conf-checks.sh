#!/bin/sh
# The mac job's checks of mac.conf's WS_TUNNEL_NETWORKS and WS_RESOLVERS, which setup.sh hands to
# lib.sh's functions. The job runs it from the repository's root after mac-lib-checks.sh, with the
# tunnel's daemon loaded and WS_PUBLIC_KEY in mac.conf (.github/workflows/mac.yml). It takes
# ws-tunnel's path from the job's TUNNEL, and stops without it. The keys' network must go into the
# daemon's utun, and their domain show in scutil --dns. A second run and --check must print nothing.
# Emptied or removed, the keys must take both back, and spare a script's own files. Each failure
# prints an ::error:: line, and any fails the script at its end.
#
# shellcheck source-path=SCRIPTDIR/..

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL, ws-tunnel's path, is unset: mac.yml's job sets it"
    exit 1
fi
# shellcheck source=mac/lib.sh
. mac/lib.sh
conf=$HOME/.config/ws/mac.conf
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

# keys LINE... puts LINE... in mac.conf in place of its WS_TUNNEL_NETWORKS and WS_RESOLVERS.
keys() {
    sed -i '' -e '/^WS_TUNNEL_NETWORKS=/d' -e '/^WS_RESOLVERS=/d' "$conf"
    [ $# = 0 ] || printf '%s\n' "$@" >>"$conf"
}

# setup [ARG] runs setup.sh, and sets out to what it printed.
setup() {
    out=$(sh mac/setup.sh "$@" 2>&1)
}

# routed ADDRESS succeeds where macOS routes ADDRESS into the daemon's utun.
routed() {
    route -n get "$1" 2>/dev/null | grep -q "interface: $utun\$"
}

# shown DOMAIN ADDRESS succeeds where scutil --dns has a resolver for DOMAIN whose first
# nameserver is ADDRESS.
shown() {
    scutil --dns | awk -v d="domain : $1" -v n="nameserver[0] : $2" '
        /^resolver #/ { r = 0 }
        { line = $0; gsub(/^ +| +$/, "", line); gsub(/ +/, " ", line) }
        line == d { r = 1 }
        r && line == n { found = 1 }
        END { exit !found }'
}

utun=$(sudo "$TUNNEL" status | awk -F': ' '$1 == "interface" { print $2 }')
# A script's own route and resolver, under lib.sh's header, which the keys must spare.
must ws_tunnel_routes own 198.51.100.0/24
must ws_resolver own.test 198.51.100.53

keys 'WS_TUNNEL_NETWORKS="203.0.113.0/24"' 'WS_RESOLVERS="conf.test=203.0.113.53,203.0.113.54"'
if ! setup --check; then
    error "setup.sh --check with the keys failed: $out"
else
    case $out in *'+203.0.113.0/24'*) ;; *) error "--check showed no network: $out" ;; esac
    case $out in
    *'+nameserver 203.0.113.54'*) ;;
    *) error "--check showed no resolver: $out" ;;
    esac
fi
! sudo test -e "$ws_etc/routes.d/mac-conf" || error '--check wrote routes.d/mac-conf'
! test -e "$ws_resolvers/conf.test" || error "--check wrote $ws_resolvers/conf.test"

setup || error "setup.sh with the keys failed: $out"
printf '%s\n' "$out"
until_ok 20 routed 203.0.113.1 || error "203.0.113.1 is not routed into the daemon's $utun"
until_ok 20 shown conf.test 203.0.113.53 || error 'scutil --dns shows no resolver for conf.test'
case $(head -n 1 "$ws_resolvers/conf.test") in
"# Written by ws's mac/setup.sh"*) ;;
*) error "$ws_resolvers/conf.test does not start with setup.sh's header" ;;
esac
for args in '' --check; do
    if ! setup ${args:+"$args"}; then
        error "a second setup.sh${args:+ $args} failed: $out"
    elif [ -n "$out" ]; then
        error "a second setup.sh${args:+ $args} printed what it should not: $out"
    fi
done

keys 'WS_RESOLVERS="conf.test=203.0.113.53,203.0.113.54,203.0.113.55,203.0.113.56"'
if setup --check; then
    error 'setup.sh took 4 nameservers for a domain'
else
    case $out in
    *'WS_RESOLVERS=conf.test='*'up to 3 IPv4 addresses'*) ;;
    *) error "setup.sh refused 4 nameservers with another error: $out" ;;
    esac
fi

# WS_TUNNEL_NETWORKS emptied, and WS_RESOLVERS removed.
keys 'WS_TUNNEL_NETWORKS='
setup || error "setup.sh with the keys emptied failed: $out"
printf '%s\n' "$out"
until_ok 20 eval '! routed 203.0.113.1' || error "203.0.113.1 is still routed into $utun"
until_ok 20 eval '! shown conf.test 203.0.113.53' || error 'scutil --dns still shows conf.test'
! sudo test -e "$ws_etc/routes.d/mac-conf" || error 'routes.d/mac-conf outlived its key'
! test -e "$ws_resolvers/conf.test" || error "$ws_resolvers/conf.test outlived its key"
sudo test -e "$ws_etc/routes.d/own" || error 'the emptied key took routes.d/own too'
routed 198.51.100.1 || error "198.51.100.1, routes.d/own's, left $utun"
shown own.test 198.51.100.53 || error 'the removed key took own.test too'

keys
must ws_tunnel_routes own
must ws_resolver own.test
exit "$fail"
