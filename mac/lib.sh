# shellcheck shell=sh
# The functions of ws's scripts for the Mac, sourced by mac/setup.sh, and by a private
# repository's own script for the Mac, which may add routes into the tunnel and resolvers for
# domains behind it (ws_tunnel_routes, ws_resolver and ws_tunnel_loaded, below). Sourcing it
# defines functions and variables whose names start with ws_, and changes nothing else.
#
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
    case ${WS_CHECK-} in
    '' | 0) return 1 ;;
    esac
}

# ws_args takes a script's arguments: --check sets WS_CHECK.
ws_args() {
    for ws_arg; do
        case $ws_arg in
        --check) WS_CHECK=1 ;;
        -h | --help)
            printf 'usage: %s [--check]\n' "$ws_prog"
            exit 0
            ;;
        *) ws_die "unknown argument '$ws_arg'; usage: $ws_prog [--check]" ;;
        esac
    done
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

# ws_is_port PORT succeeds where PORT is a port number, 1 to 65535.
ws_is_port() {
    case $1 in
    '' | 0* | *[!0-9]* | ??????*) return 1 ;;
    esac
    [ "$1" -le 65535 ]
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

# ws_is_key KEY succeeds where KEY is a WireGuard public key: 43 base64 characters and an =.
ws_is_key() {
    case $1 in
    *[!A-Za-z0-9+/=]* | *=?*) return 1 ;;
    esac
    [ "${#1}" = 44 ] && [ "${1%=}" != "$1" ]
}

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, without running it: a
# KEY=value a line, the value bare or in a pair of quotes, blank lines and lines starting with #
# skipped. It stops on any other line, any other key, and a value of the wrong form.
ws_conf() {
    [ -f "$1" ] || ws_die "$1 is missing: README.md's The Mac says what goes in it"
    WS_PUBLIC_KEY=
    WS_LAN=
    WS_LAN_ADDRESS=
    WS_REMOTE=
    WS_REMOTE_PORT=
    WS_PORT=
    ws_n=0
    while IFS=' 	' read -r ws_line || [ -n "$ws_line" ]; do
        ws_n=$((ws_n + 1))
        case $ws_line in
        '' | '#'*) continue ;;
        [A-Z]*=*) ;;
        *) ws_die "$1, line $ws_n: not a KEY=value line" ;;
        esac
        ws_key=${ws_line%%=*}
        ws_value=${ws_line#*=}
        case $ws_value in
        \"*\" | \'*\')
            ws_value=${ws_value#?}
            ws_value=${ws_value%?}
            ;;
        esac
        ws_ok=true
        case $ws_key in
        WS_PUBLIC_KEY)
            # shellcheck disable=SC2034 # for setup.sh, as WS_PORT
            WS_PUBLIC_KEY=$ws_value
            ws_is_key "$ws_value" || ws_ok=false
            ;;
        WS_LAN)
            WS_LAN=$ws_value
            ws_is_cidr "$ws_value" && ! ws_cidr_overlaps "$ws_value" "$ws_cable_net" &&
                ! ws_cidr_overlaps "$ws_value" "$ws_tunnel_net" || ws_ok=false
            ;;
        WS_LAN_ADDRESS)
            WS_LAN_ADDRESS=$ws_value
            ws_is_ipv4 "$ws_value" || ws_ok=false
            ;;
        WS_REMOTE)
            WS_REMOTE=$ws_value
            ws_is_domain "$ws_value" || ws_ok=false
            ;;
        WS_REMOTE_PORT)
            WS_REMOTE_PORT=$ws_value
            ws_is_port "$ws_value" || ws_ok=false
            ;;
        WS_PORT)
            # shellcheck disable=SC2034
            WS_PORT=$ws_value
            ws_is_port "$ws_value" || ws_ok=false
            ;;
        *) ws_die "$1, line $ws_n: unknown key $ws_key" ;;
        esac
        $ws_ok || ws_die "$1, line $ws_n: $ws_key=$ws_value is not $(ws_conf_form "$ws_key")"
    done <"$1"
    case ${WS_LAN:+lan}${WS_LAN_ADDRESS:+address} in
    lan | address) ws_die "$1: WS_LAN and WS_LAN_ADDRESS go together" ;;
    esac
    if [ -n "$WS_REMOTE_PORT" ] && [ -z "$WS_REMOTE" ]; then
        ws_die "$1: WS_REMOTE_PORT needs WS_REMOTE"
    fi
}

