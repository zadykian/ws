#!/bin/sh
# ws's programs that the wizard's remote script runs, as stand-ins on GitHub's macOS runner, each by
# the name it is called by. standin.sh links the names in WS_STANDIN/bin, which ws.sh puts first on
# PATH, claude where the remote script looks for it, and bootstrap.sh in the stand-in clone. Their
# state is in WS_STANDIN, which ws's sshd sets:
#
# - units/NAME is a unit's log, as the journal holds it, and units/NAME.on is there while it runs;
# - run.log holds systemd-run's arguments, and bootstrap.log bootstrap.sh's, a line each;
# - bootstrap.wait holds the seconds that bootstrap.sh runs, and bootstrap.status its status;
# - with bootstrap.cut, bootstrap.sh kills the unit's job, as a stop or a reboot would;
# - wg0 holds wg0's key, sb turns Secure Boot on, nct6687 builds the fans' driver, and gh and
#   claude log each in;
# - gh.keys holds the signing keys of gh's account, which bootstrap.sh --tags devtools adds ws's to,
#   and gh.nosign gives their list a 404, as a login without the write:ssh_signing_key scope gets;
# - ws.key is wg0's key, which bootstrap.sh --tags wireguard makes, and utun names wg0, root's;
# - clone.log holds the repositories that gh repo clone cloned, each with this script as apply.sh
#   and its URL on GitHub as origin;
# - apply.log holds a line for each run of apply.sh: tty where its stdin is a terminal, else the
#   SHA-256 of what it read there, and apply.status its status;
# - uplink names the port of the default route, and addr/PORT holds a port's IPv4 addresses, which
#   bootstrap.sh --tags network,wireguard gives the port that host_vars names, with carrier. That
#   run writes the port in root's netplan file, as the network role does, or removes the file.

set -u

s=$WS_STANDIN

