# shellcheck shell=sh
# The wizard's Tunnel stage: the keys' exchange of the WireGuard tunnel between the Mac and ws. The
# Mac's key goes to wg0's mac.pub on ws, where wg0 takes it as its peer, and bootstrap.sh --tags
# wireguard runs there. Both go through the remote script over ws-ssh, as a change of wg0's peers
# ends the tunnel's sessions. wizard.sh sources it, and runs the stage after claude on ws. It calls
# the functions of lib.sh, setup-lib.sh and wizard-ws.sh, and reads wizard-host.sh's root_in.

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

# tunnel_done succeeds where ws's wg0 has the Mac's key as its peer, which mac.pub holds. It reads
# ws only where root logged in there in this run.
# shellcheck disable=SC2154 # root_in is wizard-host.sh's
tunnel_done() {
    [ "$root_in" = 1 ] && remote_read && [ "$(remote_get peer)" = ok ] && tunnel_key -n &&
        [ "$(remote_get mac.pub)" = "$tunnel_mac" ]
}

# tunnel_do gives ws's wg0 the Mac's key as its peer, where mac.pub holds another or wg0 lacks it,
# as after a tagged run that failed. The peer op writes the key to mac.pub, then bootstrap.sh
# --tags wireguard runs in a unit of its own. --check says what it would do.
tunnel_do() {
    sudo -v || ws_die "Tunnel: sudo -v failed; the Mac's key is root's, as an admin of the Mac"
    tunnel_key || ws_die "Tunnel: root's wg reads no key in $ws_etc/private.key: run" \
        "mac/setup.sh, which makes both"
    if [ "$(remote_get mac.pub)" != "$tunnel_mac" ] || [ "$(remote_get peer)" != ok ]; then
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
}

# tunnel_fail says what to do where ws's wg0 still lacks the Mac's key as its peer.
tunnel_fail() {
    echo "ws's wg0 still lacks the Mac's key as its peer, as above; mend what bootstrap.sh" \
        "printed, then run the wizard again"
}
