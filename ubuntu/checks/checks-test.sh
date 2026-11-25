#!/bin/bash
# Tries the functions of host.sh, guests.sh and guest.sh, and lan.sh whole, against the stand-ins
# for the tools they call that stand-ins.sh writes. CI's lint job runs it. The scripts themselves
# run on the real machine alone.
set -uo pipefail

here=$(cd "$(dirname "$0")" && pwd)
dir=${CHECKS_DIR:-$here}
tmp=$(mktemp -d)
mkdir "$tmp/bin"
"$here/stand-ins.sh" "$tmp/bin" || exit 1
pid=
trap '[ -z "$pid" ] || kill "$pid" 2>/dev/null; rm -rf "$tmp"' EXIT
export PATH="$tmp/bin:$PATH"
failed=0

# expect NAME WANT COMMAND... runs the command. WANT is ok for an exit status of 0, else fail.
expect() {
    local name=$1 want=$2 got=ok
    shift 2
    "$@" >"$tmp/out" 2>&1 || got=fail
    if [ "$got" != "$want" ]; then
        echo "checks-test: $name: wanted $want, got $got" >&2
        sed 's/^/    /' "$tmp/out" >&2
        failed=1
    fi
}

# Each case sources the script, which runs nothing then, in a shell of its own.
host() { (
    # shellcheck disable=SC1091 # the path is the test's variable
    . "$dir/host.sh"
    "$@"
); }
guest_side() { (
    # shellcheck disable=SC1091 # the path is the test's variable
    . "$dir/guests.sh"
    "$@"
); }
guest() { (
    # shellcheck disable=SC1091 # the path is the test's variable
    . "$dir/guest.sh"
    "$@"
); }

# ufw prints a number below 10 padded, as "[ 1]".
cat >"$tmp/ufw-first" <<'UFW'
Status: active

     To                         Action      From
     --                         ------      ----
[ 1] Anywhere on wg1            DENY IN     Anywhere
[ 2] 22/tcp                     ALLOW IN    Anywhere
[ 3] 51821/udp                  ALLOW IN    Anywhere
[ 4] Anywhere (v6) on wg1       DENY IN     Anywhere (v6)
[ 5] 22/tcp (v6)                ALLOW IN    Anywhere (v6)
UFW
cat >"$tmp/ufw-late" <<'UFW'
Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] Anywhere on wg1            DENY IN     Anywhere
[10] Anywhere on wg10           DENY IN     Anywhere
UFW

UFW_STATUS=$tmp/ufw-first expect "deny of wg1 stands before SSH" ok host deny_first 1
UFW_STATUS=$tmp/ufw-late expect "deny of wg1 stands after SSH" fail host deny_first 1
UFW_STATUS=$tmp/ufw-late expect "deny of wg10 stands after SSH" fail host deny_first 10
UFW_STATUS=$tmp/ufw-late expect "wg1's deny is not wg10's" fail host deny_first 11
export UFW_STATUS=$tmp/ufw-first

# A tunnel of slot 12 routes by table 1012, as the code's 1000 + slot. What ip changes it keeps
# in IP_STATE.
export IP_STATE=$tmp/ip
slot12() {
    # shellcheck disable=SC2329 # forwarding calls it
    slot_of() { echo 12; }
    forwarding g
}
expect "a tunnel of slot 12 forwards by table 1012" ok host slot12

# ss lists the kernel's WireGuard sockets on every address, with no process, as any other.
cat >"$tmp/ss-host" <<'SS'
udp   UNCONN 0      0             127.0.0.54:53           0.0.0.0:*
udp   UNCONN 0      0          127.0.0.53%lo:53           0.0.0.0:*
udp   UNCONN 0      0             10.100.0.1:53           0.0.0.0:*
udp   UNCONN 0      0       0.0.0.0%incusbr0:67           0.0.0.0:*
udp   UNCONN 0      0    192.168.1.10%enp5s0:68           0.0.0.0:*
udp   UNCONN 0      0                0.0.0.0:51820        0.0.0.0:*
udp   UNCONN 0      0                0.0.0.0:51832        0.0.0.0:*
udp   UNCONN 0      0                   [::]:51820           [::]:*
udp   UNCONN 0      0                   [::]:51832           [::]:*
tcp   LISTEN 0      4096          127.0.0.54:53           0.0.0.0:*
tcp   LISTEN 0      4096       127.0.0.53%lo:53           0.0.0.0:*
tcp   LISTEN 0      32            10.100.0.1:53           0.0.0.0:*
tcp   LISTEN 0      4096             0.0.0.0:22           0.0.0.0:*
tcp   LISTEN 0      4096                [::]:22              [::]:*
SS
{ cat "$tmp/ss-host" && echo "udp   UNCONN 0      0                0.0.0.0:5353         0.0.0.0:*"; } >"$tmp/ss-mdns"
{ cat "$tmp/ss-host" && echo "tcp   LISTEN 0      4096             0.0.0.0:51820        0.0.0.0:*"; } >"$tmp/ss-tcp"
# listens prints what listens on every address beyond SSH, DHCP and the tunnels, and fails on any.
listens() {
    local out
    out=$(public_listeners)
    printf '%s\n' "$out"
    [ -z "$out" ]
}
SS_LISTING=$tmp/ss-host expect "the tunnels' UDP ports listen on every address" ok host listens
SS_LISTING=$tmp/ss-mdns expect "a UDP port that is no tunnel's" fail host listens
SS_LISTING=$tmp/ss-tcp expect "TCP on a tunnel's port" fail host listens
SS_LISTING=$tmp/ss-host WG_DENIED=1 expect "wg that names no port" fail host listens

