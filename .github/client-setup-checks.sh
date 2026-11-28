#!/bin/sh
# The client job's checks of ws macos cable, which the cable's unit runs. They use the job's
# server file. client-conf-checks.sh runs it last, from the repository's root. By then mise bootstrap
# has set up the Mac (.github/workflows/client.yml). As it runs the cable's command, sudo included,
# and edits the job's server file, it stops without the job's TUNNEL. It puts the file back as it exits,
# on a signal too. Each failure prints an ::error:: line, and any fails the script at its end.

set -eu
if [ -z "${TUNNEL-}" ]; then
    echo "::error::TUNNEL is unset: this runs ws macos cable, with sudo, in client.yml's job alone"
    exit 1
fi
conf=/etc/ws
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh
# macOS's TMPDIR ends in a slash, which would put // in $tmp.
tmpdir=${TMPDIR:-/tmp}
tmp=$(mktemp -d "${tmpdir%/}/client-setup-checks.XXXXXXXX")
trap '[ ! -f "$tmp/job.conf" ] || conf_write "$conf" <"$tmp/job.conf"; rm -rf "$tmp"' EXIT
trap 'exit 1' HUP INT TERM
conf_print "$conf" >"$tmp/job.conf"
nl='
'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# with 'KEY: value'... gives conf, the job's server file, those lines in place of the job's of the same
# keys, as the strict reader takes a key once.
with() {
    conf_write "$conf" <"$tmp/job.conf"
    [ $# = 0 ] || conf_set "$conf" "$@"
}

# setup [ARG] runs ws macos cable, and sets out to what it printed.
setup() {
    out=$(ws macos cable "$@" 2>&1)
}

# prints TEXT [ARG] runs ws macos cable, which must print TEXT alone.
prints() {
    text=$1
    shift
    setup "$@" || error "ws macos cable${*:+ $*} failed: $out"
    [ "$out" = "$text" ] || error "ws macos cable${*:+ $*} printed '$out', not '$text'"
}

# stops TEXT [ARG] runs ws macos cable, which must stop with one line that starts with TEXT.
stops() {
    text=$1
    shift
    ! setup "$@" || error "ws macos cable${*:+ $*} went on where it should stop: $out"
    case $out in
    *"$nl"*) error "ws macos cable${*:+ $*} printed more than one line: $out" ;;
    "$text"*) ;;
    *) error "ws macos cable${*:+ $*} stopped on '$out', not '$text'" ;;
    esac
}

# The cable's service. Empty, it must leave the network alone, and the command print nothing.
with 'WS_CABLE_SERVICE: ""'
prints ''
prints '' --check

# A stand-in networksetup goes first on PATH, with services whose names no Mac has, so that a call
# of the runner's own changes nothing. It logs each -setmanual with its count of arguments, takes it
# from root alone, and keeps in files each service's settings and any line its getinfo writes to
# stderr.
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
    [ ! -f "$info.err" ] || cat "$info.err" >&2
    [ -f "$info" ] || printf '%s\n' 'DHCP Configuration' 'IP address: 198.51.100.20' \
        'Subnet mask: 255.255.255.0' 'Router: (null)' >"$info"
    cat "$info"
    ! grep -q . "$info" || printf '%s\n' 'IPv6: Automatic' 'IPv6 Router: none'
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
# it logged other calls, ws macos cable may reach another networksetup, so the script stops
# there.
logged() {
    got=$(cat "$tmp/setmanual.log")
    : >"$tmp/setmanual.log"
    [ "$got" = "$(printf '%s\n' "$@")" ] && return
    error "the stand-in logged '$got', not '$*', so the cable's other cases go"
    exit 1
}

# differs WHAT LINE... gives the no-op service the settings LINE..., on which ws macos cable
# must run one -setmanual and stop, naming WHAT.
differs() {
    line="networksetup -setmanual \"ws stand-in no-op\" $args"
    what="ws macos cable: networksetup -getinfo \"ws stand-in no-op\" shows $1 after"
    shift
    printf '%s\n' "$@" >"$tmp/info-ws_stand_in_no_op"
    stops "$what $line: "
    logged "4 -setmanual ws stand-in no-op $args"
}

