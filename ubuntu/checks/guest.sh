#!/bin/bash
# The real machine's checks that run inside a guest, as root: ubuntu/checks/README.md. Run as
#
#   guest.sh CHECK... [NAME=ADDRESS...]
#
# CHECK is 1 (Docker and a compose stack), 2 (the GPU's CUDA), 3 (what drops) or 9 (OTLP from a
# container, whose record carries token=TOKEN). Check 3 probes each NAME=ADDRESS, which must drop.
# Then the internet must pass. Then a packet under another guest's address and MAC must drop:
# other-addr=ADDRESS and other-mac=MAC. It prints a line a check, ok or FAIL, and exits 1 where any
# fails. guests.sh pushes it to a guest, runs it and removes it. It changes nothing but the address
# and MAC that check 3 borrows, which it puts back.
set -uo pipefail

image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
internet=9.9.9.9
fails=0
work=
nic=eth0
mac_before=
addr_added=

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

# dropped succeeds where nothing answers at ADDRESS: no ping, and no TCP port that connects or
# refuses within two seconds, as a refusal is an answer too. A timeout is a drop.
dropped() {
    local addr=$1 port
    if ping -c 1 -W 2 -n "$addr" >/dev/null 2>&1; then
        echo "$addr answers a ping"
        return 1
    fi
    for port in 22 53 80 443 4317 14317; do
        timeout 2 bash -c ": >/dev/tcp/$addr/$port" 2>/dev/null
        if [ $? -ne 124 ]; then
            echo "$addr answers TCP $port"
            return 1
        fi
    done
}

# Check 1: rootful Docker and a compose stack, in an unprivileged, nested guest on ZFS.
docker_stack() {
    cat >"$work/compose.yaml" <<YAML
services:
  web:
    image: $image
    command: sh -c 'mkdir -p /www && echo ws-check >/www/index.html && exec httpd -f -p 8080 -h /www'
  client:
    image: $image
    depends_on: [web]
    command: sh -c 'sleep 2 && wget -qO- http://web:8080/'
YAML
    local out
    out=$(docker compose -f "$work/compose.yaml" -p ws-check up --abort-on-container-exit --exit-code-from client 2>&1)
    local status=$?
    docker compose -f "$work/compose.yaml" -p ws-check down -v >/dev/null 2>&1
    printf '%s\n' "$out" | tail -n 3
    [ $status -eq 0 ] && printf '%s' "$out" | grep -q ws-check
}

unprivileged() {
    [ "$(systemd-detect-virt -c)" = lxc ] && ! grep -qx " *0 *0 *4294967295" /proc/self/uid_map
}

on_zfs() { [ "$(findmnt -no FSTYPE /)" = zfs ]; }

# overlay prints Docker's storage driver. It succeeds for overlay2 and for overlayfs.
overlay() {
    local driver
    driver=$(docker info --format '{{.Driver}}') || return 1
    echo "the driver is $driver"
    [ "$driver" = overlay2 ] || [ "$driver" = overlayfs ]
}

check1() {
    check "1 the guest is a container, unprivileged" unprivileged
    check "1 its root is on ZFS" on_zfs
    check "1 Docker runs, on overlay2 or overlayfs" overlay
    check "1 a container runs" docker run --rm "$image" true
    check "1 a compose stack's services reach each other" docker_stack
}

# cuda prints how many devices the driver API sees, by libcuda, which the userspace package holds.
cuda() {
    python3 - <<'PY'
import ctypes, sys
lib = ctypes.CDLL("libcuda.so.1")
if lib.cuInit(0) != 0:
    sys.exit("cuInit failed")
n = ctypes.c_int()
if lib.cuDeviceGetCount(ctypes.byref(n)) != 0 or n.value < 1:
    sys.exit("no CUDA device")
print(n.value, "CUDA device")
PY
}

gpu_listed() { nvidia-smi -L | grep -q '^GPU 0'; }

# same_driver succeeds where nvidia-smi's driver is the host's module's version, which the guest reads.
same_driver() {
    [ "$(nvidia-smi --query-gpu=driver_version --format=csv,noheader)" = "$(cat /sys/module/nvidia/version)" ]
}

check2() {
    check "2 nvidia-smi lists the GPU" gpu_listed
    check "2 CUDA initialises" cuda
    check "2 the guest's driver is the host's" same_driver
}

restore() {
    [ -z "$addr_added" ] || ip addr del "$addr_added" dev "$nic" 2>/dev/null
    [ -z "$mac_before" ] || ip link set "$nic" address "$mac_before"
    [ -z "$work" ] || rm -rf "$work"
}