# Another guest's address drops, or passes and draws the reply.
mkdir "$IP_STATE"
SOURCES=closed expect "a table of sources that drops another guest's address" ok guest spoofed_addr 10.100.0.11
rm -f "$IP_STATE"/*
SOURCES=open expect "a table that passes another guest's address" fail guest spoofed_addr 10.100.0.11
[ ! -e "$IP_STATE/added" ] || {
    echo "checks-test: spoofed_addr left 10.100.0.11 on eth0" >&2
    failed=1
}
rm -f "$IP_STATE"/*

# lan runs lan.sh whole, against the host 192.168.1.10. It fails where the script leaves a route
# it added, or removes one it found.
lan() {
    local before=no after=no
    [ ! -e "$IP_STATE/route-10.100.0.10" ] || before=yes
    sh "$dir/lan.sh" 192.168.1.10 10.100.0.10 || return 1
    [ ! -e "$IP_STATE/route-10.100.0.10" ] || after=yes
    [ "$before" = "$after" ] || echo "the route to the guest was there $before, and $after after"
    [ "$before" = "$after" ]
}
expect "a guest that answers nothing through the host" ok lan
SUDO=no expect "a route that sudo cannot add" fail lan
echo 192.168.1.10 >"$IP_STATE/route-10.100.0.10"
expect "a route through the host that was there" ok lan
rm -f "$IP_STATE"/*

# ws-fans' cgroup holds a process named nvidia-smi, or not.
"$tmp/bin/nvidia-smi" &
pid=$!
mkdir -p "$tmp/cg/ws-fans.slice/ws-fans.service"
export CGROUP_ROOT=$tmp/cg
: >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs no nvidia-smi" fail host fans_nvidia
echo "$pid" >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs an nvidia-smi" ok host fans_nvidia
echo "$$" >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs another program" fail host fans_nvidia

# The journal holds ws-fans' lines of an earlier run and of this one.
earlier=0a1b2c3d4e5f60718293a4b5c6d7e8f9
this=5f0c3e7a9d2b4c1e8f6a0b3d7c9e2f41
nvml="no NVML: Failed to initialize NVML: Driver/library version mismatch"
{
    printf '%s\t%s\n' "$earlier" "3 fans, every 2 s" "$earlier" "gpu: $nvml"
    printf '%s\t%s\n' "$this" "3 fans, every 2 s" "$this" "gpu: no reading of GPU 0 in 10s"
    printf '%s\t%s\n' "$this" "GPU Fans: no reading of GPU 0 in 10s; the fan goes back to the board's curve"
    printf '%s\t%s\n' "$this" "gpu: back" "$this" "GPU Fans: back"
} >"$tmp/journal-back"
{
    printf '%s\t%s\n' "$this" "3 fans, every 2 s" "$this" "gpu: $nvml"
    printf '%s\t%s\n' "$this" "GPU Fans: $nvml; the fan goes back to the board's curve"
} >"$tmp/journal-nvml"
printf '%s\t%s\n' "$this" "3 fans, every 2 s" "$this" "gpu: no NVML: nvidia-smi ended" >"$tmp/journal-ended"
printf '%s\t%s\n' "$earlier" "3 fans, every 2 s" >"$tmp/journal-none"
JOURNAL=$tmp/journal-back expect "ws-fans' problems of this run, each back" ok host fans_problems
JOURNAL=$tmp/journal-nvml expect "ws-fans with no NVML" fail host fans_problems
JOURNAL=$tmp/journal-ended expect "ws-fans whose nvidia-smi ended" fail host fans_problems
JOURNAL=$tmp/journal-none expect "a journal with no line of this run" fail host fans_problems

# A guest's run is a failure where incus fails or the script exits non-zero, with no FAIL line.
# fails_after runs a check on guest g1 and prints the count of failures.
fails_after() {
    # shellcheck disable=SC2329 # the checks call it
    guests() { echo g1; }
    "$@" >/dev/null 2>&1
    # shellcheck disable=SC2154 # guests.sh sets it
    [ "$fails" -gt 0 ]
}
expect "a guest's run that passes" fail guest_side fails_after check1
expect "a run of check 9 that passes" fail guest_side fails_after check9
PUSH=1 expect "a push that fails" ok guest_side fails_after check1
RUN=1 expect "a run that exits 1 with no FAIL line" ok guest_side fails_after check1
RUN=1 expect "the same on check 9" ok guest_side fails_after check9
no_guest() {
    # shellcheck disable=SC2034 # guests uses it
    ws=$tmp/bin/ws
    main 1
}
expect "a host that lists no guest" fail guest_side no_guest

[ "$failed" -eq 0 ] || exit 1
echo "checks-test: ok"
