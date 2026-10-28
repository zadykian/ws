# shellcheck shell=sh
# The wizard's Tunnel stage: the keys' exchange of the WireGuard tunnel between the Mac and ws. The
# Mac's key goes to wg0's mac.pub on ws, where wg0 takes it as its peer, and bootstrap.sh --tags
# wireguard runs there. Both go through the remote script over ws-ssh, as a change of wg0's peers
# ends the tunnel's sessions. Then mac.conf's WS_PUBLIC_KEY follows wg0's key, and setup.sh runs
# again, which loads ws-tunnel. Last, ws-tunnel must show a handshake, and ssh ws log in through it.
# Without a handshake, the closing list says so, and the run goes on.
# wizard.sh sources it, and runs the stage after claude on ws. It calls the functions of lib.sh,
# setup-lib.sh, wizard-ask.sh, wizard-host.sh, wizard-ws.sh and wizard.sh's setup_do, and reads
# wizard.sh's lock and WS_CONF, and wizard-host.sh's root_in and host_first.

# tunnel_none becomes 1 where tunnel_check finds no handshake in this run.
tunnel_none=

# tunnel_after prints ws's run, which makes wg0, where that stage isn't done, as gh on ws's does. It
# prints setup.sh, which makes the Mac's key, where root has none to read.
tunnel_after() {
    if ! run_done; then
        echo "ws's run"
    elif ! tunnel_key -n; then
        echo setup.sh
    fi
}

# tunnel_key [-n] sets tunnel_mac to the Mac's WireGuard public key, from root's key by root's wg,
# as setup.sh prints it. With -n, sudo asks for no password. It fails where it reads no key.
# shellcheck disable=SC2154 # ws_etc is lib.sh's, ws_libexec setup-lib.sh's
tunnel_key() {
    tunnel_mac=$(sudo ${1+"$1"} sh -c '"$1" pubkey <"$2"' sh "$ws_libexec/wg" \
        "$ws_etc/private.key" 2>/dev/null) && ws_is_key "$tunnel_mac"
}

# tunnel_shake TRIES succeeds where ws-tunnel's status shows a handshake within 180 s, past which
# WireGuard drops the session. It reads up to TRIES times, a second apart, as a switch of the
# tunnel's way removes the peer and adds it again. sudo asks for no password.
tunnel_shake() {
    tunnel_try=1
    while :; do
        tunnel_age=$(sudo -n "$ws_libexec/ws-tunnel" status 2>/dev/null |
            sed -n 's/^latest handshake: \([0-9]*\) s ago$/\1/p')
        [ -z "$tunnel_age" ] || [ "$tunnel_age" -gt 180 ] || return 0
        [ "$tunnel_try" -lt "$1" ] || return 1
        tunnel_try=$((tunnel_try + 1))
        sleep 1
    done
}

# tunnel_via succeeds where ssh -G ws gives 10.99.0.1, ws's end of the tunnel, as config.d/ws does
# once launchd has ws-tunnel.
tunnel_via() {
    [ "$(ssh -G ws 2>/dev/null | awk '$1 == "hostname" { print $2; exit }')" = 10.99.0.1 ]
}

# tunnel_done succeeds where ws's wg0 has the Mac's key as its peer, WS_PUBLIC_KEY is wg0's key,
# and launchd has ws-tunnel loaded. It reads ws only where root logged in there in this run.
# ws-tunnel must have a handshake too, and ssh ws go through it. Where the run asked ws's address,
# ws isn't at the ways that ws-tunnel tries, and where tunnel_check set tunnel_none, ws-tunnel has
# no handshake: a later run checks those two.
# shellcheck disable=SC2154 # root_in and host_first are wizard-host.sh's
tunnel_done() {
    ws_conf "$WS_CONF"
    [ "$root_in" = 1 ] && remote_read && [ -n "$WS_PUBLIC_KEY" ] &&
        [ "$(remote_get wg0)" = "$WS_PUBLIC_KEY" ] && [ "$(remote_get peer)" = ok ] &&
        tunnel_key -n && [ "$(remote_get mac.pub)" = "$tunnel_mac" ] && ws_tunnel_loaded &&
        { [ -n "$host_first$tunnel_none" ] || { tunnel_shake 3 && tunnel_via; }; }
}

