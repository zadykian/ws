# shellcheck shell=sh
# ws-tunnel's settings: the checks of their values, and their reading from tunnel.conf and
# routes.d, as it starts and again on SIGHUP. ws-tunnel sources it from root's copy beside it in
# /usr/local/libexec/ws-tunnel, and shares its variables with it.

# is_ipv4 ADDRESS succeeds where ADDRESS is an IPv4 address in four decimal parts, none with a
# leading zero, which the shell's arithmetic would read as octal.
is_ipv4() {
    case $1 in
    *[!0-9.]* | *..* | .* | *.) return 1 ;;
    esac
    rest=$1.
    parts=0
    while [ -n "$rest" ]; do
        part=${rest%%.*}
        rest=${rest#*.}
        parts=$((parts + 1))
        case $part in
        [0-9] | [1-9][0-9] | 1[0-9][0-9] | 2[0-4][0-9] | 25[0-5]) ;;
        *) return 1 ;;
        esac
    done
    [ "$parts" = 4 ]
}

# ipv4_int ADDRESS prints a valid IPv4 address as a number.
ipv4_int() {
    rest=$1.
    int=0
    while [ -n "$rest" ]; do
        int=$((int * 256 + ${rest%%.*}))
        rest=${rest#*.}
    done
    echo "$int"
}

# is_cidr CIDR succeeds where CIDR is an IPv4 network with no bits set past its length.
is_cidr() {
    case $1 in
    */[0-9] | */[12][0-9] | */3[0-2]) ;;
    *) return 1 ;;
    esac
    is_ipv4 "${1%/*}" || return 1
    [ $(($(ipv4_int "${1%/*}") & ((1 << (32 - ${1#*/})) - 1))) = 0 ]
}

# contains CIDR ADDRESS succeeds where the network CIDR holds ADDRESS.
contains() {
    shift_by=$((32 - ${1#*/}))
    [ $(($(ipv4_int "${1%/*}") >> shift_by)) = $(($(ipv4_int "$2") >> shift_by)) ]
}

# overlaps CIDR CIDR succeeds where the two networks share an address.
overlaps() {
    contains "$1" "${2%/*}" || contains "$2" "${1%/*}"
}

# is_name NAME succeeds where NAME is DNS labels joined by dots.
is_name() {
    [ -n "$1" ] && [ "${#1}" -le 253 ] || return 1
    rest=$1.
    while [ -n "$rest" ]; do
        part=${rest%%.*}
        rest=${rest#*.}
        case $part in
        '' | -* | *- | *[!A-Za-z0-9-]*) return 1 ;;
        esac
        [ "${#part}" -le 63 ] || return 1
    done
}

# is_port PORT succeeds where PORT is 1 to 65535.
is_port() {
    case $1 in
    '' | 0* | *[!0-9]* | ??????*) return 1 ;;
    esac
    [ "$1" -le 65535 ]
}

# is_key KEY succeeds where KEY is a WireGuard key in base64: 43 characters and an =.
is_key() {
    case $1 in
    *[!A-Za-z0-9+/=]* | *=?*) return 1 ;;
    esac
    [ "${#1}" = 44 ] && [ "${1%=}" != "$1" ]
}

# read_conf reads tunnel.conf, without running it, into new_key, new_lan, new_lan_server,
# new_remote and new_port, and fails on a line it doesn't take or without WS_PUBLIC_KEY.
# shellcheck disable=SC2154 # etc is ws-tunnel's
read_conf() {
    new_key=
    new_lan=
    new_lan_server=
    new_remote=
    new_port=51820
    if [ ! -r "$etc/tunnel.conf" ]; then
        say "cannot read $etc/tunnel.conf: run mac/setup.sh"
        return 1
    fi
    while IFS=' 	' read -r line || [ -n "$line" ]; do
        value=${line#*=}
        case $line in
        '' | '#'*) continue ;;
        WS_PUBLIC_KEY=*) is_key "$value" && new_key=$value ;;
        WS_LAN=*) is_cidr "$value" && new_lan=$value ;;
        WS_LAN_ADDRESS=*) is_ipv4 "$value" && new_lan_server=$value ;;
        WS_REMOTE=*) is_name "$value" && new_remote=$value ;;
        WS_PORT=*) is_port "$value" && new_port=$value ;;
        *) false ;;
        esac || {
            say "tunnel.conf: cannot take '$line'"
            return 1
        }
    done <"$etc/tunnel.conf"
    if [ -z "$new_key" ]; then
        say "tunnel.conf has no WS_PUBLIC_KEY"
        return 1
    fi
}

# read_routes reads routes.d into new_routes, each network once, and leaves out a line that isn't
# a network, or one that overlaps the tunnel's or the cable's subnet.
# shellcheck disable=SC2154 # the subnets are ws-tunnel's
read_routes() {
    new_routes=
    for file in "$etc"/routes.d/*; do
        [ -f "$file" ] || continue
        while IFS=' 	' read -r line || [ -n "$line" ]; do
            case $line in
            '' | '#'*) continue ;;
            esac
            if is_cidr "$line" && ! overlaps "$line" "$tunnel_net" &&
                ! overlaps "$line" "$cable_net"; then
                case " $new_routes " in
                *" $line "*) ;;
                *) new_routes="$new_routes $line" ;;
                esac
            else
                say "routes.d/${file##*/}: leaves out '$line', not a network it can route"
            fi
        done <"$file"
    done
    new_routes=${new_routes# }
}

# reload reads tunnel.conf and routes.d again, on SIGHUP. A new key or path runs the tiers;
# otherwise it changes the peer's AllowedIPs and the routes in place, which keeps the session.
# shellcheck disable=SC2034,SC2154 # the daemon's state, which ws-tunnel keeps
reload() {
    reload=0
    old="$key $lan $lan_server $remote $port"
    old_key=$key
    if read_conf; then
        key=$new_key
        lan=$new_lan
        lan_server=$new_lan_server
        remote=$new_remote
        port=$new_port
    else
        say "keeps the settings it had"
    fi
    read_routes
    routes=$new_routes
    if [ "$old" != "$key $lan $lan_server $remote $port" ]; then
        say "read new settings; trying the paths again"
        [ "$old_key" = "$key" ] || wg set "$tun" peer "$old_key" remove 2>/dev/null
        tiers
        return 0
    fi
    allowed_for "$peer"
    if [ -n "$peer" ]; then
        wg set "$tun" peer "$key" allowed-ips "$allowed" || say "cannot set the peer's AllowedIPs"
    fi
    apply_routes
    say "read its settings again; routes: ${routed:-none}"
    case $in_use in
    none) write_state none "" ;;
    *) write_state "${in_use%% *}" "$peer:$port" ;;
    esac
}
