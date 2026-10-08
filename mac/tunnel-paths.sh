# shellcheck shell=sh
# ws-tunnel's paths to ws and its events: the Mac's addresses, the tiers and their handshakes, the
# peer's AllowedIPs and the routes, the ticks and network changes, and the daemon's end. ws-tunnel
# sources it from root's copy beside it in /usr/local/libexec/ws-tunnel, and shares its variables
# with it.

# resolve NAME prints NAME's first IPv4 address from the system's resolver, as wg would take
# getaddrinfo's first answer, which may be IPv6, and the router forwards IPv4 alone.
resolve() {
    if is_ipv4 "$1"; then
        echo "$1"
        return 0
    fi
    found=$(dscacheutil -q host -a name "$1" 2>/dev/null |
        awk '$1 == "ip_address:" { print $2; exit }')
    is_ipv4 "$found" && echo "$found"
}

# addresses prints the Mac's IPv4 addresses on interfaces that are up, the tunnel's aside, an
# interface and an address a line, in order.
# shellcheck disable=SC2154 # ws-tunnel sets tun
addresses() {
    ifconfig 2>/dev/null | awk -v tunnel="$tun" '
        /^[^ \t]/ {
            name = $1
            sub(/:$/, "", name)
            up = ($0 ~ /[<,]UP[,>]/)
            next
        }
        $1 == "inet" && up && name != tunnel { print name, $2 }
    ' | sort
}

# has_address_in CIDR succeeds where one of the Mac's addresses lies in the network CIDR.
has_address_in() {
    for each in $(printf '%s\n' "$seen" | awk '{ print $2 }'); do
        is_ipv4 "$each" && contains "$1" "$each" && return 0
    done
    return 1
}

# allowed_for ENDPOINT sets want to the networks of routes.d that the tunnel takes with ENDPOINT
# in use, and allowed to the peer's AllowedIPs: ws's address and those. A network that holds
# ENDPOINT would send the tunnel's own packets into it, so it goes in left_out instead.
# shellcheck disable=SC2154 # ws-tunnel sets server and routes
allowed_for() {
    want=
    out=
    for each in $routes; do
        if [ -n "$1" ] && contains "$each" "$1"; then
            case " $left_out " in
            *" $each "*) ;;
            *) say "leaves $each out of the tunnel while it holds the endpoint, $1" ;;
            esac
            out="$out $each"
        else
            want="$want $each"
        fi
    done
    want=${want# }
    left_out=${out# }
    allowed=$server/32
    for each in $want; do
        allowed=$allowed,$each
    done
}

# apply_routes routes want into the tunnel and stops routing what it no longer holds. A network
# it cannot route, as one another route already takes, it logs once and tries again each time.
apply_routes() {
    for each in $routed; do
        case " $want " in
        *" $each "*) ;;
        *)
            route -q -n delete -inet "$each" >/dev/null 2>&1 ||
                say "cannot remove the route to $each"
            ;;
        esac
    done
    kept=
    failed_now=
    for each in $want; do
        case " $routed " in
        *" $each "*) kept="$kept $each" ;;
        *)
            if route -q -n add -inet "$each" -interface "$tun" >/dev/null 2>&1; then
                kept="$kept $each"
            else
                case " $unroutable " in
                *" $each "*) ;;
                *) say "cannot route $each into $tun, as another route to it may be in the way" ;;
                esac
                failed_now="$failed_now $each"
            fi
            ;;
        esac
    done
    routed=${kept# }
    unroutable=${failed_now# }
}

# try HOST points the peer at HOST, and succeeds where a handshake completes within 5 s with the
# peer still at HOST. The peer is removed and added again: wireguard-go handshakes at once with a
# new peer that has a keepalive, while a new endpoint alone would wait for the next rekey, up to 2
# minutes away. A handshake that ws started over another path moves the endpoint there, and
# proves nothing of HOST.
# shellcheck disable=SC2154 # ws-tunnel sets key, port and keepalive
try() {
    allowed_for "$1"
    peer=$1
    wg set "$tun" peer "$key" remove 2>/dev/null
    if ! wg set "$tun" peer "$key" endpoint "$1:$port" allowed-ips "$allowed" \
        persistent-keepalive "$keepalive"; then
        say "cannot set the peer's endpoint to $1:$port"
        return 1
    fi
    waited=0
    while [ "$waited" -lt 5 ]; do
        sleep 1
        waited=$((waited + 1))
        handshake=$(peer_field latest-handshakes "$key")
        [ "${handshake:-0}" -gt 0 ] || continue
        at=$(peer_field endpoints "$key")
        [ "$at" = "$1:$port" ] && return 0
        say "the handshake came from $at, not $1:$port"
        return 1
    done
    return 1
}

# tiers points the tunnel at the first path that answers: the cable, where the Mac has an address
# in its subnet; the LAN, where it has one in WS_LAN; then WS_REMOTE, resolved anew. Without one,
# it logs that once, and the next tick tries them all again.
# shellcheck disable=SC2154 # ws-tunnel sets the cable's values and the LAN's
tiers() {
    misses=0
    why=
    for tier in cable lan remote; do
        case $tier in
        cable)
            has_address_in "$cable_net" || continue
            host=$cable_server
            ;;
        lan)
            if [ -z "$lan" ] || ! has_address_in "$lan"; then
                continue
            fi
            host=$lan_server
            ;;
        remote)
            [ -n "$remote" ] || continue
            if ! host=$(resolve "$remote"); then
                why="$why, $remote has no IPv4 address"
                continue
            fi
            ;;
        esac
        if try "$host"; then
            apply_routes
            if [ "$tier $host" != "$in_use" ]; then
                say "on the $tier tier, $host:$port"
                in_use="$tier $host"
                since=$(date +%s)
            fi
            write_state "$tier" "$host:$port"
            return 0
        fi
        why="$why, no handshake from $host:$port on the $tier tier"
    done
    if [ "$in_use" != none ]; then
        say "no path to ws answers${why:-, as the Mac is on none}; trying again every 10 s"
        in_use=none
        since=$(date +%s)
    fi
    # The routes stay in the tunnel, which drops what they carry, rather than let it go out by the
    # network the Mac is on.
    allowed_for "$peer"
    apply_routes
    write_state none ""
    return 1
}

