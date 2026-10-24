# shellcheck shell=sh
# The functions of mac/setup.sh that read ~/.config/ws/mac.conf, the Mac's settings. setup-lib.sh
# sources it, so that a script that sources setup-lib.sh has them too. They call the functions of
# lib.sh and setup-lib.sh, and set a variable named for each of mac.conf's keys.

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, without running it: a
# KEY=value a line, the value bare or in a pair of quotes, blank lines and lines starting with #
# skipped. It stops on any other line, any other key, a value of the wrong form, and a
# WS_LAN_ADDRESS that is not a host's in WS_LAN. An empty value, KEY= or KEY="", is the key's
# default, as a missing line is, and passes its key's check.
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
    WS_CABLE_SERVICE=
    WS_GIT_NAME=
    WS_GIT_EMAIL=
    WS_GIT_SIGNING_KEY=
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
            WS_USER=${ws_value:-root}
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
        WS_CABLE_SERVICE)
            # shellcheck disable=SC2034 # for the cable's step
            WS_CABLE_SERVICE=$ws_value
            # A blank needs quotes, as in a shell's assignment; a * first marks a disabled service.
            case $ws_value in
            \** | *[[:cntrl:]]*) ws_ok=false ;;
            *' '*) [ "$ws_value" != "${ws_line#*=}" ] || ws_ok=false ;;
            esac
            ;;
        WS_GIT_NAME)
            # shellcheck disable=SC2034 # for setup-git.sh
            WS_GIT_NAME=$ws_value
            # git's config takes it in quotes, which a " would end and a \ escape.
            case $ws_value in
            *[[:cntrl:]\"\\]*) ws_ok=false ;;
            *' '*) [ "$ws_value" != "${ws_line#*=}" ] || ws_ok=false ;;
            esac
            ;;
        WS_GIT_EMAIL)
            WS_GIT_EMAIL=$ws_value
            case $ws_value in
            *[!A-Za-z0-9._%+@-]* | @*) ws_ok=false ;;
            *@*) ws_is_domain "${ws_value#*@}" || ws_ok=false ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_GIT_SIGNING_KEY)
            # shellcheck disable=SC2034 # for setup-git.sh
            WS_GIT_SIGNING_KEY=$ws_value
            case ${ws_value#\~} in
            *[!A-Za-z0-9._/@+-]*) ws_ok=false ;;
            /*.pub) ;;
            *) ws_ok=false ;;
            esac
            ;;
        *) ws_die "$1, line $ws_n: unknown key $ws_key" ;;
        esac
        [ -z "$ws_value" ] || $ws_ok ||
            ws_die "$1, line $ws_n: $ws_key=$ws_value is not $(ws_conf_form "$ws_key")"
    done <"$1"
    case ${WS_LAN:+lan}${WS_LAN_ADDRESS:+address} in
    lan | address) ws_die "$1: WS_LAN and WS_LAN_ADDRESS go together" ;;
    esac
    if [ -n "$WS_LAN" ] && ! ws_lan_host "$WS_LAN" "$WS_LAN_ADDRESS"; then
        ws_die "$1: WS_LAN_ADDRESS=$WS_LAN_ADDRESS is not $(ws_conf_form WS_LAN_ADDRESS), $WS_LAN"
    fi
    if [ -n "$WS_REMOTE_PORT" ] && [ -z "$WS_REMOTE" ]; then
        ws_die "$1: WS_REMOTE_PORT needs WS_REMOTE"
    fi
    case ${WS_GIT_NAME:+name}${WS_GIT_EMAIL:+email} in
    name | email) ws_die "$1: WS_GIT_NAME and WS_GIT_EMAIL go together" ;;
    esac
    if [ -n "$WS_GIT_SIGNING_KEY" ] && [ -z "$WS_GIT_NAME" ]; then
        ws_die "$1: WS_GIT_SIGNING_KEY needs WS_GIT_NAME and WS_GIT_EMAIL"
    fi
}

# ws_conf_form KEY prints the form a value of KEY takes, for ws_conf's message. It names lib.sh's
# ws_cable_net and ws_tunnel_net.
ws_conf_form() {
    case $1 in
    WS_USER) echo "a user name of 32 characters at most: letters, digits, ., _ and -, no - first" ;;
    WS_PUBLIC_KEY) echo "a WireGuard public key, 43 base64 characters and an =" ;;
    WS_LAN) echo "an IPv4 network, as 10.88.0.0/24, clear of $ws_cable_net and $ws_tunnel_net" ;;
    WS_LAN_ADDRESS) echo "a host's IPv4 address inside WS_LAN" ;;
    WS_REMOTE) echo "a host name or an IPv4 address" ;;
    WS_TUNNEL_NETWORKS)
        echo "IPv4 networks split by blanks, as 10.20.0.0/16, each clear of $ws_cable_net and" \
            "$ws_tunnel_net"
        ;;
    WS_RESOLVERS)
        echo "DOMAIN=ADDRESS[,ADDRESS...] split by blanks, as example.internal=10.20.0.53, each" \
            "domain once and in lower case, with up to 3 IPv4 addresses"
        ;;
    WS_CABLE_SERVICE)
        echo "a network service as networksetup -listallnetworkservices prints it, no * first," \
            "in quotes where it holds a blank"
        ;;
    WS_GIT_NAME) echo "a name, in quotes where it holds a blank, with no \" or \\" ;;
    WS_GIT_EMAIL)
        echo "an email address, as ws@example.com: letters, digits and ._%+- before the @, and" \
            "a host name after it"
        ;;
    WS_GIT_SIGNING_KEY)
        echo "a public key's file, as ~/.ssh/NAME.pub: a path from / or ~/ to .pub," \
            "of letters, digits and ._/@+-"
        ;;
    *) echo "a port, 1 to 65535" ;;
    esac
}

# ws_lan_host CIDR ADDRESS succeeds where ADDRESS is a host's address in the network CIDR: inside
# it, and neither its first address, the network's own, nor its last, its broadcast. In a /31 or a
# /32, every address is a host's.
ws_lan_host() {
    ws_cidr_contains "$1" "$2" || return 1
    ws_last=$(((1 << (32 - ${1#*/})) - 1))
    ws_host=$(($(ws_ipv4_int "$2") & ws_last))
    [ "$ws_last" -lt 3 ] || { [ "$ws_host" != 0 ] && [ "$ws_host" != "$ws_last" ]; }
}
