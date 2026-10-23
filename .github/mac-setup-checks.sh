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
nl='
'
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

# stops TEXT [ARG] runs setup.sh, which must stop with one line that starts with TEXT.
stops() {
    text=$1
    shift
    ! setup "$@" || error "setup.sh${*:+ $*} went on where it should stop: $out"
    case $out in
    *"$nl"*) error "setup.sh${*:+ $*} printed more than one line: $out" ;;
    "$text"*) ;;
    *) error "setup.sh${*:+ $*} stopped on '$out', not '$text'" ;;
    esac
}

# The cable's service. Empty, it must leave the network alone, and setup.sh print nothing.
with 'WS_CABLE_SERVICE='
prints ''
prints '' --check

# A stand-in networksetup goes first on PATH, with services whose names no Mac has, so that a call
# of the runner's own changes nothing. It logs each -setmanual with its count of arguments, takes it
# from root alone, and keeps each service's settings in a file.
mkdir "$tmp/bin"
: >"$tmp/setmanual.log"
cat >"$tmp/bin/networksetup" <<'EOF'
#!/bin/sh
state=${0%/bin/*}
info=$state/info-$(printf '%s' "${2-}" | tr -c 'A-Za-z0-9' _)
[ "$1" != -setmanual ] || echo "$# $*" >>"$state/setmanual.log"
[ "$1" != -setmanual ] || [ "$(id -u)" = 0 ] || exit 1
case $1:${2-} in
-listallnetworkservices:)
    printf '%s\n' 'An asterisk (*) denotes that a network service is disabled.' \
        'ws stand-in 10/100/1G/2.5G LAN' '*ws stand-in bridge' 'ws stand-in no-op' \
        'ws stand-in failing'
    ;;
-getinfo:'ws stand-in '*)
    [ -f "$info" ] || printf '%s\n' 'DHCP Configuration' 'IP address: 198.51.100.20' \
        'Subnet mask: 255.255.255.0' 'Router: (null)' >"$info"
    cat "$info"
    printf '%s\n' 'IPv6: Automatic' 'IPv6 Router: none'
    ;;
-setmanual:'ws stand-in no-op') ;;
-setmanual:'ws stand-in failing') exit 1 ;;
-setmanual:'ws stand-in 10/100/1G/2.5G LAN' | -setmanual:'ws stand-in bridge')
    printf '%s\n' 'Manual Configuration' "IP address: $3" "Subnet mask: $4" \
        "Router: ${5:-(null)}" >"$info"
    ;;
*) echo "${2-} is not a recognized network service." && exit 4 ;;
esac
EOF
chmod 755 "$tmp/bin/networksetup"
PATH=$tmp/bin:$PATH
args='10.77.0.2 255.255.255.252'

# logged LINE... checks that the stand-in logged LINE..., a line each, since the last check. Where
# it logged other calls, setup.sh may reach another networksetup, so the script stops there.
logged() {
    got=$(cat "$tmp/setmanual.log")
    : >"$tmp/setmanual.log"
    [ "$got" = "$(printf '%s\n' "$@")" ] && return
    error "the stand-in logged '$got', not '$*', so the cable's other cases go"
    exit 1
}

# differs LINE... gives the no-op service the settings LINE..., on which setup.sh must run one
# -setmanual and stop on the check after it.
differs() {
    printf '%s\n' "$@" >"$tmp/info-ws_stand_in_no_op"
    stops 'setup.sh: networksetup -getinfo "ws stand-in no-op" shows '
    logged "4 -setmanual ws stand-in no-op $args"
}

# --check must print one networksetup -setmanual and run none; a run must run it through sudo and
# print it, and a second run and --check print nothing.
if [ "$(command -v networksetup)" != "$tmp/bin/networksetup" ]; then
    error "networksetup is $(command -v networksetup), not the stand-in, so the cable's cases go"
    exit 1
fi
service='ws stand-in 10/100/1G/2.5G LAN'
with "WS_CABLE_SERVICE=\"$service\""
prints "setup.sh: would run networksetup -setmanual \"$service\" $args" --check
prints "setup.sh: ran networksetup -setmanual \"$service\" $args"
logged "4 -setmanual $service $args"
prints ''
prints '' --check
logged
# A disabled service draws a warning on every run, and still gets the settings.
with 'WS_CABLE_SERVICE="ws stand-in bridge"'
warning="setup.sh: warning: networksetup lists \"ws stand-in bridge\", mac.conf's"
warning="$warning WS_CABLE_SERVICE, as disabled, so the cable carries nothing: turn it on in"
warning="$warning System Settings › Network"
prints "$warning${nl}setup.sh: ran networksetup -setmanual \"ws stand-in bridge\" $args"
prints "$warning"
logged "4 -setmanual ws stand-in bridge $args"
# A service the list lacks stops setup.sh, though its name begins or ends one listed, or is a glob
# that matches one.
for service in 'ws stand-in absent' 'ws stand-in' 'stand-in 10/100/1G/2.5G LAN' 'ws stand-in*'; do
    with "WS_CABLE_SERVICE=\"$service\""
    stops "setup.sh: networksetup -listallnetworkservices lacks \"$service\", mac.conf's"
done
logged
# A -setmanual that fails stops setup.sh.
with 'WS_CABLE_SERVICE="ws stand-in failing"'
stops "setup.sh: networksetup -setmanual \"ws stand-in failing\" $args failed"
logged "4 -setmanual ws stand-in failing $args"
# The no-op's -setmanual exits 0 and changes nothing, which only the check after it catches:
# setup.sh must stop on each setting that differs. The cable's address alone, with no mask or router
# line, must pass.
with 'WS_CABLE_SERVICE="ws stand-in no-op"'
differs 'DHCP Configuration' 'IP address: 10.77.0.2'
differs 'Manually Using DHCP Router Configuration' 'IP address: 10.77.0.2'
differs 'Manual Configuration' 'IP address: 192.0.2.2'
differs 'Manual Configuration' 'IP address: 10.77.0.2/30'
differs 'Manual Configuration' 'IP address: 10.77.0.2' 'Subnet mask: 255.255.255.0'
differs 'Manual Configuration' 'IP address: 10.77.0.2' 'Router: 10.77.0.1'
printf '%s\n' 'Manual Configuration' 'IP address: 10.77.0.2' >"$tmp/info-ws_stand_in_no_op"
prints ''
# So must a Manual Configuration with no address line, as getinfo may show while undocked.
printf '%s\n' 'Manual Configuration' >"$tmp/info-ws_stand_in_no_op"
prints ''
logged
exit "$fail"