# ws_conf_form KEY prints the form a value of KEY takes, for ws_conf's message.
ws_conf_form() {
    case $1 in
    WS_PUBLIC_KEY) echo "a WireGuard public key, 43 base64 characters and an =" ;;
    WS_LAN) echo "an IPv4 network, as 10.88.0.0/24, clear of $ws_cable_net and $ws_tunnel_net" ;;
    WS_LAN_ADDRESS) echo "an IPv4 address" ;;
    WS_REMOTE) echo "a host name or an IPv4 address" ;;
    *) echo "a port, 1 to 65535" ;;
    esac
}

# ws_mode FILE prints FILE's permissions in three octal digits, as 600, with GNU's stat or the BSD
# stat of macOS, leaving out the setuid, setgid and sticky bits.
ws_mode() {
    ws_m=$(stat -c %a "$1" 2>/dev/null || stat -f %Lp "$1") || return 1
    ws_m=00$ws_m
    echo "${ws_m#"${ws_m%???}"}"
}

# ws_tmp makes an empty file of the user's own for a function's text, and sets ws_tmpfile to its
# name.
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

# ws_dir DIR MODE makes the user's directory DIR with MODE where it is missing, and sets MODE
# where it differs.
ws_dir() {
    if [ ! -d "$1" ]; then
        ws_changed=1
        if ws_checking; then
            ws_say "would make $1, mode $2"
        else
            mkdir -m "$2" "$1" || ws_die "cannot make $1"
        fi
    elif [ "$(ws_mode "$1")" != "$2" ]; then
        ws_changed=1
        if ws_checking; then
            ws_say "would set $1's mode to $2"
        else
            chmod "$2" "$1" || ws_die "cannot set $1's mode"
        fi
    fi
}

# ws_put FILE MODE writes stdin to the user's FILE with MODE where its text or mode differs,
# through a new file renamed over it.
ws_put() {
    ws_tmp
    ws_new=$ws_tmpfile
    cat >"$ws_new" || ws_die "cannot write $ws_new"
    if [ -f "$1" ] && cmp -s "$1" "$ws_new"; then
        rm -f "$ws_new"
        if [ "$(ws_mode "$1")" != "$2" ]; then
            ws_changed=1
            if ws_checking; then
                ws_say "would set $1's mode to $2"
            else
                chmod "$2" "$1" || ws_die "cannot set $1's mode"
            fi
        fi
        return 0
    fi
    ws_changed=1
    if ws_checking; then
        if [ -f "$1" ]; then
            ws_diff "$1" "$ws_new" "$1"
        else
            ws_diff /dev/null "$ws_new" "$1"
        fi
        rm -f "$ws_new"
        return 0
    fi
    ws_next="${1%/*}/.${1##*/}.ws-new"
    if cp "$ws_new" "$ws_next" && chmod "$2" "$ws_next" && mv -f "$ws_next" "$1"; then
        rm -f "$ws_new"
        ws_say "wrote $1"
    else
        rm -f "$ws_new" "$ws_next"
        ws_die "cannot write $1"
    fi
}

