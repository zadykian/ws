# shellcheck shell=sh
# The wizard's Cable stage: the cable from the Mac's display to a port of ws's, outside the router,
# on a machine on the LAN alone.
# Docked, the Mac's cable gives that port carrier, which the files script's link lists among ws's
# wired ports. The stage names the port in host_vars/localhost.yml on ws, whose network role gives
# it 10.77.0.1. Then it names the Mac's service of the cable in mac.conf's WS_CABLE_SERVICE, and
# setup.sh gives that 10.77.0.2, and the port in WS_CABLE_PORT, which the stage names on ws again
# after ws's install. Undocked, the first setup waits, and the closing list says to dock the Mac.
# wizard.sh sources it, and runs the stage last. It calls the functions of lib.sh,
# setup-lib.sh, wizard-ask.sh, wizard-ws.sh, wizard-hook.sh, and wizard.sh's setup_do and by_hand.
# It reads wizard.sh's lock and WS_CONF, and wizard-host.sh's root_in.

# cable_away succeeds on a cloud VM or a VM on the Mac, which have no cable, so that the stage is
# done there, and the closing list names no dock.
cable_away() {
    [ "$WS_TARGET" != lan ]
}

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

# cable_done succeeds off the LAN, and where mac.conf's WS_CABLE_SERVICE is empty, or its
# WS_CABLE_PORT is empty or the cable that ws's status names, read once a run. Without the service,
# it succeeds too where no port that link lists has carrier, as the Mac isn't docked: the stage
# waits then, and cable_hand names the dock in the closing list. WS_CABLE_PORT in the environment
# ends the wait, as for a port that ws keeps down. It reads ws only where root logged in there in
# this run.
# shellcheck disable=SC2154 # root_in is wizard-host.sh's
cable_done() {
    ! cable_away || return 0
    ws_conf "$WS_CONF"
    if ! ask_lacks WS_CABLE_SERVICE; then
        [ -z "$WS_CABLE_SERVICE" ] || [ -z "$WS_CABLE_PORT" ] || { [ "$root_in" = 1 ] &&
            remote_read && [ "$(remote_get cable)" = "$WS_CABLE_PORT" ]; }
        return
    fi
    [ -z "${ask_set_WS_CABLE_PORT-}" ] && ! cable_none && [ "$root_in" = 1 ] && cable_read &&
        [ -z "$cable_up" ]
}

# cable_read sets cable_ports to what link lists on ws, a line per port: its name, its carrier, 1
# where a cable plugs in, and its IPv4 addresses. It sets cable_up to the ports with carrier, and
# cable_ws to the one that holds ws's end of the cable, 10.77.0.1. It fails where link did. With
# wait PORT, the files script lists them once PORT holds 10.77.0.1, or after 10 s, in one login.
# shellcheck disable=SC2154 # lock is wizard.sh's
cable_read() {
    cable_ports=$(remote_files "${1:-link}" ${2+"$2"} 2>"$lock/link.err") || return 1
    cable_up=$(printf '%s\n' "$cable_ports" | awk '$2 == 1 { print $1 }')
    cable_ws=$(printf '%s\n' "$cable_ports" | awk '/ 10\.77\.0\.1\// { print $1; exit }')
}

