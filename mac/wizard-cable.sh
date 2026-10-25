# shellcheck shell=sh
# The wizard's Cable stage: the cable from the Mac's display to a port of ws's, outside the router.
# Docked, the Mac's cable gives that port carrier, which the files script's link lists among ws's
# wired ports. The port that host_vars/localhost.yml names on ws takes 10.77.0.1 from ws's network
# role, and the Mac's service that mac.conf's WS_CABLE_SERVICE names takes 10.77.0.2 from setup.sh.
# Undocked, the stage waits, and the closing list says to dock the Mac. wizard.sh sources it, and
# runs the stage last. It calls the functions of lib.sh, setup-lib.sh, wizard-ask.sh, wizard-ws.sh,
# wizard-hook.sh, and wizard.sh's by_hand. It reads wizard.sh's lock and WS_CONF, and
# wizard-host.sh's root_in.

# cable_none succeeds where the environment answers none for the Mac's service of the cable, as an
# empty WS_CABLE_SERVICE does.
cable_none() {
    [ "${ask_set_WS_CABLE_SERVICE-}" = 1 ] && [ -z "${ask_env_WS_CABLE_SERVICE-}" ]
}

# cable_after prints ws's run, which makes the clone that holds host_vars on ws, where that stage
# isn't done.
cable_after() {
    cable_none || run_done || echo "ws's run"
}

# cable_done succeeds where mac.conf has a line for WS_CABLE_SERVICE, even an empty one. It succeeds
# too where no port that link lists has carrier, as the Mac isn't docked: the stage waits then, and
# cable_hand names the dock in the closing list. It reads ws only where root logged in there in
# this run.
# shellcheck disable=SC2154 # root_in is wizard-host.sh's
cable_done() {
    ask_lacks WS_CABLE_SERVICE || return 0
    ! cable_none && [ "$root_in" = 1 ] && cable_read && [ -z "$cable_up" ]
}

# cable_read sets cable_ports to what link lists on ws, a line per port: its name, its carrier, 1
# where a cable plugs in, and its IPv4 addresses. It sets cable_up to the ports with carrier, and
# cable_ws to the one that holds ws's end of the cable, 10.77.0.1. It fails where link did.
# shellcheck disable=SC2154 # lock is wizard.sh's
cable_read() {
    cable_ports=$(remote_files link 2>"$lock/link.err") || return 1
    cable_up=$(printf '%s\n' "$cable_ports" | awk '$2 == 1 { print $1 }')
    cable_ws=$(printf '%s\n' "$cable_ports" | awk '/ 10\.77\.0\.1\// { print $1; exit }')
}

# cable_do writes an empty WS_CABLE_SERVICE where the environment answers none. Docked, it stops,
# naming the step by hand: ws's port in host_vars on ws, where none holds ws's end, 10.77.0.1, or
# else the Mac's service in mac.conf.
cable_do() {
    cable_value=
    if cable_none; then
        cable_write
        return
    fi
    cable_read || ws_die "Cable: link failed on ws: $(tail -n 1 "$lock/link.err")"
    [ -n "$cable_ws" ] || ws_die "Cable: no port of ws's holds 10.77.0.1, ws's end of the cable:" \
        "name the one with carrier in host_vars/localhost.yml there (docs/network.md, The direct" \
        "link), then run the wizard again"
    ws_die "Cable: ws's $cable_ws holds 10.77.0.1: set WS_CABLE_SERVICE in mac.conf to the Mac's" \
        "service of the cable (docs/mac.md, The cable), or to nothing for none"
}

# cable_write writes cable_value as mac.conf's WS_CABLE_SERVICE, in double quotes, or empty for
# none, which the wizard never asks again.
cable_write() {
    cable_line=WS_CABLE_SERVICE=
    [ -z "$cable_value" ] || cable_line="WS_CABLE_SERVICE=\"$cable_value\""
    ask_write "$cable_line
" Cable && ws_conf "$WS_CONF"
}

# cable_hand adds the dock to what stays by hand, where mac.conf has no WS_CABLE_SERVICE yet, as no
# port of ws's had carrier, and an empty WS_CABLE_SERVICE for a Mac without the cable.
cable_hand() {
    ask_lacks WS_CABLE_SERVICE || return 0
    by_hand "The cable: dock the Mac at the display that ws's cable plugs into, then run the line" \
        "again, which takes the cable's steps. Without the cable, run the line once with" \
        "WS_CABLE_SERVICE= in the environment (docs/mac-wizard.md, The cable)."
}

# cable_fail says what to do where ws's port of the cable, its address or the Mac's service failed.
cable_fail() {
    echo "ws's port of the cable, its 10.77.0.1, or the Mac's service of it failed, as above;" \
        "mend what that printed, then run the wizard again"
}