# spoof sends from the NIC under a source that is not its own: a ping to the internet that a table
# of sources or the MAC filter must drop.
spoofed_addr() {
    local own
    own=$(ip -4 -o addr show dev "$nic" scope global | awk 'NR == 1 { split($4, a, "/"); print a[1] }')
    ip addr add "$1/24" dev "$nic" || return 1
    addr_added="$1/24"
    local status=0
    ping -c 2 -W 2 -I "$1" "$internet" >/dev/null 2>&1 && status=1
    ip addr del "$addr_added" dev "$nic"
    addr_added=
    [ $status -eq 0 ] || echo "a packet from $1 reached $internet, as $own's own"
    return $status
}

spoofed_mac() {
    mac_before=$(cat "/sys/class/net/$nic/address")
    ip link set "$nic" address "$1" || return 1
    sleep 1
    local status=0
    ping -c 2 -W 2 "$internet" >/dev/null 2>&1 && status=1
    ip link set "$nic" address "$mac_before"
    mac_before=
    [ $status -eq 0 ] || echo "a frame under $1 reached $internet"
    return $status
}

check3() {
    local arg other_addr='' other_mac=''
    for arg in "$@"; do
        case $arg in
        other-addr=*) other_addr=${arg#*=} ;;
        other-mac=*) other_mac=${arg#*=} ;;
        *=*) check "3 $arg drops" dropped "${arg#*=}" ;;
        esac
    done
    check "3 the internet passes" ping -c 2 -W 3 "$internet"
    check "3 a name resolves" getent hosts github.com
    check "3 HTTPS reaches GitHub" curl -fsS --max-time 15 -o /dev/null https://github.com
    [ -z "$other_addr" ] || check "3 a packet under another guest's address drops" spoofed_addr "$other_addr"
    [ -z "$other_mac" ] || check "3 a frame under another guest's MAC drops" spoofed_mac "$other_mac"
}

# hex turns the hex digits of $1 into the text of printf's \x escapes.
hex() {
    local s=$1 i
    for ((i = 0; i < ${#s}; i += 2)); do printf '\\x%s' "${s:i:2}"; done
}

# otlp_export sends one log record by OTLP/gRPC from a container to Docker's bridge address, the
# collector's socket there, with the token as its body. The frame is built by hand: a flag byte, the
# length, and the request's protobuf, whose every length is under 128 and takes one byte.
otlp_export() {
    local bridge text body time le="" i n any record scope resource request
    bridge=$(docker network inspect -f '{{(index .IPAM.Config 0).Gateway}}' bridge) || return 1
    text="ws-check-$1"
    body=$(printf '%s' "$text" | od -An -v -tx1 | tr -d ' \n')
    n=${#text}
    time=$(date +%s%N)
    for ((i = 0; i < 8; i++)); do le+=$(printf '%02x' $(((time >> (8 * i)) & 255))); done
    any=$(printf '0a%02x%s' "$n" "$body")
    record=$(printf '09%s2a%02x%s' "$le" $((n + 2)) "$any")
    scope=$(printf '12%02x%s' $((${#record} / 2)) "$record")
    resource=$(printf '12%02x%s' $((${#scope} / 2)) "$scope")
    request=$(printf '0a%02x%s' $((${#resource} / 2)) "$resource")
    printf '%b' "$(hex "00000000$(printf '%02x' $((${#request} / 2)))$request")" |
        docker run --rm -i "$image" sh -ec '
            apk add -q --no-cache curl
            curl --silent --show-error --max-time 10 --http2-prior-knowledge \
                --header "content-type: application/grpc" --header "te: trailers" --data-binary @- \
                --dump-header - --output /dev/null \
                "http://$1:4317/opentelemetry.proto.collector.logs.v1.LogsService/Export" |
                grep -qiE "^grpc-status: 0[[:space:]]*$"' sh "$bridge"
}

no_intake() { ! ss -Hltn | awk '{ print $4 }' | grep -q ':14317$'; }

socket_owner() { [ "$(stat -c '%U %a' /var/lib/ws/otlp/signoz.sock)" = 'otelcol-contrib 600' ]; }

check9() {
    local arg token=
    for arg in "$@"; do
        case $arg in token=*) token=${arg#*=} ;; esac
    done
    [ -n "$token" ] || token=$(cut -c 1-8 /proc/sys/kernel/random/uuid)
    check "9 a container's log record is taken at the bridge address on 4317" otlp_export "$token"
    check "9 nothing in this guest listens on 14317" no_intake
    check "9 the collector's socket is its own, mode 600" socket_owner
}

main() {
    local arg
    [ $# -gt 0 ] || {
        echo "usage: guest.sh CHECK... [NAME=ADDRESS...]" >&2
        exit 2
    }
    work=$(mktemp -d)
    trap restore EXIT
    for arg in "$@"; do
        case $arg in
        1) check1 ;;
        2) check2 ;;
        3) check3 "$@" ;;
        9) check9 "$@" ;;
        *=*) ;;
        *)
            echo "guest.sh: no check $arg" >&2
            exit 2
            ;;
        esac
    done
    [ "$fails" -eq 0 ]
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
