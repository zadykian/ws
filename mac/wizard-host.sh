# shellcheck shell=sh
# The wizard's Host key stage: ws's host key, matched against ws's console, in known_hosts under
# ws, the name that config.d/ws's HostKeyAlias gives each way to ws. wizard.sh sources it, and runs
# the stage after setup.sh, which writes config.d/ws. It calls the functions of lib.sh,
# setup-lib.sh and wizard-keys.sh, and reads wizard.sh's lock and WS_CONF, mac.conf's path.

# The host and port of the last scan of ws's key, which host_scan keeps in host_line, and ws's
# address that the run asked, where ws didn't answer at its way.
host_scanned=
host_first=
# The command at ws's console that prints the fingerprint of its ed25519 key.
host_print='ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub'

# host_after prints setup.sh, which writes the ways of ws-ssh, where ~/.ssh/config.d/ws is missing.
host_after() {
    [ -f "$HOME/.ssh/config.d/ws" ] || echo setup.sh
}

# host_done succeeds where ssh finds a way to ws-ssh, and known_hosts holds a key under ws. It reads
# no key of ws's, as ws's sshd counts each connection that doesn't log in against the Mac's address
# (PerSourcePenalties).
host_done() {
    host_way && ssh-keygen -F ws -f "$host_file" >/dev/null 2>&1
}

# host_do adds the ed25519 key that ws offers at its way to known_hosts, under ws, once WS_HOST_KEY,
# or the answer at the terminal, says that ws's console shows it. Where ws doesn't answer at its
# way, it reads the key at ws's address now. It stops where no way leads to ws, and warns where the
# way is the router's forward. --check asks nothing, and shows the line it would add.
# shellcheck disable=SC2154 # lock is wizard.sh's
host_do() {
    ws_conf "$WS_CONF"
    host_way || ws_die "Host key: ssh finds no way to ws-ssh from this network: no cable, no" \
        "address in WS_LAN and no WS_REMOTE_PORT; set those in mac.conf as docs/router.md has them"
    host_scan || host_ask || return 0
    # ssh -G prints the host name in lower case, which WS_REMOTE may not be.
    [ "$host_name" != "$(echo "$WS_REMOTE" | tr '[:upper:]' '[:lower:]')" ] ||
        ws_say "Host key: warning: through the router's forward, ws's new sshd faces the" \
            "internet, with any password login its installer allowed, until ws's run sets up its" \
            "security role"
    echo "$host_line" >"$lock/host.pub"
    host_fp=$(ssh-keygen -l -f "$lock/host.pub" | awk '{ print $2 }')
    [ -n "$host_fp" ] || ws_die "Host key: ssh-keygen cannot read the key that ws offers"
    host_trust
    host_mode=$(ws_mode "$host_file" 2>/dev/null) || host_mode=600
    { [ ! -f "$host_file" ] || awk 1 "$host_file"; } >"$lock/hosts" ||
        ws_die "Host key: cannot read $host_file"
    echo "ws $host_line" >>"$lock/hosts"
    ws_put "$host_file" "$host_mode" <"$lock/hosts"
}

