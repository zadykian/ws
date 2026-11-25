#!/bin/sh
# Writes the stand-ins for the tools that the checks call into DIR, which checks-test.sh puts first
# on its PATH. Each answers as the real tool does, from the files and the variables the test sets,
# and none calls the machine's own tool. Run as
#
#   stand-ins.sh DIR
set -eu

bin=$1

# ufw prints the status that UFW_STATUS holds.
cat >"$bin/ufw" <<'STUB'
#!/bin/sh
cat "$UFW_STATUS"
STUB

# ip answers for a tunnel of slot 12, which routes by table 1012, for the host's uplink enp5s0, and
# for a guest's NIC eth0. What a call changes it keeps in IP_STATE. A call it does not know fails.
cat >"$bin/ip" <<'STUB'
#!/bin/sh
case "$*" in
"-4 rule show") echo "32000: from all iif wg12 lookup 1012" ;;
"-4 route show table 1012") echo "default via 10.100.0.12 dev incusbr0 onlink" ;;
"-4 route show default") echo "default via 192.168.1.1 dev enp5s0 proto dhcp src 192.168.1.10 metric 100" ;;
"-4 -o addr show dev enp5s0")
    printf '%s\n' '2: enp5s0    inet 192.168.1.10/24 metric 100 brd 192.168.1.255 scope global dynamic enp5s0\       valid_lft 85841sec preferred_lft 85841sec'
    ;;
"-4 -o addr show dev eth0 scope global")
    printf '%s\n' '2: eth0    inet 10.100.0.10/24 metric 1024 brd 10.100.0.255 scope global dynamic eth0\       valid_lft 3591sec preferred_lft 3591sec'
    ;;
"addr add "*" dev eth0") : >"$IP_STATE/added" ;;
"addr del "*" dev eth0") rm -f "$IP_STATE/added" ;;
"neigh flush dev eth0") : >"$IP_STATE/flushed" ;;
"route add "*" via "*)
    [ ! -e "$IP_STATE/route-$3" ] || {
        echo "RTNETLINK answers: File exists" >&2
        exit 2
    }
    echo "$5" >"$IP_STATE/route-$3"
    ;;
"route delete "*" via "*) rm "$IP_STATE/route-$3" ;;
"route get "*)
    via=192.168.1.1
    [ ! -e "$IP_STATE/route-$3" ] || via=$(cat "$IP_STATE/route-$3")
    printf '%s via %s dev eth0 src 192.168.1.20 uid 1000 \n    cache \n' "$3" "$via"
    ;;
*)
    echo "ip: the stand-in knows no call $*" >&2
    exit 1
    ;;
esac
STUB

# ss prints the listing that SS_LISTING holds. wg names the tunnels' ports, and refuses a user who
# is not root, as WG_DENIED stands for.
cat >"$bin/ss" <<'STUB'
#!/bin/sh
cat "$SS_LISTING"
STUB
cat >"$bin/wg" <<'STUB'
#!/bin/sh
[ "$*" = "show all listen-port" ] || exit 1
[ -z "${WG_DENIED:-}" ] || {
    echo "Unable to access interface: Operation not permitted" >&2
    exit 1
}
printf 'wg0\t51820\nwg12\t51832\n'
STUB

# A guest's ping under a borrowed source has its answer where the table of sources passes it and
# the host took that source for this guest's MAC. The host does so when the guest, its neighbour
# cache flushed, asks for the gateway's MAC under that source. Any other ping has no answer.
cat >"$bin/ping" <<'STUB'
#!/bin/sh
src=
while [ $# -gt 1 ]; do
    case $1 in
    -I)
        src=$2
        shift 2
        ;;
    -c | -W) shift 2 ;;
    *) shift ;;
    esac
done
[ -n "$src" ] && [ "${SOURCES:-}" = open ] || exit 1
[ ! -e "$IP_STATE/flushed" ] || {
    rm "$IP_STATE/flushed"
    : >"$IP_STATE/mine-$src"
}
[ -e "$IP_STATE/mine-$src" ]
STUB

# sudo either runs the command or wants a password that the run cannot give. The machine is Linux,
# and of the host at 192.168.1.10 nc reaches SSH alone.
cat >"$bin/sudo" <<'STUB'
#!/bin/sh
[ "${SUDO:-}" = no ] || exec "$@"
echo "sudo: a terminal is required to read the password" >&2
exit 1
STUB
printf '#!/bin/sh\necho Linux\n' >"$bin/uname"
printf '#!/bin/sh\n[ "$*" = "-z -w 3 192.168.1.10 22" ]\n' >"$bin/nc"

# nvidia-smi runs until the test kills it. systemctl names ws-fans' cgroup.
printf '#!/bin/sh\nwhile :; do sleep 1; done\n' >"$bin/nvidia-smi"
printf '#!/bin/sh\necho /ws-fans.slice/ws-fans.service\n' >"$bin/systemctl"

# incus pushes a file and runs a guest's script as PUSH and RUN say. ws lists no guest.
cat >"$bin/incus" <<'STUB'
#!/bin/sh
case "$*" in
"file push"*) exit "${PUSH:-0}" ;;
*bash*) echo "ok   fine" && exit "${RUN:-0}" ;;
esac
STUB
printf '#!/bin/sh\necho 1\n' >"$bin/docker"
printf '#!/bin/sh\n' >"$bin/ws"

chmod +x "$bin"/*
