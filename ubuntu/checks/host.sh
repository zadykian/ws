#!/bin/bash
# The real machine's checks of the host's own state, as root on the host: ubuntu/checks/README.md.
# Run as
#
#   host.sh [CHECK...]
#
# CHECK is a check's number on the page. 2 is ws-fans and the GPU. 6 is snapshots. 7 is forwarding with no
# br_netfilter. 8 is the host's ports and ufw. 10 is a second run's plan. Without one, it runs
# each. It prints a line a check, ok or FAIL, and exits 1 where any fails. It changes nothing.
# The checks that drive the guests are in guests.sh.
set -uo pipefail

fails=0
ws=/usr/local/bin/ws

pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    fails=$((fails + 1))
}

# check runs the command and prints ok where it succeeds, else FAIL with the last line it printed.
check() {
    local name=$1 out
    shift
    if out=$("$@" 2>&1); then pass "$name"; else fail "$name" "$(printf '%s' "$out" | tail -n 1)"; fi
}

# guests prints the name of each guest of the host's list, a line each.
guests() { "$ws" ubuntu host guests list | awk -F'\t' 'NF == 3 { print $1 }'; }

# slot_of prints the N of a guest's tunnel address, 10.99.N.1/24, from its own values.
slot_of() {
    awk -F'[ ./]+' '$1 == "WS_TUNNEL_ADDRESS:" { print $4 }' "/etc/ws/guests/$1/values.yml"
}

# plans succeeds where a setup's plan, which the command is, exits 0.
plans() {
    local status=0
    "$@" >/dev/null 2>&1 || status=$?
    [ "$status" -eq 0 ] || echo "the plan of $* exits $status"
    return "$status"
}

# fans_nvidia succeeds where a process named nvidia-smi runs in ws-fans' cgroup. A listing of the
# processes by args would match the check's own command line.
fans_nvidia() {
    local group pids
    group=$(systemctl show -p ControlGroup --value ws-fans.service)
    [ -n "$group" ] || {
        echo "ws-fans.service has no cgroup"
        return 1
    }
    pids=$(paste -sd, "${CGROUP_ROOT:-/sys/fs/cgroup}$group/cgroup.procs")
    [ -n "$pids" ] && ps -o comm= -p "$pids" | grep -qx nvidia-smi
}

check2() {
    check "2 the host's nvidia-smi answers" nvidia-smi -L
    check "2 ws-fans runs" systemctl is-active --quiet ws-fans.service
    check "2 ws-fans' nvidia-smi, which feeds it, runs" fans_nvidia
    check "2 ws-fans logs no failure of it since the last minute" bash -c \
        "! journalctl -u ws-fans.service --since -1min --no-pager | grep -iE 'nvidia.*(fail|error)'"
}

snapshots() {
    local n
    n=$(incus snapshot list "$1" --format csv | wc -l)
    [ "$n" -ge 2 ] || {
        echo "the guest $1 holds $n snapshots: take a second one, as the page says"
        return 1
    }
}

check6() {
    local g
    check "6 the profile takes a daily snapshot" bash -c \
        "[ \"\$(incus profile get ws snapshots.schedule)\" = @daily ] && [ -n \"\$(incus profile get ws snapshots.expiry)\" ]"
    for g in $(guests); do check "6 the guest $g holds two snapshots at least" snapshots "$g"; done
}

forwards_ipv4() { [ "$(sysctl -n net.ipv4.ip_forward)" = 1 ]; }

no_netfilter() { ! lsmod | grep -q '^br_netfilter' && [ ! -e /proc/sys/net/bridge ]; }

forwarding() {
    local g=$1 n table
    n=$(slot_of "$g")
    [ -n "$n" ] || {
        echo "the guest $g has no WS_TUNNEL_ADDRESS"
        return 1
    }
    table=$((1000 + n))
    ip -4 rule show | grep -q "iif wg$n lookup $table" || {
        echo "no rule of wg$n"
        return 1
    }
    ip -4 route show table "$table" | grep -q 'default via .* dev incusbr0' || {
        echo "no route in table $table"
        return 1
    }
}

check7() {
    local g
    check "7 no br_netfilter is loaded" no_netfilter
    check "7 IPv4 forwards" forwards_ipv4
    check "7 the table of sources is loaded" bash -c \
        "nft list table bridge ws_sources >/dev/null && nft list table ip ws_guest_out >/dev/null"
    check "7 the bridge is up" bash -c "ip -br link show incusbr0 | grep -q ' UP '"
    check "7 tunnel-forward.service has run" systemctl is-active --quiet tunnel-forward.service
    for g in $(guests); do check "7 the guest $g's tunnel forwards" forwarding "$g"; done
}

