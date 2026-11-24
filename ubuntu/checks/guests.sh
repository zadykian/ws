#!/bin/bash
# The real machine's checks that drive each guest from the host, as root: ubuntu/checks/README.md.
# Run as
#
#   guests.sh CHECK... [guest=NAME...] [MAC_LAN=ADDRESS] [MAC_CABLE=ADDRESS]
#
# CHECK is a check's number on the page. 1 is Docker and a compose stack. 2 is CUDA in two guests. 3 is
# what a guest's root cannot reach. 5 is ikeward's VPN. 9 is a container's OTLP in SigNoz. 11 is
# leases, names, host keys and secrets. It acts on each guest of the host's list, or on those that
# guest=NAME names. MAC_LAN is the Mac's address on the home LAN. MAC_CABLE is its end of the
# cable, 10.77.0.2 by default. It pushes guest.sh to each guest, runs it and removes it. It prints
# a line a check, ok or FAIL, and exits 1 where any fails.
set -uo pipefail

fails=0
ws=/usr/local/bin/ws
here=$(cd "$(dirname "$0")" && pwd)
mac_lan=
mac_cable=10.77.0.2
named=()

pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    fails=$((fails + 1))
}

check() {
    local name=$1 out
    shift
    if out=$("$@" 2>&1); then pass "$name"; else fail "$name" "$(printf '%s' "$out" | tail -n 1)"; fi
}

# list prints each guest as name, address and state, tab apart.
list() { "$ws" ubuntu host guests list | awk -F'\t' 'NF == 3'; }

