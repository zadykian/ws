# shellcheck shell=sh
# The wizard's Host key and Root's key stages: ws's host key, matched against ws's console, in
# known_hosts under ws, the name that config.d/ws's HostKeyAlias gives each way to ws. Then a check
# that root on ws takes the Mac's key, so that ssh ws-ssh logs in as root. wizard.sh sources it, and
# runs both after setup.sh, which writes config.d/ws. They call the functions of lib.sh,
# setup-lib.sh, wizard-keys.sh and wizard-gh.sh, and read wizard.sh's lock and WS_CONF, mac.conf's
# path.

# The host and port of the last scan of ws's key, which host_scan keeps in host_line, and ws's
# address that the run asked, where ws didn't answer at its way.
host_scanned=
host_first=
# root_in becomes 1 once ssh logs in to ws as root by the Mac's key, in this run.
root_in=0
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
# way, it reads the key at ws's address now. It stops where no way leads to ws, naming what ws's
# target needs, and warns where the way is WS_REMOTE. --check asks nothing, and shows the line it
# would add. Where known_hosts is a link, as a dotfiles manager makes it, it prints that line, and
# stops, as ask_write does for mac.conf: ws_put would replace the link. --check, which compared no
# key with ws's console, stops without it.
# shellcheck disable=SC2154 # lock is wizard.sh's
host_do() {
    ws_conf "$WS_CONF"
    if ! host_way; then
        case $WS_TARGET in
        lan) host_how="set those in mac.conf as docs/router.md has them" ;;
        mac-vm) host_how="start the VM, or set WS_LAN to the Mac's network for it" ;;
        *) host_how="set those in mac.conf" ;;
        esac
        [ "$WS_TARGET" = lan ] || host_how="$host_how, as docs/mac-targets.md says"
        ws_die "Host key: ssh finds no way to ws-ssh from this network: no cable, no address in" \
            "WS_LAN and no WS_REMOTE_PORT; $host_how"
    fi
    host_scan || host_ask || return 0
    # ssh -G prints the host name in lower case, which WS_REMOTE may not be.
    [ "$host_name" != "$(echo "$WS_REMOTE" | tr '[:upper:]' '[:lower:]')" ] ||
        ws_say "Host key: warning: through WS_REMOTE, ws's new sshd faces the internet, with any" \
            "password login its installer allowed, until ws's run sets up its security role"
    echo "$host_line" >"$lock/host.pub"
    host_fp=$(ssh-keygen -l -f "$lock/host.pub" | awk '{ print $2 }')
    [ -n "$host_fp" ] || ws_die "Host key: ssh-keygen cannot read the key that ws offers"
    host_trust
    if [ -L "$host_file" ]; then
        host_link="$host_file is a link to $(readlink "$host_file"), which the wizard leaves"
        ! ws_checking || ws_die "Host key: $host_link: a run would print ws's line for its file"
        ws_say "Host key: $host_link: add this line to its file, then run the wizard again"
        echo "ws $host_line" >&2
        exit 1
    fi
    host_mode=$(ws_mode "$host_file" 2>/dev/null) || host_mode=600
    { [ ! -f "$host_file" ] || awk 1 "$host_file"; } >"$lock/hosts" ||
        ws_die "Host key: cannot read $host_file"
    echo "ws $host_line" >>"$lock/hosts"
    ws_put "$host_file" "$host_mode" <"$lock/hosts"
}

# host_ask asks ws's address now, where ws doesn't answer at its way with an ed25519 key, as a new
# ws with a lease from the router's pool. WS_FIRST_ADDRESS gives it, else the answer at the
# terminal, where none stops the run, as before Ubuntu's install. The question names where ws's
# machine shows it (docs/mac-targets.md). ssh takes it as HostName for the rest of the run, and the
# stage reads ws's key there. --check asks nothing, and fails.
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
        case $WS_TARGET in
        cloud) host_where="as the provider shows the VM's public one" ;;
        mac-vm) host_where="as ip -4 address prints it at ws's console" ;;
        *) host_where="as the router's leases show it" ;;
        esac
        until ws_is_ipv4 "$host_first"; do
            [ -z "$host_first" ] || ws_say "Host key: $host_first is not an IPv4 address"
            printf "ws's address now, %s: " "$host_where" >&2
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

