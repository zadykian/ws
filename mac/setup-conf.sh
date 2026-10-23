# shellcheck shell=sh
# The functions of mac/setup.sh that read ~/.config/ws/mac.conf, the Mac's settings. setup-lib.sh
# sources it, so that a script that sources setup-lib.sh has them too. They call the functions of
# lib.sh and setup-lib.sh, and set a variable named for each of mac.conf's keys.

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, without running it: a
# KEY=value a line, the value bare or in a pair of quotes, blank lines and lines starting with #
# skipped. It stops on any other line, any other key, and a value of the wrong form.
# shellcheck disable=SC2154 # ws_cable_net and ws_tunnel_net are lib.sh's
ws_conf() {
    [ -f "$1" ] || ws_die "$1 is missing: docs/mac.md says what goes in it"
    WS_USER=root
    WS_PUBLIC_KEY=
    WS_LAN=
    WS_LAN_ADDRESS=
    WS_REMOTE=
    WS_REMOTE_PORT=
    WS_PORT=
    WS_TUNNEL_NETWORKS=
    WS_RESOLVERS=
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
        WS_USER)
            # shellcheck disable=SC2034 # for setup-ssh.sh and setup.sh
            WS_USER=$ws_value
            ws_is_user "$ws_value" || ws_ok=false
            ;;
        WS_PUBLIC_KEY)
            # shellcheck disable=SC2034 # for setup-ssh.sh and setup.sh
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
            # shellcheck disable=SC2034 # for setup.sh
            WS_PORT=$ws_value
            ws_is_port "$ws_value" || ws_ok=false
            ;;
        WS_TUNNEL_NETWORKS)
            # shellcheck disable=SC2034 # for setup.sh
            WS_TUNNEL_NETWORKS=$ws_value
            ws_are_networks "$ws_value" || ws_ok=false
            ;;
        WS_RESOLVERS)
            # shellcheck disable=SC2034 # for setup.sh
            WS_RESOLVERS=$ws_value
            ws_are_resolvers "$ws_value" || ws_ok=false
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

# ws_conf_form KEY prints the form a value of KEY takes, for ws_conf's message. It names lib.sh's
# ws_cable_net and ws_tunnel_net.
ws_conf_form() {
    case $1 in
    WS_USER) echo "a user name of 32 characters at most: letters, digits, ., _ and -, no - first" ;;
    WS_PUBLIC_KEY) echo "a WireGuard public key, 43 base64 characters and an =" ;;
    WS_LAN) echo "an IPv4 network, as 10.88.0.0/24, clear of $ws_cable_net and $ws_tunnel_net" ;;
    WS_LAN_ADDRESS) echo "an IPv4 address" ;;
    WS_REMOTE) echo "a host name or an IPv4 address" ;;
    WS_TUNNEL_NETWORKS)
        echo "IPv4 networks split by blanks, as 10.20.0.0/16, each clear of $ws_cable_net and" \
            "$ws_tunnel_net"
        ;;
    WS_RESOLVERS)
        echo "DOMAIN=ADDRESS[,ADDRESS...] split by blanks, as example.internal=10.20.0.53, each" \
            "domain once and in lower case, with up to 3 IPv4 addresses"
        ;;
    *) echo "a port, 1 to 65535" ;;
    esac
}
