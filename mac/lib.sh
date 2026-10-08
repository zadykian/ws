# shellcheck shell=sh
# The functions of ws's scripts for the Mac, sourced by mac/setup.sh, and by a private
# repository's own script for the Mac, which may add routes into the tunnel and resolvers for
# domains behind it (ws_tunnel_routes, ws_resolver and ws_tunnel_loaded, below). Sourcing it
# defines functions and variables whose names start with ws_, and changes nothing else.
# Every function that writes reads WS_CHECK: set to anything but empty or 0, as ws_args sets it
# for --check, it prints each change as a diff, or a line where there is no text to compare, and
# makes none. A function exits with status 1 on an error, with a message on stderr. Files outside
# the user's home are written through sudo, owned by root.

ws_prog=${0##*/}
# The subnets of the cable and of the tunnel, which the home LAN and the routes stay clear of.
ws_cable_net=10.77.0.0/30
ws_tunnel_net=10.99.0.0/24
# The tunnel's daemon: its launchd label and settings, and macOS's directory of resolvers.
ws_label=com.github.zadykian.ws-tunnel
ws_etc=/etc/ws-tunnel
ws_resolvers=/etc/resolver
# The first line of each file ws_resolver writes, and the only kind it removes.
ws_resolver_header="# Written by ws's mac/lib.sh: ws_resolver removes it when given no address."
# ws_changed becomes 1 once a function changes something, or would under WS_CHECK. Nothing sets
# it back to 0 but the script that sources lib.sh, before the steps it wants to know about.
ws_changed=0

# ws_say prints a message of the script's on stderr.
ws_say() {
    printf '%s: %s\n' "$ws_prog" "$*" >&2
}

# ws_die prints a message of the script's and exits with status 1.
ws_die() {
    ws_say "$*"
    exit 1
}

# ws_checking succeeds under WS_CHECK.
ws_checking() {
    [ -n "${WS_CHECK-}" ] && [ "$WS_CHECK" != 0 ]
}

# ws_is_ipv4 ADDRESS succeeds where ADDRESS is an IPv4 address in four decimal parts, none with a
# leading zero, which the shell's arithmetic would read as octal.
ws_is_ipv4() {
    case $1 in
    *[!0-9.]* | *..* | .* | *.) return 1 ;;
    esac
    ws_rest=$1.
    ws_parts=0
    while [ -n "$ws_rest" ]; do
        ws_part=${ws_rest%%.*}
        ws_rest=${ws_rest#*.}
        ws_parts=$((ws_parts + 1))
        case $ws_part in
        [0-9] | [1-9][0-9] | 1[0-9][0-9] | 2[0-4][0-9] | 25[0-5]) ;;
        *) return 1 ;;
        esac
    done
    [ "$ws_parts" = 4 ]
}