# host_ask asks ws's address now, where ws doesn't answer at its way with an ed25519 key, as a new
# ws with a lease from the router's pool. WS_FIRST_ADDRESS gives it, else the answer at the
# terminal, where none stops the run, as before Ubuntu's install. ssh takes it as HostName for the
# rest of the run, and the stage reads ws's key there. --check asks nothing, and fails.
host_ask() {
    host_none="ws doesn't answer at $host_name:$host_port with an ed25519 key"
    if ws_checking; then
        ws_say "Host key: $host_none; a run would ask its address now"
        return 1
    fi
    host_first=${WS_FIRST_ADDRESS-}
    if [ -z "${WS_FIRST_ADDRESS+set}" ]; then
        [ -t 0 ] || ws_die "Host key: $host_none, and no terminal is there to ask its address" \
            "now: set WS_FIRST_ADDRESS"
        ws_say "Host key: $host_none; where Ubuntu isn't installed on ws yet, an empty answer" \
            "stops the run"
        until ws_is_ipv4 "$host_first"; do
            [ -z "$host_first" ] || ws_say "Host key: $host_first is not an IPv4 address"
            printf "ws's address now, as the router's leases show it: " >&2
            read -r host_first || ws_die "Host key: the terminal ended before ws's address"
            [ -n "$host_first" ] || ws_die "Host key: no address for ws; once Ubuntu is installed" \
                "on it, run the wizard again"
        done
    fi
    ws_is_ipv4 "$host_first" ||
        ws_die "Host key: WS_FIRST_ADDRESS=$host_first is not an IPv4 address"
    if ! host_way || ! host_scan; then
        ws_die "Host key: ws doesn't answer at $host_first:$host_port with an ed25519 key either"
    fi
}

# host_way reads ssh's settings for ws-ssh, which setup.sh's config.d/ws gives, with HostName the
# address that the run asked: the host and port that ssh reaches, and known_hosts. It fails where
# no way matched, as ws-ssh is then its own host.
host_way() {
    host_g=$(ssh -G ${host_first:+-o "HostName=$host_first"} ws-ssh 2>/dev/null) || return 1
    host_name=$(host_get hostname)
    host_port=$(host_get port)
    host_file=$(keys_home "$(host_get userknownhostsfile)")
    [ "$host_name" != ws-ssh ]
}

# host_get KEY prints the first word of KEY's value in what ssh -G prints for ws-ssh.
host_get() {
    printf '%s\n' "$host_g" | awk -v key="$1" '$1 == key { print $2; exit }'
}

# host_scan sets host_line to the ed25519 key that ws offers at host_name and host_port, within
# 10 s, over IPv4 alone where ssh takes it alone, once a run. It fails where ws offers none.
host_scan() {
    if [ "$host_scanned" != "$host_name $host_port" ]; then
        host_scanned="$host_name $host_port"
        [ "$(host_get addressfamily)" != inet ] || set -- -4
        host_line=$(ssh-keyscan "$@" -T 10 -t ed25519 -p "$host_port" "$host_name" 2>/dev/null |
            awk '$2 == "ssh-ed25519" { print $2, $3; exit }')
    fi
    [ -n "$host_line" ]
}

# host_trust stops unless host_fp, the fingerprint of ws's key, is the one at ws's console.
# WS_HOST_KEY, else the answer at the terminal, must be yes, or hold that fingerprint as the
# console's ssh-keygen prints it. --check asks nothing.
host_trust() {
    host_offer="ws at $host_name:$host_port offers the ed25519 key $host_fp"
    if ws_checking; then
        ws_say "Host key: $host_offer; a run would ask whether ws's console shows it"
        return 0
    fi
    host_answer=${WS_HOST_KEY-}
    if [ -z "${WS_HOST_KEY+set}" ]; then
        [ -t 0 ] || ws_die "Host key: $host_offer, and no terminal is there to compare it with" \
            "ws's console: set WS_HOST_KEY to yes where it shows it, as $host_print prints it"
        ws_say "Host key: $host_offer. At ws's console, $host_print prints the one it has"
        printf 'Answer yes where the two match, or paste what it prints: ' >&2
        read -r host_answer || ws_die "Host key: the terminal ended before the answer"
    fi
    case " $host_answer " in
    *" $host_fp "* | ' '[Yy][Ee][Ss]' ') ;;
    *) ws_die "Host key: '$host_answer' is neither yes nor $host_fp, the key that ws offers at" \
        "$host_name:$host_port: compare that with ws's console, and check ws's address" ;;
    esac
}

# host_fail says what to do where known_hosts still lacks ws's key.
host_fail() {
    echo "$host_file still holds no key under ws, as above; run the wizard again"
}