# write_state writes TIER and ENDPOINT, with the interface, the time of the switch and the routes,
# for status.
# shellcheck disable=SC2154 # ws-tunnel sets state_file
write_state() {
    {
        echo "tier=$1"
        echo "endpoint=$2"
        echo "interface=$tun"
        echo "since=$since"
        echo "routes=$routed"
        echo "left_out=$left_out"
    } >"$state_file.new" && mv -f "$state_file.new" "$state_file"
}

# peer_field FIELD KEY prints what wg show IF FIELD gives for the peer KEY.
peer_field() {
    wg show "$tun" "$1" 2>/dev/null | awk -v key="$2" '$1 == key { print $2 }'
}

# alive succeeds while wireguard-go runs and answers on its socket.
alive() {
    kill -0 "$wg_pid" 2>/dev/null && wg show "$tun" public-key >/dev/null 2>&1
}

# tick runs every 10 s: it starts the log anew where it is too long, ends the daemon where
# wireguard-go has exited, and runs the tiers where the Mac's addresses changed, where no path
# answered, or where ws missed 3 pings in a row.
tick() {
    trim_log
    if ! alive; then
        say "wireguard-go has exited"
        exit 1
    fi
    if ! kill -0 "$notify_pid" 2>/dev/null; then
        watch_changes
    fi
    now=$(addresses)
    if [ "$now" != "$seen" ]; then
        seen=$now
        tiers
    elif [ "$in_use" = none ]; then
        tiers
    elif ping -c 1 -t 2 -q "$server" >/dev/null 2>&1; then
        misses=0
    else
        misses=$((misses + 1))
        if [ "$misses" -ge 3 ]; then
            say "ws missed 3 pings in a row; trying the paths again"
            tiers
        fi
    fi
}

# settle runs 2 s after a network change, as one change brings several and DHCP takes a moment,
# and runs the tiers where the Mac's addresses changed.
settle() {
    now=$(addresses)
    if [ "$now" != "$seen" ]; then
        seen=$now
        tiers
    fi
}

# watch_changes starts notifyutil, which prints a line each time configd posts a network change,
# as when an interface's IPv4 address comes or goes, into the daemon's events.
# shellcheck disable=SC2154 # ws-tunnel sets change
watch_changes() {
    notifyutil -w "$change" 3<&- >&4 &
    notify_pid=$!
}

# stop ends the helpers and wireguard-go, which takes the utun and its routes with it.
# shellcheck disable=SC2154 # ws-tunnel sets its files' names
stop() {
    trap - EXIT
    # The ticker's sleep first, which the ticker's end would leave running.
    [ -z "$ticker_pid" ] || pkill -P "$ticker_pid" 2>/dev/null
    for each in $ticker_pid $notify_pid; do
        kill "$each" 2>/dev/null
    done
    if [ -n "$wg_pid" ]; then
        kill "$wg_pid" 2>/dev/null
        wait "$wg_pid" 2>/dev/null
    fi
    rm -f "$name_file" "$state_file" "$state_file.new" "$fifo"
    say "stopped"
}