# ws_ipv4_int ADDRESS prints a valid IPv4 address as a number.
ws_ipv4_int() {
    ws_rest=$1.
    ws_int=0
    while [ -n "$ws_rest" ]; do
        ws_int=$((ws_int * 256 + ${ws_rest%%.*}))
        ws_rest=${ws_rest#*.}
    done
    echo "$ws_int"
}

# ws_is_cidr CIDR succeeds where CIDR is an IPv4 network, ADDRESS/LENGTH, with no bits set past
# its length: 10.88.0.0/24, not 10.88.0.10/24.
ws_is_cidr() {
    case $1 in
    */[0-9] | */[12][0-9] | */3[0-2]) ;;
    *) return 1 ;;
    esac
    ws_is_ipv4 "${1%/*}" || return 1
    ws_int=$(ws_ipv4_int "${1%/*}")
    [ $((ws_int & ((1 << (32 - ${1#*/})) - 1))) = 0 ]
}

# ws_cidr_contains CIDR ADDRESS succeeds where the network CIDR holds ADDRESS.
ws_cidr_contains() {
    ws_shift=$((32 - ${1#*/}))
    [ $(($(ws_ipv4_int "${1%/*}") >> ws_shift)) = $(($(ws_ipv4_int "$2") >> ws_shift)) ]
}

# ws_cidr_overlaps CIDR CIDR succeeds where the two networks share an address: one holds the
# other's first address. 0.0.0.0/0 overlaps every network.
ws_cidr_overlaps() {
    ws_cidr_contains "$1" "${2%/*}" || ws_cidr_contains "$2" "${1%/*}"
}

# ws_is_domain NAME succeeds where NAME is one or more DNS labels joined by dots: each of 1 to 63
# letters, digits and hyphens, with no hyphen first or last, and 253 characters in all.
ws_is_domain() {
    [ -n "$1" ] && [ "${#1}" -le 253 ] || return 1
    ws_rest=$1.
    while [ -n "$ws_rest" ]; do
        ws_part=${ws_rest%%.*}
        ws_rest=${ws_rest#*.}
        case $ws_part in
        '' | -* | *- | *[!A-Za-z0-9-]*) return 1 ;;
        esac
        [ "${#ws_part}" -le 63 ] || return 1
    done
}

# ws_tmp makes an empty file of the user's own for a function's text, and sets ws_tmpfile to it.
ws_tmp() {
    ws_tmpfile=$(mktemp "${TMPDIR:-/tmp}/ws.XXXXXXXX") || ws_die "cannot make a temporary file"
}

# ws_diff OLD NEW NAME prints the diff from OLD to NEW, each labelled NAME, or /dev/null for a
# file that is missing.
ws_diff() {
    ws_from=$3
    [ "$1" != /dev/null ] || ws_from=/dev/null
    ws_to=$3
    [ "$2" != /dev/null ] || ws_to=/dev/null
    diff -u -L "$ws_from" -L "$ws_to" "$1" "$2" || true
}

# ws_root_attrs FILE prints the owner, group and mode of a file outside the home, read through
# sudo, as 0:0 644, or nothing where FILE is missing.
ws_root_attrs() {
    ws_a=$(sudo stat -c '%u:%g %a' "$1" 2>/dev/null || sudo stat -f '%u:%g %Lp' "$1" 2>/dev/null) ||
        return 0
    [ -n "$ws_a" ] || return 0
    ws_m=00${ws_a#* }
    echo "${ws_a%% *} ${ws_m#"${ws_m%???}"}"
}

# ws_root_fix FILE MODE ATTRS gives root's FILE, whose text is right, root as its owner and group,
# and MODE, where ATTRS, its owner, group and mode now, differ.
ws_root_fix() {
    [ "$3" != "0:0 $2" ] || return 0
    ws_changed=1
    if ws_checking; then
        ws_say "would make $1 root's, with mode $2, rather than $3"
    elif ! sudo chown 0:0 "$1" || ! sudo chmod "$2" "$1"; then
        ws_die "cannot set $1's owner and mode"
    fi
}

# ws_dir_root DIR MODE makes the directory DIR, root's, with MODE, where it is missing, and gives
# it root and MODE where they differ.
ws_dir_root() {
    ws_attrs=$(ws_root_attrs "$1")
    if [ -n "$ws_attrs" ]; then
        ws_root_fix "$1" "$2" "$ws_attrs"
        return 0
    fi
    ws_changed=1
    if ws_checking; then
        ws_say "would make $1, root's, with mode $2"
    elif ! sudo mkdir -m "$2" "$1" || ! sudo chown 0:0 "$1"; then
        ws_die "cannot make $1"
    fi
}

# ws_put_root FILE MODE writes stdin to root's FILE with MODE where its text, owner or mode differs,
# through a new file renamed over it. WS_CHECK's diff reads the old file by sudo, into the user's.
# shellcheck disable=SC2024 # the redirections are the user's, around sudo, on purpose
ws_put_root() {
    ws_tmp
    ws_new=$ws_tmpfile
    cat >"$ws_new" || ws_die "cannot write $ws_new"
    ws_attrs=$(ws_root_attrs "$1")
    ws_old=/dev/null
    if [ -n "$ws_attrs" ]; then
        ws_tmp
        ws_old=$ws_tmpfile
        sudo cat "$1" >"$ws_old" || ws_die "cannot read $1"
        if cmp -s "$ws_old" "$ws_new"; then
            rm -f "$ws_old" "$ws_new"
            ws_root_fix "$1" "$2" "$ws_attrs"
            return 0
        fi
    fi
    ws_changed=1
    if ws_checking; then
        ws_diff "$ws_old" "$ws_new" "$1"
    elif sudo sh -c 'cat >"$1" && chown 0:0 "$1" && chmod "$2" "$1" && mv -f "$1" "$3"' \
        sh "${1%/*}/.${1##*/}.ws-new" "$2" "$1" <"$ws_new"; then
        ws_say "wrote $1"
    else
        sudo rm -f "${1%/*}/.${1##*/}.ws-new"
        ws_die "cannot write $1"
    fi
    [ "$ws_old" = /dev/null ] || rm -f "$ws_old"
    rm -f "$ws_new"
}

# ws_rm_root FILE removes root's FILE where it is there.
# shellcheck disable=SC2024 # the redirection is the user's, around sudo, as in ws_put_root
ws_rm_root() {
    [ -n "$(ws_root_attrs "$1")" ] || return 0
    ws_changed=1
    if ws_checking; then
        ws_tmp
        ws_old=$ws_tmpfile
        sudo cat "$1" >"$ws_old" || ws_die "cannot read $1"
        ws_diff "$ws_old" /dev/null "$1"
        rm -f "$ws_old"
    else
        sudo rm -f "$1" || ws_die "cannot remove $1"
        ws_say "removed $1"
    fi
}

# ws_tunnel_loaded succeeds where launchd has the tunnel's daemon loaded, as once mac/setup.sh has
# run with WS_PUBLIC_KEY. Anyone may read launchd's system domain, so it needs no sudo.
ws_tunnel_loaded() {
    launchctl print "system/$ws_label" >/dev/null 2>&1
}

# ws_tunnel_routes NAME [CIDR...] has the tunnel take the IPv4 networks CIDR too: it writes them,
# a network a line, to /etc/ws-tunnel/routes.d/NAME, and has a loaded daemon read routes.d again
# (SIGHUP). The daemon adds them to the peer's AllowedIPs and routes them into its utun, keeping
# open connections. NAME is of a-z, 0-9 and -. A network must not overlap the tunnel's
# 10.99.0.0/24 or the cable's 10.77.0.0/30, so 0.0.0.0/0 is refused too. With no CIDR, it
# removes the file, and where routes.d is missing, as once the tunnel is gone, does nothing. The
# file outlives the daemon, but its routes go with the daemon's utun.
ws_tunnel_routes() {
    [ $# -ge 1 ] || ws_die "usage: ws_tunnel_routes NAME [CIDR...]"
    ws_name=$1
    shift
    case $ws_name in
    '' | *[!a-z0-9-]*) ws_die "ws_tunnel_routes: '$ws_name' is not a name of a-z, 0-9 and -" ;;
    esac
    for ws_cidr; do
        ws_is_cidr "$ws_cidr" ||
            ws_die "ws_tunnel_routes $ws_name: '$ws_cidr' is not an IPv4 network, as 192.0.2.0/24"
        ! ws_cidr_overlaps "$ws_cidr" "$ws_tunnel_net" ||
            ws_die "ws_tunnel_routes $ws_name: $ws_cidr overlaps the tunnel's $ws_tunnel_net"
        ! ws_cidr_overlaps "$ws_cidr" "$ws_cable_net" ||
            ws_die "ws_tunnel_routes $ws_name: $ws_cidr overlaps the cable's $ws_cable_net"
    done
    if [ -z "$(ws_root_attrs "$ws_etc/routes.d")" ]; then
        [ $# != 0 ] || return 0
        ws_die "$ws_etc/routes.d is missing: run ws's mac/setup.sh first"
    fi
    ws_was=$ws_changed
    ws_changed=0
    if [ $# = 0 ]; then
        ws_rm_root "$ws_etc/routes.d/$ws_name"
    else
        ws_text=$(printf '%s\n' "$@")
        ws_put_root "$ws_etc/routes.d/$ws_name" 644 <<EOF
# Written by ws's mac/lib.sh: ws_tunnel_routes $ws_name.
$ws_text
EOF
    fi
    if [ "$ws_changed" = 1 ] && ! ws_checking && ws_tunnel_loaded; then
        sudo launchctl kill SIGHUP "system/$ws_label" >/dev/null 2>&1 ||
            ws_say "warning: cannot signal $ws_label, which reads routes.d when it next starts"
    fi
    [ "$ws_was" = 0 ] || ws_changed=1
}

# ws_resolver DOMAIN [ADDRESS...] has macOS ask the nameservers ADDRESS, up to 3, for DOMAIN and
# the names under it, and the Mac's own for the rest: it writes /etc/resolver/DOMAIN, with
# ws_resolver_header first and a nameserver line for each ADDRESS (resolver(5)). configd watches
# /etc/resolver and takes a change at once. DOMAIN is DNS labels joined by dots. With no ADDRESS,
# it removes the file, but only one that starts with the header; it stops rather than replace
# such a file of another's.
ws_resolver() {
    [ $# -ge 1 ] || ws_die "usage: ws_resolver DOMAIN [ADDRESS...]"
    ws_domain=$1
    shift
    ws_is_domain "$ws_domain" ||
        ws_die "ws_resolver: '$ws_domain' is not a domain of DNS labels joined by dots"
    for ws_address; do
        ws_is_ipv4 "$ws_address" ||
            ws_die "ws_resolver $ws_domain: '$ws_address' is not an IPv4 address"
    done
    [ $# -le 3 ] || ws_die "ws_resolver $ws_domain: macOS takes 3 nameservers for a domain, not $#"
    ws_file=$ws_resolvers/$ws_domain
    if [ -n "$(ws_root_attrs "$ws_file")" ] &&
        [ "$(sudo head -n 1 "$ws_file")" != "$ws_resolver_header" ]; then
        if [ $# = 0 ]; then
            ws_say "leaves $ws_file, which ws_resolver didn't write"
            return 0
        fi
        ws_die "$ws_file is there, and ws_resolver didn't write it: move it away first"
    fi
    if [ $# = 0 ]; then
        ws_rm_root "$ws_file"
        return 0
    fi
    [ -n "$(ws_root_attrs "$ws_resolvers")" ] || ws_dir_root "$ws_resolvers" 755
    ws_text=$(printf 'nameserver %s\n' "$@")
    ws_put_root "$ws_file" 644 <<EOF
$ws_resolver_header
$ws_text
EOF
}