# tunnel_do gives ws's wg0 the Mac's key as its peer, where mac.pub holds another or wg0 lacks it,
# as after a tagged run that failed. The peer op writes the key to mac.pub, then bootstrap.sh
# --tags wireguard runs in a unit of its own. It sets WS_PUBLIC_KEY to wg0's key through ask_write,
# where it differs, and runs setup.sh again. --check says what it would do, with mac.conf's diff
# and setup.sh --check's where it knows wg0's key. Last, it waits for the handshake and logs in, or,
# where the run asked ws's address, leaves that to a later run.
tunnel_do() {
    tunnel_peered=0
    tunnel_had=$WS_PUBLIC_KEY
    sudo -v || ws_die "Tunnel: sudo -v failed; the Mac's key is root's, as an admin of the Mac"
    tunnel_key || ws_die "Tunnel: root's wg reads no key in $ws_etc/private.key: run" \
        "mac/setup.sh, which makes both"
    if [ "$(remote_get mac.pub)" != "$tunnel_mac" ] || [ "$(remote_get peer)" != ok ]; then
        tunnel_peered=1
        if ws_checking; then
            [ "$(remote_get mac.pub)" = "$tunnel_mac" ] || ws_say "Tunnel: would write the" \
                "Mac's key, $tunnel_mac, to /etc/wireguard/mac.pub on ws"
            ws_say "Tunnel: would run bootstrap.sh --tags wireguard on ws, for wg0's peer"
        else
            ws_say "Tunnel: the Mac's key goes to /etc/wireguard/mac.pub on ws, and bootstrap.sh" \
                "--tags wireguard runs there in a unit of its own, which gives wg0 that peer"
            # shellcheck disable=SC2034 # wizard-ws.sh's remote_read reads it
            remote_got=0
            remote peer "$tunnel_mac" >&2 || return 1
            remote run --tags wireguard >&2 || return 1
            remote_read || return 1
            [ "$(remote_get peer)" = ok ] || ws_die "Tunnel: ws's wg0 lacks the Mac's key after" \
                "bootstrap.sh --tags wireguard: mend what it printed, then run the wizard again"
        fi
    fi
    tunnel_ws=$(remote_get wg0) || tunnel_ws=-
    if [ "$tunnel_ws" = - ]; then
        ws_say "Tunnel: would set WS_PUBLIC_KEY in mac.conf to wg0's key, which that run makes," \
            "then run mac/setup.sh again, which loads ws-tunnel"
        return 0
    fi
    if [ "$WS_PUBLIC_KEY" != "$tunnel_ws" ]; then
        ask_write "WS_PUBLIC_KEY=$tunnel_ws
" Tunnel || return 1
        ws_conf "$WS_CONF"
    fi
    setup_do || return 1
    tunnel_check
}

# tunnel_check waits up to 30 s for ws-tunnel's handshake, where the run reached ws at its ways.
# Then it logs in to ws through the tunnel once, by the key that root takes, as fail2ban on ws bans
# 10.99.0.2 after a few that fail. ws's new key reaches ws-tunnel by setup.sh's SIGHUP, and the
# Mac's new peer on ws by a restart, so that it handshakes at once. --check names the restart too.
# Without a handshake, it sets tunnel_none and adds the tunnel to the closing list, and the run goes
# on, as the later stages reach ws by ws-ssh, outside the tunnel.
# shellcheck disable=SC2154 # ws_label is lib.sh's, lock wizard.sh's
tunnel_check() {
    if ws_checking; then
        [ "$tunnel_peered" = 0 ] || [ "$tunnel_had" != "$WS_PUBLIC_KEY" ] ||
            ws_say "Tunnel: would run launchctl kickstart -k system/$ws_label, for wg0's new peer"
        ws_say "Tunnel: would wait up to 30 s for ws-tunnel's handshake, then log in to ws" \
            "through it"
        return 0
    fi
    if [ -n "$host_first" ]; then
        tunnel_once="ws's lease is static"
        [ "$WS_TARGET" = lan ] || tunnel_once="mac.conf's way leads to ws"
        by_hand "The tunnel's handshake: ws-tunnel goes to ws's ways, not $host_first, where this" \
            "run reached ws; once $tunnel_once, run the wizard again, which checks it."
        return 0
    fi
    if [ "$tunnel_peered" = 1 ] && [ "$tunnel_had" = "$WS_PUBLIC_KEY" ]; then
        sudo launchctl kickstart -k "system/$ws_label" >/dev/null 2>&1 ||
            ws_die "Tunnel: launchctl kickstart -k system/$ws_label failed"
        ws_say "Tunnel: ran launchctl kickstart -k system/$ws_label, for wg0's new peer on ws"
    fi
    sleep 2
    if ! tunnel_shake 30; then
        tunnel_none=1
        by_hand "The tunnel's handshake: ws-tunnel had none with ws within 30 s, so ssh ws fails" \
            "while ssh ws-ssh works. ws must take UDP ${WS_PORT:-51820}, through the router's" \
            "forward or a cloud's firewall where WS_REMOTE leads through one, and ws-tunnel must" \
            "stay allowed in the Background (docs/mac.md, The Mac's tunnel). Then run the wizard" \
            "again, which checks it."
        return 0
    fi
    tunnel_via || ws_die "Tunnel: ssh -G ws gives another host than 10.99.0.1, ws's end of the" \
        "tunnel, which mac/setup.sh's config.d/ws names once launchd has ws-tunnel"
    [ "$(host_ssh -o BatchMode=yes ws echo ok 2>"$lock/tunnel.err" </dev/null)" = ok ] ||
        ws_die "Tunnel: ssh ws failed through the tunnel, where ws's run lets SSH in on wg0, and" \
            "fail2ban bans 10.99.0.2 for 10 minutes after 5 failures:" \
            "$(tail -n 1 "$lock/tunnel.err")"
}

# tunnel_fail says what to do where the keys still differ, launchd lacks ws-tunnel, or it has no
# handshake.
tunnel_fail() {
    echo "ws's wg0 still lacks the Mac's key, WS_PUBLIC_KEY wg0's key, or ws-tunnel its" \
        "handshake, as above; mend what it printed, then run the wizard again"
}
