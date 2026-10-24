# shellcheck shell=sh source-path=SCRIPTDIR
# The Mac's own steps of mac/setup.sh, beside ssh's hosts and the tunnel: the cable's address on
# the display's adapter, and git's identity and signing key. setup.sh sources it after lib.sh and
# setup-lib.sh, whose functions it calls, and runs extras before the tunnel.

# The Mac's end of the cable, in lib.sh's ws_cable_net, and that /30's mask.
cable_client=10.77.0.2
cable_mask=255.255.255.252

# The git step's functions are setup-git.sh's, sourced from dir, mac/'s path, as setup.sh sets it.
# shellcheck source=setup-git.sh disable=SC2154 # dir is setup.sh's
. "$dir/setup-git.sh"

# extras runs the Mac's own steps, each of which leaves the Mac alone where mac.conf lacks its key.
extras() {
    cable
    git_mac
}

# cable gives the service that mac.conf's WS_CABLE_SERVICE names a Manual Configuration of
# cable_client and cable_mask with no router, through sudo, where networksetup -getinfo shows other
# settings. With no router, the cable takes no default route from the uplink. A service
# networksetup lacks stops the run, as does one with a router outside the cable's /30, as an
# uplink has. A disabled one draws a warning. Without WS_CABLE_SERVICE, it leaves the network alone.
# shellcheck disable=SC2154 # ws_cable_net is lib.sh's
cable() {
    [ -n "$WS_CABLE_SERVICE" ] || return 0
    # sudo gets the program's path as the user's PATH finds it, whatever PATH sudo would keep.
    networksetup=$(command -v networksetup) ||
        ws_die "cannot find networksetup: put /usr/sbin, where macOS keeps it, on PATH"
    services=$("$networksetup" -listallnetworkservices) ||
        ws_die "networksetup -listallnetworkservices failed"
    nl='
'
    case $nl$services$nl in
    *"$nl$WS_CABLE_SERVICE$nl"*) ;;
    *"$nl*$WS_CABLE_SERVICE$nl"*)
        ws_say "warning: networksetup lists \"$WS_CABLE_SERVICE\", mac.conf's WS_CABLE_SERVICE," \
            "as disabled, so the cable carries nothing: turn it on in System Settings › Network"
        ;;
    *)
        ws_die "networksetup -listallnetworkservices lacks \"$WS_CABLE_SERVICE\", mac.conf's" \
            "WS_CABLE_SERVICE: name the display's adapter's service as the list prints it"
        ;;
    esac
    cable_set && return 0
    getinfo="networksetup -getinfo \"$WS_CABLE_SERVICE\" shows"
    # A router outside the cable's /30 is an uplink's: the cable, with no DHCP, gives none.
    case $cable_other in
    *'Router: '*)
        router=${cable_other##*Router: }
        { ws_is_ipv4 "$router" && ws_cidr_contains "$ws_cable_net" "$router"; } ||
            ws_die "$getinfo Router: $router, so the service routes to another network:" \
                "mac.conf's WS_CABLE_SERVICE must name the display's adapter's service"
        ;;
    esac
    manual="networksetup -setmanual \"$WS_CABLE_SERVICE\" $cable_client $cable_mask"
    if ws_checking; then
        ws_say "would run $manual"
        return 0
    fi
    sudo "$networksetup" -setmanual "$WS_CABLE_SERVICE" "$cable_client" "$cable_mask" ||
        ws_die "$manual failed"
    cable_set || ws_die "$getinfo $cable_other after $manual: set it by hand in System Settings ›" \
        "Network, as docs/mac.md's The cable says, or empty mac.conf's WS_CABLE_SERVICE"
    ws_say "ran $manual"
}

# cable_set succeeds where networksetup -getinfo shows WS_CABLE_SERVICE in a Manual Configuration of
# cable_client and cable_mask with no router, Router: (null). No source shows which lines getinfo
# prints while the display, and so its adapter, is away. So an address, mask or router line it
# leaves out passes, and a run docked checks the service again. It sets cable_other to the lines
# that differ, joined by "; ", or to nothing.
cable_set() {
    cable_other=$("$networksetup" -getinfo "$WS_CABLE_SERVICE" |
        awk -v address="$cable_client" -v mask="$cable_mask" '
            { sub(/[ \t\r]+$/, "") }
            !NF { next }
            !first++ && $0 != "Manual Configuration" { method = "; " $0 }
            $0 == "IP address: " address { ours = 1 }
            /^IP address: / && $0 != "IP address: " address { ips = ips "; " $0 }
            /^Subnet mask: / && $0 != "Subnet mask: " mask { masks = masks "; " $0 }
            /^Router: [0-9]/ { routers = routers "; " $0 }
            END {
                if (!first) method = "; nothing"
                if (ours) ips = ""
                print substr(method ips masks routers, 3)
            }
        ')
    [ -z "$cable_other" ]
}