# ws_include_first FILE LINE puts LINE, an Include, above every setting of the ssh_config FILE,
# as ssh keeps the first value it reads for a key. It makes FILE where it is missing, and adds
# LINE at the top where FILE lacks it, keeping the old file as FILE.TIME~. It stops where LINE
# sits below a setting, and where FILE is a link, which it would replace with a file.
ws_include_first() {
    [ ! -L "$1" ] || ws_die "$1 is a link: put '$2' at the top of the file it points to"
    if [ ! -e "$1" ]; then
        ws_put "$1" 600 <<EOF
$2
EOF
        return 0
    fi
    ws_at=$(awk -v want="$2" '
        {
            line = $0
            sub(/^[ \t]+/, "", line)
            sub(/[ \t\r]+$/, "", line)
            gsub(/[ \t]+/, " ", line)
        }
        line == "" || line ~ /^#/ { next }
        tolower(line) == tolower(want) { print (++settings == 1 ? "first" : NR); exit }
        { settings++ }
    ' "$1") || ws_die "cannot read $1"
    case $ws_at in
    first) return 0 ;;
    '') ;;
    *)
        ws_die "line $ws_at of $1 is '$2', below other settings, which ssh would take first:" \
            "move it to the top"
        ;;
    esac
    if ! ws_checking; then
        ws_backup=$1.$(date +%Y%m%d%H%M%S)~
        cp -p "$1" "$ws_backup" || ws_die "cannot keep $1 as $ws_backup"
        ws_say "kept the old $1 as $ws_backup"
    fi
    ws_tmp
    ws_top=$ws_tmpfile
    { printf '%s\n\n' "$2" && cat "$1"; } >"$ws_top" || ws_die "cannot write $ws_top"
    ws_put "$1" "$(ws_mode "$1")" <"$ws_top"
    rm -f "$ws_top"
}

# ws_own_hosts FILE NAME... warns of each line of the ssh_config FILE that starts a Host or Match
# block for one of the NAMEs, whose settings come after those config.d's first line gives.
ws_own_hosts() {
    [ -f "$1" ] || return 0
    ws_file=$1
    shift
    awk -v names=" $* " '
        {
            key = tolower($1)
            if (key != "host" && key != "match") next
            for (i = 2; i <= NF; i++) {
                if (index(names, " " $i " ")) { print NR ": " $0; next }
            }
        }
    ' "$ws_file" | while IFS= read -r ws_hit; do
        ws_say "warning: line ${ws_hit%%:*} of $ws_file,${ws_hit#*:}, sets its own values for" \
            "ws: those in config.d, read first, win where both set a key"
    done
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
    else
        if ! sudo chown 0:0 "$1" || ! sudo chmod "$2" "$1"; then
            ws_die "cannot set $1's owner and mode"
        fi
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
    else
        if ! sudo mkdir -m "$2" "$1" || ! sudo chown 0:0 "$1"; then
            ws_die "cannot make $1"
        fi
    fi
}

# ws_put_root FILE MODE writes stdin to FILE, root's, with MODE, where its text, owner or mode
# differs, through a new file renamed over it. The diff under WS_CHECK reads the old file through
# sudo, into a file of the user's.
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

# ws_copy_root FROM FILE MODE copies FROM, a program, to FILE, root's, with MODE, where it differs.
ws_copy_root() {
    ws_attrs=$(ws_root_attrs "$2")
    if [ -n "$ws_attrs" ] && sudo cmp -s "$1" "$2"; then
        ws_root_fix "$2" "$3" "$ws_attrs"
        return 0
    fi
    ws_changed=1
    if ws_checking; then
        ws_say "would copy $1 to $2"
    elif sudo sh -c 'cp "$1" "$2" && chown 0:0 "$2" && chmod "$3" "$2" && mv -f "$2" "$4"' \
        sh "$1" "${2%/*}/.${2##*/}.ws-new" "$3" "$2"; then
        ws_say "copied $1 to $2"
    else
        sudo rm -f "${2%/*}/.${2##*/}.ws-new"
        ws_die "cannot copy $1 to $2"
    fi
}

# ws_rm_root FILE removes root's FILE where it is there.
# shellcheck disable=SC2024
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
        if ws_cidr_overlaps "$ws_cidr" "$ws_tunnel_net"; then
            ws_die "ws_tunnel_routes $ws_name: $ws_cidr overlaps the tunnel's $ws_tunnel_net"
        fi
        if ws_cidr_overlaps "$ws_cidr" "$ws_cable_net"; then
            ws_die "ws_tunnel_routes $ws_name: $ws_cidr overlaps the cable's $ws_cable_net"
        fi
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
