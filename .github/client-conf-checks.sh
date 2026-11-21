#!/bin/sh
# The client job's checks of the server's WS_TUNNEL_NETWORKS and WS_RESOLVERS, which mise bootstrap
# writes through ws client networks. The job runs it from the repository's root after
# client-ws-checks.sh (80-conf-keys.sh). The tunnel's daemon is loaded, and
# WS_PUBLIC_KEY is in the server's file (.github/workflows/client.yml). It takes ws's path from the
# job's TUNNEL, and stops without it. ws client networks --check must name both files and
# write none. The keys' network must go into the daemon's utun, and their domain show in scutil
# --dns. A second bootstrap's plan must exit 0. Emptied or removed, the keys must take both back,
# and spare a script's own files. Each failure prints an ::error:: line, and any fails the script
# at its end. Last, it runs client-setup-checks.sh, whose cases run the cable's command. The conf's
# rules are go test's (roles/client/values).
#
# shellcheck source-path=SCRIPTDIR/..

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL, ws's path, is unset: client.yml's job sets it"
    exit 1
fi
tunnel_dir=/etc/ws/tunnel
ws_resolvers=/etc/resolver
conf=/etc/ws/values.yml
fail=0
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# must COMMAND... runs COMMAND in a subshell, so that its failure is an ::error:: too.
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

# keys 'KEY: value'... puts those lines in the server's file in place of its WS_TUNNEL_NETWORKS and WS_RESOLVERS.
keys() {
    conf_set "$conf" WS_TUNNEL_NETWORKS WS_RESOLVERS "$@"
}

# setup runs ws client setup, as client-mise.sh's client_bootstrap runs it, and sets out to what it
# printed.
setup() {
    out=$(bash -c '. .github/client-mise.sh && client_bootstrap' 2>&1)
}

# check runs ws client networks --check, as the networks' unit checks, and sets out to what it
# printed.
check() {
    out=$(ws client networks --check 2>&1)
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

utun=$(sudo "$TUNNEL" client tunnel status | awk -F': ' '$1 == "interface" { print $2 }')
# A script's own route and resolver, which the keys must spare.
must "$TUNNEL" client tunnel routes --server ws own 198.51.100.0/24
must "$TUNNEL" client resolver own.test 198.51.100.53

keys 'WS_TUNNEL_NETWORKS: 203.0.113.0/24' 'WS_RESOLVERS: conf.test=203.0.113.53,203.0.113.54'
if check; then
    error "ws client networks --check with the keys found nothing to do: $out"
else
    case $out in *"$tunnel_dir/ws/routes.d/values"*) ;; *) error "--check named no routes.d/values: $out" ;; esac
    case $out in
    *"$ws_resolvers/conf.test"*) ;;
    *) error "--check named no resolver: $out" ;;
    esac
fi
! sudo test -e "$tunnel_dir/ws/routes.d/values" || error '--check wrote routes.d/values'
! test -e "$ws_resolvers/conf.test" || error "--check wrote $ws_resolvers/conf.test"

setup || error "ws client setup with the keys failed: $out"
printf '%s\n' "$out"
until_ok 20 routed 203.0.113.1 || error "203.0.113.1 is not routed into the daemon's $utun"
until_ok 20 shown conf.test 203.0.113.53 || error 'scutil --dns shows no resolver for conf.test'
case $(head -n 1 "$ws_resolvers/conf.test") in
"# Written by github.com/zadykian/ws: ws client networks"*) ;;
*) error "$ws_resolvers/conf.test does not start with ws client networks's header" ;;
esac
check || error "ws client networks --check after the bootstrap failed: $out"
[ -z "$out" ] || error "ws client networks --check after the bootstrap printed: $out"
out=$(bash -c '. .github/client-mise.sh && client_bootstrap plan --detailed-exitcode' 2>&1) ||
    error "the plan after the bootstrap with the keys failed: $out"

keys 'WS_RESOLVERS: conf.test=203.0.113.53,203.0.113.54,203.0.113.55,203.0.113.56'
if check; then
    error 'ws client networks --check took 4 nameservers for a domain'
else
    case $out in
    *'WS_RESOLVERS=conf.test='*'up to 3 IPv4 addresses'*) ;;
    *) error "ws client networks --check refused 4 nameservers with another error: $out" ;;
    esac
fi

# WS_TUNNEL_NETWORKS emptied, and WS_RESOLVERS removed.
keys 'WS_TUNNEL_NETWORKS: ""'
setup || error "ws client setup with the keys emptied failed: $out"
printf '%s\n' "$out"
until_ok 20 eval '! routed 203.0.113.1' || error "203.0.113.1 is still routed into $utun"
until_ok 20 eval '! shown conf.test 203.0.113.53' || error 'scutil --dns still shows conf.test'
! sudo test -e "$tunnel_dir/ws/routes.d/values" || error 'routes.d/values outlived its key'
! test -e "$ws_resolvers/conf.test" || error "$ws_resolvers/conf.test outlived its key"
sudo test -e "$tunnel_dir/ws/routes.d/own" || error 'the emptied key took routes.d/own too'
routed 198.51.100.1 || error "198.51.100.1, routes.d/own's, left $utun"
shown own.test 198.51.100.53 || error 'the removed key took own.test too'

keys
must "$TUNNEL" client tunnel routes --server ws own
must "$TUNNEL" client resolver own.test

sh .github/client-setup-checks.sh || fail=1
exit "$fail"