# --check must fail on one networksetup -setmanual and run none; a run must run it through sudo
# and print it, and a second run and --check print nothing.
if [ "$(command -v networksetup)" != "$tmp/bin/networksetup" ]; then
    error "networksetup is $(command -v networksetup), not the stand-in, so the cable's cases go"
    exit 1
fi
service='ws stand-in 10/100/1G/2.5G LAN'
with "WS_CABLE_SERVICE: $service"
stops "ws macos cable: to do: run networksetup -setmanual \"$service\" $args" --check
logged
prints "ws macos cable: run networksetup -setmanual \"$service\" $args"
logged "4 -setmanual $service $args"
prints ''
prints '' --check
logged
# A disabled service draws a warning on every run, and still gets the settings.
with 'WS_CABLE_SERVICE: ws stand-in bridge'
warning="ws macos cable: warning: networksetup lists \"ws stand-in bridge\", WS_CABLE_SERVICE in"
warning="$warning the server's values.yml, as disabled, so the cable carries nothing: turn it on in"
warning="$warning System Settings › Network"
prints "$warning${nl}ws macos cable: run networksetup -setmanual \"ws stand-in bridge\" $args"
prints "$warning"
logged "4 -setmanual ws stand-in bridge $args"
# A service the list lacks stops ws macos cable, though its name begins or ends one listed,
# or is a glob that matches one.
for service in 'ws stand-in absent' 'ws stand-in' 'stand-in 10/100/1G/2.5G LAN' 'ws stand-in*'; do
    with "WS_CABLE_SERVICE: $service"
    stops "ws macos cable: networksetup -listallnetworkservices lacks \"$service\", WS_CABLE_SERVICE in the server's values.yml"
done
logged
# A -setmanual that fails stops ws macos cable.
with 'WS_CABLE_SERVICE: ws stand-in failing'
stops "ws macos cable: networksetup -setmanual \"ws stand-in failing\" $args failed: "
logged "4 -setmanual ws stand-in failing $args"
# The no-op's -setmanual exits 0 and changes nothing, which only the check after it catches.
# So ws macos cable must stop on each setting that differs, and name it. The cable's address
# alone, with no mask or router line, must pass.
with 'WS_CABLE_SERVICE: ws stand-in no-op'
differs 'DHCP Configuration' 'DHCP Configuration' 'IP address: 10.77.0.2'
differs 'Manually Using DHCP Router Configuration' 'Manually Using DHCP Router Configuration' \
    'IP address: 10.77.0.2'
differs 'IP address: 192.0.2.2' 'Manual Configuration' 'IP address: 192.0.2.2'
differs 'IP address: 10.77.0.2/30' 'Manual Configuration' 'IP address: 10.77.0.2/30'
differs 'Subnet mask: 255.255.255.0' 'Manual Configuration' 'IP address: 10.77.0.2' \
    'Subnet mask: 255.255.255.0'
differs 'Router: 10.77.0.1' 'Manual Configuration' 'IP address: 10.77.0.2' 'Router: 10.77.0.1'
differs nothing
printf '%s\n' 'Manual Configuration' 'IP address: 10.77.0.2' >"$tmp/info-ws_stand_in_no_op"
prints ''
# So must a Manual Configuration with no address line, as getinfo may show while undocked.
printf '%s\n' 'Manual Configuration' >"$tmp/info-ws_stand_in_no_op"
prints ''
logged
# What getinfo writes to stderr reaches the terminal, and counts as none of its settings.
echo 'networksetup: a diagnostic' >"$tmp/info-ws_stand_in_no_op.err"
prints 'networksetup: a diagnostic'
rm "$tmp/info-ws_stand_in_no_op.err"
logged
# A router outside the cable's /30, an uplink's, stops ws macos cable before any -setmanual.
printf '%s\n' 'DHCP Configuration' 'IP address: 192.0.2.10' 'Router: 192.0.2.1' \
    >"$tmp/info-ws_stand_in_no_op"
stops 'ws macos cable: networksetup -getinfo "ws stand-in no-op" shows Router: 192.0.2.1, so the'
logged
exit "$fail"
