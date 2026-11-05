#!/bin/sh
# The wizard's files script on ws, which writes what the wizard names there: the cable's port
# and the tunnel's peer. The Mac's wizard sends it over ssh, as it sends wizard-remote.sh:
#
#   ssh -o HostName=ADDRESS -o Port=PORT ws sh -s -- OP [ARG...] <mac/wizard-remote-files.sh
#
# - wait PORT prints what link prints, below, once PORT holds 10.77.0.1, or after 10 s.
# - link lists ws's wired ports but the uplinks, those that a default route names, a line each. A
#   line holds the port, its carrier, 1 or 0, or - where the port is down, and its IPv4 addresses,
#   or -. link PORT names that port in ws's /etc/ws/host_vars.yml as
#   network_direct_link_interface, the port of the Mac's cable, which the network and wireguard
#   roles read.
# - peer KEY writes KEY, the Mac's WireGuard key, to mac.pub where that holds another.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# Sysfs's network interfaces, the machine's values for ws server apply and the Mac's key for wg0.
# CI's stand-in for ws roots each, a line NAME=PATH, in a directory of its own.
net=/sys/class/net
vars=/etc/ws/host_vars.yml
peer=/etc/wireguard/mac.pub

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote-files.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# ports prints ws's wired ports: a device's, of ARPHRD_ETHER's type 1 and not wireless, but those
# that a default route names, in any table, as an uplink's. A bridge's or bond's member, which
# sysfs's master marks, is left out too, as an uplink may be one. So is a name of other characters
# than letters, digits and ._-, or with a - first, as no argument may hold them.
ports() {
    uplinks=$({
        ip route show default table all
        ip -6 route show default table all
    } 2>/dev/null | awk '{ for (i = 1; i < NF; i++) if ($i == "dev") print $(i + 1) }')
    for path in "$net"/*; do
        port=${path##*/}
        case $port in *[!A-Za-z0-9_.-]* | -*) continue ;; esac
        [ -e "$path/device" ] && [ ! -e "$path/master" ] &&
            [ "$(cat "$path/type" 2>/dev/null)" = 1 ] &&
            [ ! -e "$path/wireless" ] && [ ! -e "$path/phy80211" ] || continue
        printf '%s\n' "$uplinks" | grep -qxF -- "$port" || echo "$port"
    done
}

# link_wait PORT runs link once PORT holds 10.77.0.1, which networkd gives it a moment after the
# network role's networkctl reload, and later for a port that was down. It reads ip each second, 10
# s at most, so that the wizard waits in one login, which a key may take a touch for.
link_wait() {
    ports | grep -qxF -- "${1-}" || die "'${1-}' is none of the wired ports that link lists"
    n=0
    until ip -o -4 address show dev "$1" 2>/dev/null | grep -q ' 10\.77\.0\.1/' || [ "$n" = 10 ]; do
        sleep 1
        n=$((n + 1))
    done
    link
}

# name KEY VALUE names VALUE in vars as KEY, by a rename that keeps the file's other lines and its
# mode. A new file is 0600 in a 0755 directory, and starts with ---, as yamllint asks of YAML.
name() {
    [ -d "${vars%/*}" ] || { mkdir -p "${vars%/*}" && chmod 755 "${vars%/*}"; }
    umask 077
    [ ! -f "$vars" ] || cp -p "$vars" "$vars.new"
    {
        [ -s "$vars" ] || echo ---
        cat "$vars" 2>/dev/null | awk -v key="$1:" -v line="$1: \"$2\"" '
            index($0, key) == 1 { if (!done++) print line; next }
            { print }
            END { if (!done) print line }'
    } >"$vars.new"
    mv -f "$vars.new" "$vars"
    say "wrote $vars"
}

# link [PORT] lists the ports, as the header says, with sysfs's carrier, which a port that is down
# lacks, and ip's IPv4 addresses. Given PORT, one that it lists, it names it in vars, where that
# names another.
link() {
    if [ -z "${1-}" ]; then
        for port in $(ports); do
            carrier=$(cat "$net/$port/carrier" 2>/dev/null) || carrier=
            addresses=$(ip -o -4 address show dev "$port" 2>/dev/null | awk '{ printf " %s", $4 }')
            echo "$port ${carrier:--}${addresses:- -}"
        done
        return 0
    fi
    ports | grep -qxF -- "$1" || die "'$1' is none of the wired ports that link lists"
    [ "$(sed -n 's/^network_direct_link_interface: *"\{0,1\}\([A-Za-z0-9_.-]*\).*/\1/p' "$vars" \
        2>/dev/null)" != "$1" ] || return 0
    name network_direct_link_interface "$1"
}

# peer KEY writes KEY, a WireGuard public key, to mac.pub, which wg0's peer mac takes at the
# wireguard role's next run, where mac.pub holds another. Its directory is root's alone, as
# wireguard-tools makes it.
peer() {
    printf '%s\n' "$1" | grep -qxE '[A-Za-z0-9+/]{43}=' && [ "${#1}" = 44 ] ||
        die "'$1' is not a WireGuard public key"
    [ "$(head -n 1 "$peer" 2>/dev/null)" != "$1" ] || return 0
    (umask 077 && mkdir -p "${peer%/*}") || die "cannot make ${peer%/*}"
    printf '%s\n' "$1" >"$peer.new" || die "cannot write $peer.new"
    mv -f "$peer.new" "$peer" || die "cannot write $peer"
    say "wrote $peer"
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, through sudo or a root login"
    case ${1-} in
    wait) link_wait "${2-}" ;;
    link) link "${2-}" ;;
    peer) peer "${2-}" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote-files.sh wait PORT | link [PORT] |" \
        "peer KEY" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
