#!/bin/bash
# The server job's cases, as root on a runner that booted (.github/workflows/server.md). It
# proves what a container can't: the bootstrap on a booted machine, and the firewall around Docker.
# Each case that expects a refusal has a control, as a broken setup refuses everything. Run as
# `server-checks.sh DIR`, DIR being .tools/serve-build's for the tree in /opt/ws.
set -euo pipefail

rel=$1

image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
target=10.250.0.2:8000
web='while true; do printf "HTTP/1.0 200 OK\r\n\r\nhi\n" | nc -l -p 80; done'
# The runner's sudo keeps the runner's environment, which a real sudo drops, and its
# /etc/environment, which PAM gives each session, names the runner's own directories. They would
# send root's settings to the runner's home.
unset XDG_CONFIG_HOME
sed -i '/^XDG_/d' /etc/environment
# The runner's apt takes the amd64v3 variant, which a machine turns on by hand. The dotnet PPA's
# index of that variant holds no .NET 8, so the cases run with no variant, as a machine has none.
# apt reads this file after apt.conf.d, so the line wins wherever the runner set the variant.
echo '#clear APT::Architecture-Variants;' >>/etc/apt/apt.conf
export HOME=/root DEBIAN_FRONTEND=noninteractive
# A tool installs only as a lock holds it, here the build's in mise.local.lock, as in playbook. The
# runs by hand put the tools where install.sh does.
export MISE_LOCKED=1 MISE_DATA_DIR=/usr/local/share/mise

fail() {
    echo "::error::$*"
    exit 1
}

# finish ends the lab's listener, which the cases start below, and prints the log on a failure.
finish() {
    local status=$?
    if [ -n "${lab_pid:-}" ]; then kill "$lab_pid" 2>/dev/null || true; fi
    if [ -n "${host_pid:-}" ]; then kill "$host_pid" 2>/dev/null || true; fi
    [ "$status" = 0 ] || journalctl -n 80 --no-pager -u docker.service || true
    return "$status"
}
trap finish EXIT

must() {
    local name=$1
    shift
    "$@" || fail "$name"
    echo "ok: $name"
}

refused() {
    local name=$1
    shift
    ! "$@" || fail "$name"
    echo "ok: $name"
}

fetch() {
    curl --silent --fail --max-time 5 --output /dev/null "$@"
}

# up waits for a port that a container has just published.
up() {
    local try
    for try in $(seq 30); do
        if fetch "$1"; then
            return 0
        fi
        echo "waiting for $1, try $try"
        sleep 1
    done
    return 1
}