# cable_do takes ws's port of the cable, where none holds ws's end, from WS_CABLE_PORT or the
# terminal, the one with carrier by default. link names it in host_vars on ws, and bootstrap.sh
# --tags network,wireguard runs there, as both roles read it: the network role must give it
# 10.77.0.1. Then cable_service takes the Mac's service. none, as either answer, writes an empty
# WS_CABLE_SERVICE. --check asks nothing, and says what it would do. Where mac.conf has a service,
# cable_again takes the stage's steps.
cable_do() {
    cable_value=
    if ! ask_lacks WS_CABLE_SERVICE; then
        cable_again
        return
    fi
    if cable_none; then
        cable_write
        return
    fi
    cable_read || ws_die "Cable: link failed on ws: $(tail -n 1 "$lock/link.err")"
    if [ -z "$cable_ws" ]; then
        if ws_checking && [ -z "${ask_set_WS_CABLE_PORT-}" ]; then
            ws_say "Cable: would ask ws's port of the cable, name it in host_vars/localhost.yml" \
                "on ws, and run bootstrap.sh --tags network,wireguard there"
            return 0
        fi
        cable_ask WS_CABLE_PORT "ws's port of the cable, of $(printf '%s\n' "$cable_ports" |
            awk '{ printf "%s%s%s", (NR > 1 ? ", " : ""), $1, ($2 == 1 ? " (carrier)" : "") }')" \
            "$(cable_one "$cable_up")" "$(printf '%s\n' "$cable_ports" | cut -d ' ' -f 1)"
        if [ -z "$cable_value" ]; then
            cable_write
            return
        fi
        if ws_checking; then
            ws_say "Cable: would name $cable_value in host_vars/localhost.yml on ws, run" \
                "bootstrap.sh --tags network,wireguard there, then ask the Mac's service"
            return 0
        fi
        ws_say "Cable: $cable_value goes in host_vars/localhost.yml on ws, and bootstrap.sh" \
            "--tags network,wireguard runs there in a unit of its own"
        # shellcheck disable=SC2034 # wizard-ws.sh's remote_read reads it
        remote_got=0
        remote_files link "$cable_value" >&2 || return 1
        remote run --tags network,wireguard >&2 || return 1
        cable_read wait "$cable_value" && [ "$cable_ws" = "$cable_value" ] ||
            ws_die "Cable: ws's $cable_value lacks 10.77.0.1 after bootstrap.sh --tags" \
                "network,wireguard: plug the Mac's cable into it, or mend what that printed, then" \
                "run the wizard again"
    fi
    cable_service
}

# cable_service takes the Mac's service of the cable, from WS_CABLE_SERVICE in the environment or
# the terminal. The default is the one service with a 169.254 address, which an adapter gives
# itself where no DHCP server answers. It writes it in mac.conf, and runs setup.sh, which gives it
# 10.77.0.2. Then ping must reach ws's end over it. none writes an empty WS_CABLE_SERVICE.
cable_service() {
    cable_list=$(networksetup -listallnetworkservices) ||
        ws_die "Cable: networksetup -listallnetworkservices failed"
    cable_list=$(printf '%s\n' "$cable_list" | sed -e '/^An asterisk (\*) denotes/d' -e 's/^\*//')
    if ws_checking && [ -z "${ask_set_WS_CABLE_SERVICE-}" ]; then
        ws_say "Cable: would ask the Mac's service of the cable, write it in mac.conf as" \
            "WS_CABLE_SERVICE, run mac/setup.sh, which gives it 10.77.0.2, and ping 10.77.0.1"
        return 0
    fi
    cable_linked=$(printf '%s\n' "$cable_list" | while IFS= read -r each; do
        networksetup -getinfo "$each" 2>/dev/null | grep -q '^IP address: 169\.254\.' &&
            printf '%s\n' "$each"
    done)
    cable_ask WS_CABLE_SERVICE "The Mac's service of the cable, of $(cable_join "$cable_list")" \
        "$(cable_one "$cable_linked")" "$cable_list"
    cable_write || return 1
    [ -n "$cable_value" ] || return 0
    setup_do || return 1
    if ws_checking; then
        ws_say "Cable: would run ping -c 1 -t 3 10.77.0.1"
    elif ! ping -c 1 -t 3 10.77.0.1 >/dev/null 2>&1; then
        ws_die "Cable: ping -c 1 -t 3 10.77.0.1, ws's end of the cable, got no answer: mac.conf's" \
            "WS_CABLE_SERVICE must name the service of the adapter that the cable plugs into"
    fi
}

# cable_again names WS_CABLE_PORT in host_vars on ws where ws's status names none or another, as
# after ws's install, a failed roles' run or a change in mac.conf, and runs the roles there, with no
# question and no dock. It stops on a port that ws lacks. --check says what it would do.
cable_again() {
    remote_read || ws_die "Cable: status failed on ws: $(tail -n 1 "$lock/remote.err")"
    cable_read || ws_die "Cable: link failed on ws: $(tail -n 1 "$lock/link.err")"
    cable_vars=$(remote_get cable)
    cable_value=$WS_CABLE_PORT
    cable_names=$(printf '%s\n' "$cable_ports" | cut -d ' ' -f 1)
    cable_in "$cable_names" || ws_die "Cable: WS_CABLE_PORT=$cable_value names none of ws's wired" \
        "ports, $(cable_join "$cable_names"): set it to one"
    if [ "$cable_vars" != "$cable_value" ]; then
        if ws_checking; then
            ws_say "Cable: would name $cable_value in host_vars/localhost.yml on ws again, and" \
                "run bootstrap.sh --tags network,wireguard there"
        else
            ws_say "Cable: $cable_value goes in host_vars/localhost.yml on ws again, and" \
                "bootstrap.sh --tags network,wireguard runs there in a unit of its own"
            # shellcheck disable=SC2034 # wizard-ws.sh's remote_read reads it
            remote_got=0
            remote_files link "$cable_value" >&2 || return 1
            remote run --tags network,wireguard >&2 || return 1
        fi
    fi
}