# allowed prints each allow rule of ufw's that no role of ws's opens. The roles open SSH, a tunnel's
# port, DHCP on the bridge, and Docker's networks to the resolver and OTLP.
allowed() {
    ufw show added | grep -E '^ufw (allow|route allow)' |
        grep -vE 'ufw allow (22/tcp|in on incusbr0 .*port 67|in on (docker0|br-\+) to [0-9.]+ port (53|4317)|[0-9]+/udp)' |
        grep -vE 'ufw route allow (in on incusbr0|in on wg[0-9]+)' || true
}

stray_rules() {
    local rules
    rules=$(allowed)
    [ -z "$rules" ] || {
        printf '%s\n' "$rules" | head -n 3
        return 1
    }
}

# rule_number prints the number of the first rule of ufw's status that matches the pattern. ufw pads
# a number below 10, as "[ 1]".
rule_number() {
    ufw status numbered | awk -v re="$1" '$0 ~ re { n = $0; sub(/^\[ */, "", n); sub(/\].*/, "", n); print n; exit }'
}

# deny_first succeeds where ufw's deny of wgN stands before the rule of SSH.
deny_first() {
    local deny ssh
    deny=$(rule_number "^[[] *[0-9]+[]].* on wg$1[[:space:]]+DENY IN")
    ssh=$(rule_number "^[[] *[0-9]+[]][[:space:]]+22/tcp[[:space:]]+ALLOW IN")
    [ -n "$deny" ] && [ -n "$ssh" ] && [ "$deny" -lt "$ssh" ]
}

# public_listeners prints each socket that listens on every address or on the uplink's, but SSH, the
# DHCP client and the tunnels' UDP ports. The kernel's WireGuard sockets list with no process, so
# their ports are those wg reports. A socket on a bridge's address, such as Incus's dnsmasq or the
# collector's socket, is the bridge's alone.
public_listeners() {
    local uplink tunnels
    tunnels=$(wg show all listen-port | awk '{ printf "%s ", $2 }') || {
        echo "wg show all listen-port failed"
        return
    }
    uplink=$(ip -4 route show default | awk '{ print $5; exit }')
    uplink=$(ip -4 -o addr show dev "$uplink" | awk '{ split($4, a, "/"); print a[1]; exit }')
    ss -Hlntu | awk -v up="$uplink" -v tunnels="$tunnels" '
        BEGIN { n = split(tunnels, t, " "); for (i = 1; i <= n; i++) tunnel[t[i]] = 1 }
        { n = split($5, a, ":"); h = substr($5, 1, length($5) - length(a[n]) - 1)
          if ($1 == "udp" && a[n] in tunnel) next
          if (h == "*" || h == "0.0.0.0" || h == "[::]" || h == up) print $1, $5 }' |
        grep -vE ' [^ ]*:(22|68)$' || true
}

check8() {
    local g n out
    check "8 ufw denies what comes in" bash -c "ufw status verbose | grep -q 'Default: deny (incoming)'"
    check "8 ufw holds no allow that ws's roles do not open" stray_rules
    for g in $(guests); do
        n=$(slot_of "$g")
        check "8 ufw denies a peer of $g's tunnel, ahead of SSH" deny_first "$n"
    done
    out=$(public_listeners)
    if [ -z "$out" ]; then pass "8 nothing listens beyond SSH and the tunnels"; else
        fail "8 nothing listens beyond SSH and the tunnels" "$(printf '%s' "$out" | head -n 3 | tr '\n' ';')"
    fi
}

check10() {
    local g
    check "10 ws ubuntu host setup plans nothing" plans "$ws" ubuntu host setup --plan
    for g in $(guests); do
        check "10 the guest $g's setup plans nothing" plans incus exec "$g" -- "$ws" ubuntu guest setup --plan
    done
}

main() {
    local arg all=(2 6 7 8 10)
    [ $# -gt 0 ] || set -- "${all[@]}"
    for arg in "$@"; do
        case $arg in
        2) check2 ;;
        6) check6 ;;
        7) check7 ;;
        8) check8 ;;
        10) check10 ;;
        *)
            echo "host.sh: no check $arg here" >&2
            exit 2
            ;;
        esac
    done
    [ "$fails" -eq 0 ]
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