# host_ssh ARG... runs ssh with ARGs, and HostName the address that the Host key stage asked, where
# it asked one, with 10 s to connect. Whatever the user's ssh_config says, ssh stops where
# known_hosts lacks ws's key, or holds another, and adds nothing there: no other key, no address.
host_ssh() {
    ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no \
        ${host_first:+-o "HostName=$host_first"} "$@"
}

# host_fail says what to do where known_hosts still lacks ws's key.
host_fail() {
    echo "$host_file still holds no key under ws, as above; run the wizard again"
}

# root_after prints Host key where known_hosts lacks ws's key, without which ssh logs in nowhere.
root_after() {
    host_done || echo "Host key"
}

# root_done succeeds where ssh logs in to ws-ssh as root by the Mac's key, and runs a command, and
# sets root_in then. It connects only where known_hosts holds a key under ws, so --check meets ws
# only once the Host key stage is done. ssh's messages go in root.err in the lock.
# shellcheck disable=SC2034 # wizard-ws.sh reads root_in
root_done() {
    [ "$(host_get user)" = root ] && ssh-keygen -F ws -f "$host_file" >/dev/null 2>&1 &&
        [ "$(host_ssh -o BatchMode=yes ws-ssh echo ok 2>"$lock/root.err" </dev/null)" = ok ] &&
        root_in=1
}

# root_do stops, as the wizard adds no key to ws: ws's keys come from the GitHub account at its
# install, and the stage gh on the Mac has put the Mac's there. It names the Mac's key, and what
# gives root that key without the wizard, at a terminal or not: the account's keys at the install,
# or, on a ws installed before the key, its line by hand. Where ws doesn't answer, the Host
# key stage's question gives its address now, and root may take the key there. It stops where ssh
# logs in as another user than root, and where ws offers another host key than known_hosts holds,
# which it never removes. --check asks nothing, and says that a run would stop. Both show ssh's
# error, where it printed one.
# shellcheck disable=SC2154 # wizard-gh.sh's gh_key sets gh_pub
root_do() {
    root_as=$(host_get user)
    [ "$root_as" = root ] || ws_die "Root's key: ssh logs in to ws as $root_as, mac.conf's" \
        "WS_USER, and the wizard sets ws up as root alone: take the README's steps by hand"
    if ! host_scan; then
        host_ask || return 0
        ! root_done || return 0
    fi
    ! grep -q 'Host key verification failed' "$lock/root.err" || ws_die "Root's key: ws at" \
        "$host_name:$host_port offers another host key than known_hosts holds under ws: where" \
        "ws was installed again, run ssh-keygen -R ws, then the wizard again"
    # ssh ends each line with \r\n, and an agent's refusal to sign comes before its last line.
    root_err=$(awk '{ sub(/\r$/, "") } NF { printf "%s%s", (n++ ? "; " : ""), $0 }' \
        "$lock/root.err" 2>/dev/null)
    [ -z "$root_err" ] || root_err=". ssh's error: $root_err"
    if ws_checking; then
        ws_say "Root's key: root on ws doesn't log in by the Mac's key; a run would stop, as the" \
            "wizard adds no key to ws$root_err"
        return 0
    fi
    gh_key || ws_die "Root's key: the Mac's key for ws has no .pub"
    ws_die "Root's key: root on ws doesn't log in by the Mac's key, $gh_pub, and the wizard adds" \
        "no key to ws. ws takes its keys from the GitHub account at its install: Ubuntu's" \
        "installer imports them for its user, and on a cloud VM, cloud-init gives its default" \
        "user the provider's keys, which must be the account's. Copy that user's" \
        "authorized_keys to root, as the README's \"A new machine\" does: root's file is" \
        "replaced, so cloud-init's forced command goes. On a ws installed before the key," \
        "$(ws_identity_step root). Then run the wizard again$root_err"
}

# root_fail says what to do where root still refuses the Mac's key.
root_fail() {
    echo "root on ws still refuses the Mac's key: ws's sshd must let root log in by key"
}