# cable_ask KEY QUESTION DEFAULT LIST sets cable_value to KEY's answer: the environment's, else a
# line read at the terminal, stdin. There an empty line takes DEFAULT, none no value, and a pair of
# quotes comes off. A value must be a line of LIST. A bad answer at the terminal is asked again, and
# the environment's stops the run, as the Answers stage's.
cable_ask() {
    eval "cable_set=\${ask_set_$1-} cable_value=\${ask_env_$1-}"
    if [ -n "$cable_set" ]; then
        cable_in "$4" || ws_die "Cable: $1 comes from the environment, and '$cable_value' is" \
            "none of $(cable_join "$4"): set it to one, or to nothing for none"
        return 0
    fi
    [ -t 0 ] || ws_die "Cable: no terminal to ask $1: set $1, or set it to nothing for none"
    while :; do
        printf '%s (%s) [%s]: ' "$2" "$1" "${3:+$3, or }none" >&2
        read -r cable_value || ws_die "Cable: the terminal ended before $1's answer: set $1"
        case $cable_value in
        '') cable_value=$3 ;;
        none) cable_value= ;;
        \"*\" | \'*\')
            cable_value=${cable_value#?}
            cable_value=${cable_value%?}
            ;;
        esac
        cable_in "$4" && return 0
        ws_say "Cable: '$cable_value' is none of those"
    done
}

# cable_in LIST succeeds where cable_value is empty, for none, or a line of LIST.
cable_in() {
    [ -z "$cable_value" ] || printf '%s\n' "$1" |
        v=$cable_value awk '$0 == ENVIRON["v"] { found = 1 } END { exit !found }'
}

# cable_one LIST prints LIST's line where it has one alone.
cable_one() {
    printf '%s\n' "$1" | awk 'NF { n++; one = $0 } END { if (n == 1) print one }'
}

# cable_join LIST prints LIST's lines joined by commas.
cable_join() {
    printf '%s\n' "$1" | awk '{ printf "%s%s", (NR > 1 ? ", " : ""), $0 }'
}

# cable_write writes cable_value as mac.conf's WS_CABLE_SERVICE, in double quotes, after cable_ws as
# WS_CABLE_PORT, or empty for none, which the wizard never asks again.
cable_write() {
    cable_line=WS_CABLE_SERVICE=
    [ -z "$cable_value" ] || cable_line="WS_CABLE_PORT=$cable_ws
WS_CABLE_SERVICE=\"$cable_value\""
    ask_write "$cable_line
" Cable && ws_conf "$WS_CONF"
}

# cable_hand adds the dock to what stays by hand, where mac.conf has no WS_CABLE_SERVICE yet, as no
# port of ws's had carrier. It names WS_CABLE_PORT for the ports that ws keeps down, whose carrier
# link can't read, and an empty WS_CABLE_SERVICE for a Mac without the cable.
cable_hand() {
    ! cable_away || return 0
    ask_lacks WS_CABLE_SERVICE || return 0
    set -- "The cable: dock the Mac at the display that ws's cable plugs into, then run the line" \
        "again, which takes the cable's steps."
    cable_down=$(printf '%s\n' "${cable_ports-}" |
        awk '$2 == "-" { printf "%s%s", (n++ ? " or " : ""), $1 }')
    [ -z "$cable_down" ] || set -- "$@" "Where the cable plugs into $cable_down, which ws keeps" \
        "down, run the line with WS_CABLE_PORT=PORT in the environment."
    by_hand "$@" "Without the cable, run the line once with WS_CABLE_SERVICE= in the environment" \
        "(docs/mac-wizard.md, The cable)."
}

# cable_fail says what to do where ws's port of the cable, its address or the Mac's service failed.
cable_fail() {
    echo "ws's port of the cable, its 10.77.0.1, or the Mac's service of it failed, as above;" \
        "mend what that printed, then run the wizard again"
}
