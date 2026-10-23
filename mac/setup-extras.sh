# shellcheck shell=sh
# The Mac's own steps of mac/setup.sh, beside ssh's hosts and the tunnel: for now, the cable's
# address on the display's adapter. setup.sh sources it after lib.sh and setup-lib.sh, whose
# functions it calls, and runs extras before the tunnel.

# The Mac's end of the cable, in lib.sh's ws_cable_net, and that /30's mask.
cable_client=10.77.0.2
cable_mask=255.255.255.252

# extras runs the Mac's own steps, each of which leaves the Mac alone where mac.conf lacks its key.
extras() {
    cable
}

# cable gives the service that mac.conf's WS_CABLE_SERVICE names a Manual Configuration of
# cable_client and cable_mask with no router, through sudo, where networksetup -getinfo shows other
# settings. With no router, the cable takes no default route from the uplink. A service
# networksetup lacks stops the run. Without WS_CABLE_SERVICE, it leaves the network alone.
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
    *)
        ws_die "networksetup -listallnetworkservices lacks \"$WS_CABLE_SERVICE\", mac.conf's" \
            "WS_CABLE_SERVICE: name the display's adapter's service as the list prints it"
        ;;
    esac
    cable_set && return 0
    manual="networksetup -setmanual \"$WS_CABLE_SERVICE\" $cable_client $cable_mask"
    if ws_checking; then
        ws_say "would run $manual"
        return 0
    fi
    sudo "$networksetup" -setmanual "$WS_CABLE_SERVICE" "$cable_client" "$cable_mask" ||
        ws_die "$manual failed"
    cable_set ||
        ws_die "networksetup -getinfo \"$WS_CABLE_SERVICE\" shows other settings after $manual"
    ws_say "ran $manual"
}

# cable_set succeeds where networksetup -getinfo shows WS_CABLE_SERVICE in a Manual Configuration of
# cable_client and cable_mask with no router, Router: (null). No source shows which lines getinfo
# prints while the display, and so its adapter, is away. So an address, mask or router line it
# leaves out passes, and a run docked checks the service again.
cable_set() {
    "$networksetup" -getinfo "$WS_CABLE_SERVICE" |
        awk -v address="$cable_client" -v mask="$cable_mask" '
            { sub(/[ \t\r]+$/, "") }
            NR == 1 { manual = ($0 == "Manual Configuration") }
            /^IP address: / { ips = 1 }
            $0 == "IP address: " address { ours = 1 }
            /^Subnet mask: / && $0 != "Subnet mask: " mask { other = 1 }
            /^Router: [0-9]/ { router = 1 }
            END { exit !(manual && (ours || !ips) && !other && !router) }
        '
}
