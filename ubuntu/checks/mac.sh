#!/bin/sh
# The real machine's checks that run from the Mac, as its user: ubuntu/checks/README.md. Run as
#
#   mac.sh 4 NAME...
#   mac.sh 5 ADDRESS [reach|refuse]
#
# Check 4 takes each guest NAME, which ssh knows as a host. It shows the tunnel's way in use, tries
# a login by key, and reads the address that sshd in the guest saw. Run it at each place the Mac
# connects from: docked at the cable, on the home LAN, and on a phone's hotspot, which reaches the
# router's forward. Check 5 pings ADDRESS, an address of the VPN's network, which the guest's tunnel
# carries. The ping must reach it. With refuse, it must not: that is a peer that
# WS_IKEWARD_FORWARD_FROM leaves out. It prints a line a check, ok or FAIL, and exits 1 where any
# fails. It changes nothing.
set -u

fails=0

pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    fails=$((fails + 1))
}

# check runs the command and prints ok where it succeeds, else FAIL with the last line it printed.
check() {
    name=$1
    shift
    if out=$("$@" 2>&1); then pass "$name"; else fail "$name" "$(printf '%s' "$out" | tail -n 1)"; fi
}

way() { sudo /usr/local/bin/ws macos tunnel status --server "$1"; }

login() { ssh -o BatchMode=yes -o ConnectTimeout=15 "$1" true; }

# seen_from succeeds where the guest's last login came from 10.99.N.2, the Mac's end of its tunnel.
# N is the slot of the guest's own 10.99.N.1. A login from the host's address fails it.
seen_from() {
    slot=$(ssh -o BatchMode=yes "$1" "ip -4 -o addr show dev eth0" | sed -n 's|.* 10\.99\.\([0-9]*\)\.1/32 .*|\1|p' | head -n 1)
    [ -n "$slot" ] || {
        echo "$1 holds no 10.99.N.1/32 on eth0"
        return 1
    }
    from=$(ssh -o BatchMode=yes "$1" "journalctl -u ssh -n 200 --no-pager | grep 'Accepted publickey' | tail -n 1" |
        sed -n 's/.* from \([0-9.]*\) port .*/\1/p')
    [ "$from" = "10.99.$slot.2" ] || {
        echo "sshd saw $from, not 10.99.$slot.2"
        return 1
    }
}

jail_up() { ssh -o BatchMode=yes "$1" "fail2ban-client status sshd" | grep -q 'Currently banned'; }

check4() {
    [ $# -gt 0 ] || {
        echo "mac.sh 4 needs a guest's name" >&2
        exit 2
    }
    for g in "$@"; do
        check "4 the tunnel to $g has a way in use" way "$g"
        printf 'info %s\n' "$(way "$g" 2>/dev/null | head -n 3 | tr '\n' ';')"
        check "4 ssh logs in to $g by key" login "$g"
        check "4 sshd of $g saw the Mac's tunnel address" seen_from "$g"
        check "4 fail2ban of $g runs its sshd jail" jail_up "$g"
    done
}

check5() {
    addr=$1
    mode=${2:-reach}
    if ping -c 3 -t 6 "$addr" >/dev/null 2>&1; then reached=yes; else reached=no; fi
    printf 'info %s\n' "$(route -n get "$addr" 2>/dev/null | sed -n 's/ *interface: //p')"
    case $mode in
    reach)
        if [ "$reached" = yes ]; then pass "5 the Mac reaches $addr"; else fail "5 the Mac reaches $addr" "no reply"; fi
        ;;
    refuse)
        if [ "$reached" = no ]; then pass "5 the Mac does not reach $addr"; else fail "5 the Mac does not reach $addr" "it answered"; fi
        ;;
    *)
        echo "mac.sh 5: $mode is neither reach nor refuse" >&2
        exit 2
        ;;
    esac
}

case ${1:-} in
4)
    shift
    check4 "$@"
    ;;
5)
    shift
    [ $# -ge 1 ] || {
        echo "mac.sh 5 needs an address" >&2
        exit 2
    }
    check5 "$@"
    ;;
*)
    echo "usage: mac.sh 4 NAME... | mac.sh 5 ADDRESS [reach|refuse]" >&2
    exit 2
    ;;
esac
[ "$fails" -eq 0 ]
