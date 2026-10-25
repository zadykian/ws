#!/bin/sh
# Written by github.com/zadykian/ws, whose next run replaces any change made here.
#
# Fills the outbound table's sets of resolved's DNS servers from the nameserver lines of the
# resolv.conf that resolved rewrites as they change. Who runs it, and when, is said here alone.
# security-outbound.service runs it with the table's FILE, which it loads with the sets' elements in
# one nft transaction, or alone where nft refuses them. A missing resolv.conf leaves the sets empty.

set -eu

conf=/run/systemd/resolve/resolv.conf
cr=$(printf '\r')

# ipv4 ADDR succeeds where ADDR is four numbers to 255 joined by dots, none with a leading zero.
# nft reads other forms loosely, as 010 for 8, or looks them up as names.
ipv4() (
    addr=$1
    IFS=.
    set -f
    # shellcheck disable=SC2086 # Split at the dots, on purpose.
    set -- $1
    [ $# -eq 4 ] && [ "$*" = "$addr" ] || exit 1
    for n; do
        case $n in
        [0-9] | [1-9][0-9] | 1[0-9][0-9] | 2[0-4][0-9] | 25[0-5]) ;;
        *) exit 1 ;;
        esac
    done
)

# ipv6 ADDR succeeds where ADDR is eight groups of 1 to 4 hex digits joined by colons, or fewer
# around one ::.
ipv6() (
    case $1 in
    *[!0-9A-Fa-f:]* | *:::* | *::*::* | :[!:]* | *[!:]:) exit 1 ;;
    *::*) min=0 max=7 ;;
    *) min=8 max=8 ;;
    esac
    IFS=:
    # shellcheck disable=SC2086 # Split at the colons, on purpose.
    set -- $1
    groups=0
    for group; do
        [ ${#group} -le 4 ] || exit 1
        [ -z "$group" ] || groups=$((groups + 1))
    done
    [ "$groups" -ge "$min" ] && [ "$groups" -le "$max" ]
)

# elements prints the command that adds each server of the resolv.conf on stdin to its set, and
# names those it leaves out: no address, or one in 0.0.0.0/8, an IPv6 one of zeros, broadcast or
# multicast. It skips a bare nameserver line. An IPv4-mapped server's packets are IPv4. A set holds
# no zone, so fe80::1%2 matches on any link.
elements() {
    while read -r key addr _; do
        addr=${addr%"$cr"}
        [ "$key" = nameserver ] && [ -n "$addr" ] || continue
        addr=${addr%%\%*}
        case $addr in
        ::ffff:*.* | ::FFFF:*.*) addr=${addr#::????:} ;;
        esac
        if ipv4 "$addr"; then
            case $addr in
            0.* | 255.255.255.255 | 22[4-9].* | 23[0-9].*) ;;
            *) echo "add element inet ws resolvers4 { $addr }" && continue ;;
            esac
        elif ipv6 "$addr"; then
            case $addr in
            [fF][fF][0-9A-Fa-f][0-9A-Fa-f]:*) ;;
            *[!0:]*) echo "add element inet ws resolvers6 { $addr }" && continue ;;
            esac
        fi
        printf 'resolvers.sh: left out %s\n' "$addr" >&2
    done
}

now=$(cat "$conf" 2>/dev/null) || now=
if [ $# -gt 0 ]; then
    table=$(cat "$1")
    add=$(printf '%s\n' "$now" | elements)
    printf '%s\n' "$table" "$add" | nft -f - 2>/dev/null && exit 0
    echo "resolvers.sh: nft refused $1 with the sets' elements, so it loads $1 alone" >&2
    exec nft -f "$1"
fi
