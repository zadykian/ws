#!/bin/bash
# Tries the functions of host.sh and guests.sh against stand-ins for ufw, ip, systemctl and incus,
# on the outputs the real tools print. CI's lint job runs it. The scripts themselves run on the
# real machine alone.
set -uo pipefail

dir=${CHECKS_DIR:-$(cd "$(dirname "$0")" && pwd)}
tmp=$(mktemp -d)
mkdir "$tmp/bin"
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
cat >"$tmp/bin/ufw" <<'STUB'
#!/bin/sh
cat "$UFW_STATUS"
STUB
chmod +x "$tmp/bin/ufw"

UFW_STATUS=$tmp/ufw-first expect "deny of wg1 stands before SSH" ok host deny_first 1
UFW_STATUS=$tmp/ufw-late expect "deny of wg1 stands after SSH" fail host deny_first 1
UFW_STATUS=$tmp/ufw-late expect "deny of wg10 stands after SSH" fail host deny_first 10
UFW_STATUS=$tmp/ufw-late expect "wg1's deny is not wg10's" fail host deny_first 11
export UFW_STATUS=$tmp/ufw-first

# A tunnel of slot 12 routes by table 1012, as the code's 1000 + slot.
cat >"$tmp/bin/ip" <<'IP'
#!/bin/sh
case "$*" in
"-4 rule show") echo "32000: from all iif wg12 lookup 1012" ;;
"-4 route show table 1012") echo "default via 10.100.0.12 dev incusbr0 onlink" ;;
esac
IP
chmod +x "$tmp/bin/ip"
slot12() {
    # shellcheck disable=SC2329 # forwarding calls it
    slot_of() { echo 12; }
    forwarding g
}
expect "a tunnel of slot 12 forwards by table 1012" ok host slot12

# ws-fans' cgroup holds a process named nvidia-smi, or not.
printf '#!/bin/sh\nwhile :; do sleep 1; done\n' >"$tmp/bin/nvidia-smi"
chmod +x "$tmp/bin/nvidia-smi"
"$tmp/bin/nvidia-smi" &
pid=$!
printf '#!/bin/sh\necho /ws-fans.slice/ws-fans.service\n' >"$tmp/bin/systemctl"
chmod +x "$tmp/bin/systemctl"
mkdir -p "$tmp/cg/ws-fans.slice/ws-fans.service"
export CGROUP_ROOT=$tmp/cg
: >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs no nvidia-smi" fail host fans_nvidia
echo "$pid" >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs an nvidia-smi" ok host fans_nvidia
echo "$$" >"$tmp/cg/ws-fans.slice/ws-fans.service/cgroup.procs"
expect "ws-fans runs another program" fail host fans_nvidia

# A guest's run is a failure where incus fails or the script exits non-zero, with no FAIL line.
cat >"$tmp/bin/incus" <<'STUB'
#!/bin/sh
case "$*" in
"file push"*) exit "${PUSH:-0}" ;;
*bash*) echo "ok   fine" && exit "${RUN:-0}" ;;
esac
STUB
chmod +x "$tmp/bin/incus"
printf '#!/bin/sh\necho 1\n' >"$tmp/bin/docker"
chmod +x "$tmp/bin/docker"
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
printf '#!/bin/sh\n' >"$tmp/bin/ws"
chmod +x "$tmp/bin/ws"
expect "a host that lists no guest" fail guest_side no_guest

[ "$failed" -eq 0 ] || exit 1
echo "checks-test: ok"