# wg0 sets up ws's wg0 as the wireguard role does: its key, made once, whose public key goes in
# wg0, and the Mac's key in mac.pub as its peer. wg0 is a wireguard-go of root's copy beside the
# Mac's, on UDP 51820, as mac.yml's stand-in ws. It has no address in the tunnel.
wg0() {
    bin=/usr/local/libexec/ws-tunnel
    [ -f "$s/ws.key" ] || (umask 077 && "$bin/wg" genkey >"$s/ws.key") || return 1
    "$bin/wg" pubkey <"$s/ws.key" >"$s/wg0" || return 1
    if [ ! -f "$s/utun" ] || ! "$bin/wg" show "$(cat "$s/utun")" >/dev/null 2>&1; then
        mkdir -p /var/run/wireguard
        WG_TUN_NAME_FILE=$s/utun "$bin/wireguard-go" utun || return 1
    fi
    "$bin/wg" set "$(cat "$s/utun")" private-key "$s/ws.key" listen-port 51820 \
        peer "$(cat "$s/root/etc/wireguard/mac.pub")" allowed-ips 10.99.0.2/32 || return 1
    ifconfig "$(cat "$s/utun")" up
}
# direct names the port that ws's host_vars names in the netplan file, as the network role does,
# or removes the file where host_vars names none. It gives the port ws's end of the cable,
# 10.77.0.1/30, once networkd sees carrier, which the cases' sysfs shows.
direct() {
    port=$(sed -n 's/^network_direct_link_interface: "\(.*\)"$/\1/p' \
        "$s/root/root/repository/ws/host_vars/localhost.yml" 2>/dev/null)
    netplan=$s/root/etc/netplan/60-direct-link.yaml
    rm -f "$netplan"
    [ -z "$port" ] || { mkdir -p "${netplan%/*}" && echo "    \"$port\":" >"$netplan"; } || return 1
    [ -n "$port" ] && [ "$(cat "$s/root/sys/class/net/$port/carrier" 2>/dev/null)" = 1 ] ||
        return 0
    mkdir -p "$s/addr"
    grep -qsx 10.77.0.1/30 "$s/addr/$port" || echo 10.77.0.1/30 >>"$s/addr/$port"
}
case ${0##*/} in
systemd-run)
    echo "$*" >>"$s/run.log"
    while [ "${1#--}" != "$1" ]; do
        case $1 in --unit=*) unit=${1#--unit=} ;; esac
        shift
    done
    touch "$s/units/$unit.on"
    # The unit goes on where ssh drops, as systemd's does.
    (
        trap '' HUP
        "$@" >>"$s/units/$unit" 2>&1
        rm -f "$s/units/$unit.on"
    ) </dev/null >/dev/null 2>&1 &
    ;;
systemctl)
    case $1 in
    list-units)
        for on in "$s"/units/*.on; do
            [ ! -e "$on" ] || echo "$(basename "$on" .on).service loaded active running stand-in"
        done
        ;;
    show)
        eval "unit=\${$#}"
        if [ -e "$s/units/$unit.on" ]; then echo active; else echo inactive; fi
        ;;
    *) exit 2 ;;
    esac
    ;;
journalctl)
    # The lines of the unit's log after those that the cursor's file counts.
    while [ $# -gt 0 ]; do
        case $1 in
        -u) unit=$2 && shift ;;
        --cursor-file=*) cursor=${1#--cursor-file=} ;;
        esac
        shift
    done
    [ -f "$s/units/$unit" ] || exit 0
    at=$(cat "$cursor" 2>/dev/null || echo 0)
    lines=$(wc -l <"$s/units/$unit")
    [ "$lines" -gt "$at" ] || exit 0
    sed -n "$((at + 1)),${lines}p" "$s/units/$unit"
    echo "$lines" >"$cursor"
    ;;
bootstrap.sh)
    echo "$*" >>"$s/bootstrap.log"
    echo "bootstrap.sh's stand-in, with '$*'"
    sleep "$(cat "$s/bootstrap.wait" 2>/dev/null || echo 0)"
    if [ -f "$s/bootstrap.cut" ]; then
        kill -9 "$PPID"
        exit 1
    fi
    # The devtools role adds ws's signing key to the account, where gh's login has the scope.
    [ "$*" != '--tags devtools' ] || [ ! -f "$s/gh" ] || [ -f "$s/gh.nosign" ] ||
        cut -d ' ' -f 1-2 "$s/signing.pub" >>"$s/gh.keys"
    [ "$*" != '--tags wireguard' ] || wg0 || exit 1
    [ "$*" != '--tags network,wireguard' ] || direct
    echo 'PLAY RECAP'
    exit "$(cat "$s/bootstrap.status" 2>/dev/null || echo 0)"
    ;;
wg)
    # wg0's key, and with show wg0 peers the peers of wg0's wireguard-go.
    [ "$*" = 'show wg0 peers' ] || exec cat "$s/wg0" 2>/dev/null
    /usr/local/libexec/ws-tunnel/wg show "$(cat "$s/utun")" peers
    ;;
gh)
    # The calls of the wizard's scripts on ws, then the device flow's login and refresh, which the
    # wizard runs in ssh -t: each fails without a terminal.
    case $* in
    'auth status -h github.com') [ -f "$s/gh" ] ;;
    'api --paginate user/ssh_signing_keys --jq .[].key')
        [ -f "$s/gh" ] || exit 4
        [ ! -f "$s/gh.nosign" ] || { echo 'gh: Not Found (HTTP 404)' >&2 && exit 1; }
        cat "$s/gh.keys"
        ;;
    'auth login -h github.com -p https -w -s write:ssh_signing_key')
        [ -t 0 ] || exit 1
        echo '! First copy your one-time code: 0000-0000'
        touch "$s/gh" && rm -f "$s/gh.nosign"
        ;;
    'auth refresh -h github.com -s write:ssh_signing_key') [ -t 0 ] && rm -f "$s/gh.nosign" ;;
    'repo clone '*' -- -q')
        [ -f "$s/gh" ] || exit 4
        echo "$3" >>"$s/clone.log"
        git init -q -b main "$4" && ln -s "$(readlink "$0")" "$4/apply.sh" &&
            git -C "$4" remote add origin "https://github.com/$3.git" && git -C "$4" add apply.sh &&
            git -C "$4" -c user.name=ws -c user.email=ws@example.com commit -q -m "$3"
        ;;
    *) exit 2 ;;
    esac
    ;;
claude)
    case $* in
    'auth status') [ -f "$s/claude" ] ;;
    'auth login') [ -t 0 ] && touch "$s/claude" ;;
    *) exit 2 ;;
    esac
    ;;
apply.sh)
    if [ -t 0 ]; then echo tty; else shasum -a 256 | cut -d ' ' -f 1; fi >>"$s/apply.log"
    exit "$(cat "$s/apply.status" 2>/dev/null || echo 0)"
    ;;
ip)
    # The default routes, by the port that uplink names, and the IPv4 addresses in addr/PORT.
    case $* in
    *'route show default table all')
        [ ! -f "$s/uplink" ] || echo "default via 192.0.2.1 dev $(cat "$s/uplink") metric 100"
        ;;
    "-o -4 address show dev ${6-}")
        [ ! -f "$s/addr/$6" ] || sed "s|.*|2: $6    inet & scope global $6|" "$s/addr/$6"
        ;;
    *) exit 2 ;;
    esac
    ;;
mokutil) if [ -f "$s/sb" ]; then echo 'SecureBoot enabled'; else echo 'SecureBoot disabled'; fi ;;
modinfo) [ -f "$s/nct6687" ] ;;
*) exit 2 ;;
esac