guests() {
    if [ ${#named[@]} -gt 0 ]; then printf '%s\n' "${named[@]}"; else list | cut -f1; fi
}

address_of() { list | awk -F'\t' -v n="$1" '$1 == n { print $2 }'; }

# slot_of prints the N of a guest's tunnel address, 10.99.N.1/24, from its own values.
slot_of() { awk -F'[ ./]+' '$1 == "WS_TUNNEL_ADDRESS:" { print $4 }' "/etc/ws/guests/$1/values.yml"; }

# run_in pushes guest.sh to the guest, runs it there with the arguments and removes it. The exit
# status is the script's.
run_in() {
    local g=$1 status
    shift
    incus file push "$here/guest.sh" "$g/root/ws-guest-checks.sh" --mode 0700 >/dev/null || {
        echo "FAIL the push of guest.sh failed"
        return 1
    }
    incus exec "$g" -- bash /root/ws-guest-checks.sh "$@"
    status=$?
    incus exec "$g" -- rm -f /root/ws-guest-checks.sh
    return $status
}

# drive runs guest.sh in the guest and prints its lines, prefixed with the guest. A run that exits
# non-zero fails the check as a whole, whatever it printed.
drive() {
    local g=$1
    run_in "$@" | sed "s/^\\(ok  \\|FAIL\\) /\\1 [$g] /" || {
        fails=$((fails + 1))
        return 1
    }
}

# each drives the guest script on each guest.
each() {
    local g
    for g in $(guests); do drive "$g" "$@"; done
}

check1() { each 1; }

holds_gpu() { incus config device show "$1" | grep -q '^gpu:'; }

check2() {
    local g pid pids=() holders=() tmp
    tmp=$(mktemp -d)
    for g in $(guests); do holds_gpu "$g" && holders+=("$g"); done
    [ ${#holders[@]} -ge 2 ] || fail "2 two guests hold the GPU" "${#holders[@]} guest holds it: ${holders[*]}"
    for g in "${holders[@]}"; do
        drive "$g" 2 >"$tmp/gpu-$g" &
        pids+=($!)
    done
    for pid in "${pids[@]}"; do wait "$pid" || fails=$((fails + 1)); done
    for g in "${holders[@]}"; do cat "$tmp/gpu-$g"; done
    rm -rf "$tmp"
}

check3() {
    local g o n router lan bridge tunnel cable resolver args
    router=$(ip -4 route show default | awk '{ print $3; exit }')
    lan=$(ip -4 -o addr show dev "$(ip -4 route show default | awk '{ print $5; exit }')" | awk '{ split($4, a, "/"); print a[1]; exit }')
    bridge=$(incus network get incusbr0 ipv4.address | cut -d/ -f1)
    tunnel=$(ip -4 -o addr show dev wg0 2>/dev/null | awk '{ split($4, a, "/"); print a[1]; exit }')
    cable=$(ip -4 -o addr | awk '$4 ~ /^10\.77\.0\./ { split($4, a, "/"); print a[1]; exit }')
    resolver=$(resolvectl dns 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | sort -u | head -n 1)
    [ -n "$mac_lan" ] || fail "3 the Mac's LAN address is given" "pass MAC_LAN=ADDRESS"
    for g in $(guests); do
        n=$(slot_of "$g")
        args=(router="$router" host-lan="$lan" host-bridge="$bridge" mac-cable="$mac_cable" mac-tunnel="10.99.${n:-0}.2")
        [ -z "$tunnel" ] || args+=(host-tunnel="$tunnel" mac-tunnel0=10.99.0.2)
        [ -z "$cable" ] || args+=(host-cable="$cable")
        [ -z "$resolver" ] || args+=(host-resolver="$resolver")
        [ -z "$mac_lan" ] || args+=(mac-lan="$mac_lan")
        for o in $(guests); do
            [ "$o" != "$g" ] || continue
            args+=("guest-$o=$(address_of "$o")" "slot-of-$o=10.99.$(slot_of "$o").1")
            args+=(other-addr="$(address_of "$o")" other-mac="$(incus config get "$o" volatile.eth0.hwaddr)")
        done
        drive "$g" 3 "${args[@]}"
    done
}

ikeward_up() {
    incus exec "$1" -- test -x /usr/local/sbin/ikeward || {
        echo "the guest $1 runs no ikeward"
        return 1
    }
    incus exec "$1" -- /usr/local/sbin/ikeward status
}

check5() {
    local g
    for g in $(guests); do check "5 ikeward's VPN is up in $g" ikeward_up "$g"; done
}

# logged waits up to a minute for SigNoz's ClickHouse to hold a log whose body is the token, under
# the guest's name as host.name.
logged() {
    local q n
    q="SELECT count() FROM signoz_logs.distributed_logs_v2 WHERE body = 'ws-check-$1' AND resources_string['host.name'] = '$2'"
    for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
        n=$(docker exec signoz-telemetrystore-clickhouse-0-0 clickhouse-client --query "$q" 2>&1)
        [ "$n" = 1 ] && return 0
        sleep 5
    done
    echo "SigNoz holds $n records of ws-check-$1 under $2"
    return 1
}

check9() {
    local g token
    for g in $(guests); do
        token=$(cut -c 1-8 /proc/sys/kernel/random/uuid)
        drive "$g" 9 "token=$token"
        check "9 SigNoz shows the record under host.name $g" logged "$token" "$g"
    done
}

# key_of prints the fingerprint of a guest's own SSH host key.
key_of() { incus exec "$1" -- ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub | awk '{ print $2 }'; }

lease() { incus list "$1" --format csv -c 4 | grep -q "^$2 "; }

check11() {
    local g keys=''
    for g in $(guests); do
        check "11 $g took its lease $(address_of "$g")" lease "$g" "$(address_of "$g")"
        check "11 $g is named $g" bash -c "[ \"\$(incus exec $g -- hostname)\" = $g ]"
        check "11 $g resolves a name" incus exec "$g" -- getent hosts github.com
        check "11 $g fetched its secrets" incus exec "$g" -- test -d /var/lib/ws/secrets/.git
        keys+="$(key_of "$g")"$'\n'
    done
    [ "$(printf '%s' "$keys" | grep -c .)" -ge 2 ] || fail "11 several guests run" "set up a second guest"
    [ "$(printf '%s' "$keys" | sort | uniq -d | wc -l)" -eq 0 ] || fail "11 each guest has its own host key" "two guests share one"
}

main() {
    local arg
    [ $# -gt 0 ] || {
        echo "usage: guests.sh CHECK... [guest=NAME...] [MAC_LAN=ADDRESS]" >&2
        return 2
    }
    for arg in "$@"; do
        case $arg in
        guest=*) named+=("${arg#*=}") ;;
        MAC_LAN=*) mac_lan=${arg#*=} ;;
        MAC_CABLE=*) mac_cable=${arg#*=} ;;
        esac
    done
    [ -n "$(guests)" ] || {
        fail "a guest is listed" "the host lists none, or its list failed"
        return 1
    }
    for arg in "$@"; do
        case $arg in
        1) check1 ;;
        2) check2 ;;
        3) check3 ;;
        5) check5 ;;
        9) check9 ;;
        11) check11 ;;
        guest=* | MAC_*=*) ;;
        *)
            echo "guests.sh: no check $arg here" >&2
            return 2
            ;;
        esac
    done
    [ "$fails" -eq 0 ]
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