# The image's Docker is not ws's, and ws must install its own. The purge leaves docker0, which the
# route check accepts. The image's source of Docker goes too, as apt refuses one repository signed
# by two keys, and mise adds its own. So do the plugins that the image put in Docker's directory
# past dpkg, which would keep it and warn.
apt-get update
rm -rf /usr/libexec/docker/cli-plugins
apt-get purge -y docker-ce docker-ce-cli containerd.io
rm -rf /etc/docker /var/lib/docker /var/lib/containerd
for f in /etc/apt/sources.list.d/*; do
    if [ -f "$f" ] && grep -q download.docker.com "$f"; then rm -f "$f"; fi
done
apt-get install -y --no-install-recommends netcat-openbsd

# Root logs in by key, as a cloud VM's does.
keys=$(mktemp -d)
ssh-keygen -q -t ed25519 -N '' -f "$keys/key"
install -d -m 700 /root/.ssh
install -m 600 "$keys/key.pub" /root/.ssh/authorized_keys

# A private address that answers, in a namespace of its own: the runner has no other.
ip netns add lab
ip link add veth0 type veth peer name veth1
ip link set veth1 netns lab
ip addr add 10.250.0.1/24 dev veth0
ip link set veth0 up
ip netns exec lab ip addr add 10.250.0.2/24 dev veth1
ip netns exec lab ip link set veth1 up
ip netns exec lab ip link set lo up
ip netns exec lab bash -c \
    'while true; do printf "HTTP/1.0 200 OK\r\n\r\nlan\n" | nc -l -N 8000; done' >/dev/null 2>&1 &
lab_pid=$!

install -d -m 755 /etc/ws
# The swap file is small, as the runner's disk has no room for one the size of its RAM.
printf '%s\n' --- 'WS_USER: root' 'WS_BASE_SWAP_SIZE_MB: 64' >/etc/ws/values.yml
chmod 600 /etc/ws/values.yml

# apt's daily timers, which the image turns off, keep the stamps of their runs at its build. On a
# start, systemd takes a stamp from before the boot as a run missed at the boot, and adds the
# timer's RandomizedDelaySec, which each daemon-reload draws again. The bootstrap starts the timers,
# then reloads systemd for each unit it enables. So a draw soon falls in the past, and
# unattended-upgrade holds dpkg's lock under the bootstrap. A new machine's timers have no stamp
# before their first start, so the stamps go.
apt_units=(apt-daily.timer apt-daily-upgrade.timer apt-daily.service apt-daily-upgrade.service)
# apt_state shows what starts an upgrade: the uptime, the stamps, and apt's timers and services.
apt_state() {
    echo "uptime: $(cut -d' ' -f1 /proc/uptime) s"
    ls -l --time-style=+%FT%T /var/lib/systemd/timers/stamp-apt-daily* /var/lib/apt/periodic || true
    systemctl show -p Id,LastTriggerUSec,NextElapseUSecRealtime \
        -p InactiveExitTimestamp,InactiveEnterTimestamp "${apt_units[@]}"
}
apt_state
rm -fv /var/lib/systemd/timers/stamp-apt-daily{,-upgrade}.timer

# apt_quiet holds where no apt command of the log $1 waited for dpkg's lock, and neither of apt's
# daily services ran. The image keeps them off, so a run is the bootstrap's. systemd's lines since
# $2, in seconds since the epoch, show its reloads and a service's start among them.
apt_quiet() {
    local status=0 unit
    apt_state
    journalctl --no-pager -o short-iso-precise --since "@$2" _PID=1 |
        grep -E 'Reloading finished|apt-daily' || true
    if grep -m 5 -e 'Waiting for cache lock' -e 'Could not get lock' "$1"; then
        status=1
    fi
    for unit in apt-daily.service apt-daily-upgrade.service; do
        if [ "$(systemctl show -P InactiveExitTimestampMonotonic "$unit")" != 0 ]; then
            echo "$unit ran"
            status=1
        fi
    done
    return "$status"
}

# The controls: each of these holds before ws, and the opposite after it.
must "the private address answers before the table" up "http://$target/"

# As on a new machine: install.sh runs the bootstrap, which installs ws and sets up the firewall
# before Docker's ufw rules. It runs twice, and the second run must change nothing. The tree must
# be root's alone for install.sh's guard. So must /opt and /usr/share, as on a machine, for mise's:
# the runner's image lets others write both, and mise refuses a system plugin below either.
chown -R 0:0 /opt/ws
chmod -R go-w /opt/ws
chmod go-w /opt /usr/share
log=$(mktemp)
since=$(date +%s)
if ! "$(dirname "$0")/bootstrap.sh" "$rel" > >(tee "$log") 2>&1; then
    apt_quiet "$log" "$since" || true
    fail "the first bootstrap"
fi
must "no upgrade ran, and no apt command waited for dpkg's lock" apt_quiet "$log" "$since"
check=$(dirname "$0")/warnings-check.sh
boot() { "$check" -- mise -C /opt/ws/roles/server bootstrap "$@" </dev/null; }
must "the first bootstrap leaves nothing to do" boot plan --detailed-exitcode
must "the first bootstrap prints no warning" "$check" "$log"
must "the second bootstrap runs" boot
must "the second bootstrap leaves nothing to do" boot plan --detailed-exitcode

# The pulls ahead of the runs, as a run of an image not yet pulled warns.
must "docker pulls the image" docker pull -q "$image"
must "docker runs a container" docker run --rm "$image" true
must "a container reaches the internet" \
    docker run --rm "$image" wget -qO /dev/null --timeout=10 http://example.com

# The firewall must hold under the system daemon. Root's own programs meet the table too.
refused "a bridged container reaches a private address" \
    docker run --rm "$image" wget -qO /dev/null --timeout=5 "http://$target/"
refused "a host-network container reaches a private address" \
    docker run --rm --net=host "$image" wget -qO /dev/null --timeout=5 "http://$target/"
refused "root reaches a private address" curl --silent --fail --max-time 5 "http://$target/"

# ufw judges what a host program listens on, and refuses another network's packets to it. The
# listener stops by its pid, and by its timeout where a case fails first.
timeout 120 bash -c 'while true; do printf "HTTP/1.0 200 OK\r\n\r\nhost\n" | nc -l -N 8084; done' \
    >/dev/null 2>&1 &
host_pid=$!
must "a host listener answers on loopback" up http://127.0.0.1:8084/
refused "a host listener on every address answers on another network" \
    ip netns exec lab curl --silent --fail --max-time 5 --output /dev/null http://10.250.0.1:8084/
ufw --force disable
must "the same listener answers with ufw off" ip netns exec lab curl --silent --fail --max-time 5 \
    --output /dev/null http://10.250.0.1:8084/
ufw --force enable
kill "$host_pid"

# A port published with no address goes on loopback, as Docker's ports get past ufw
# (roles/server/ubuntu/docker/README.md#published-ports). It answers on the host, not on the
# address of another network. The control publishes on every address, which the lab reaches.
docker run -d --rm --name ws-web -p 0.0.0.0:8083:80 "$image" sh -c "$web" >/dev/null
must "a port published on every address answers on loopback" up http://127.0.0.1:8083/
must "a port published on every address answers on another network" \
    ip netns exec lab curl --silent --fail --max-time 5 --output /dev/null http://10.250.0.1:8083/
docker stop -t 0 ws-web >/dev/null
docker run -d --rm --name ws-web -p 8082:80 "$image" sh -c "$web" >/dev/null
must "a port published with no address answers on loopback" up http://127.0.0.1:8082/
refused "a port published with no address answers on another network" \
    ip netns exec lab curl --silent --fail --max-time 5 --output /dev/null http://10.250.0.1:8082/
docker stop -t 0 ws-web >/dev/null

docker run -d --rm --name ws-probe -p 127.0.0.1:18080:80 "$image" sh -c "$web" >/dev/null
address=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ws-probe)
must "root reaches a container's address" up "http://$address/"
must "root reaches a published port" fetch http://127.0.0.1:18080/
docker stop -t 0 ws-probe >/dev/null

# The shell's line refuses a link at ~/.bashrc. A dry run is enough, as the refusal is in the plan.
mv /root/.bashrc /root/.bashrc.orig
ln -s /etc/hostname /root/.bashrc
if out=$(WS_UNITS=roles/server/ubuntu ws unit write --dry-run file/shell-bashrc-line 2>&1); then
    fail "ws unit write planned through a link at ~/.bashrc"
fi
must "a link at ~/.bashrc is refused" grep -q 'is a link to /etc/hostname' <<<"$out"
rm /root/.bashrc
mv /root/.bashrc.orig /root/.bashrc
