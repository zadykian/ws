#!/bin/bash
# The rootless job's cases, as root on a runner that booted (.github/workflows/rootless.md). It
# proves what a container can't: the rootless daemon, and the firewall around it. Each case that
# expects a refusal has a control, as a broken setup refuses everything. Run as
# `rootless-checks.sh DIR`, DIR being .tools/serve-build's for the tree in /opt/ws.
set -euo pipefail

rel=$1

user=zadykian
home=/home/$user
image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
target=10.250.0.2:8000
web='while true; do printf "HTTP/1.0 200 OK\r\n\r\nhi\n" | nc -l -p 80; done'
# shellcheck disable=SC2016 # the bash that as starts expands these, not this one.
sudo_form='printf "%s\n" "$1" | sudo -S -p "" -v; sudo -n true; rc=$?; exit "$rc"'
# The runner's sudo keeps the runner's environment, which a real sudo drops, and its
# /etc/environment, which PAM gives each session, names the runner's own directories. They would
# send the user's settings to the runner's home, and the user's systemd to the runner's runtime
# directory.
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

# journal prints the user's daemon's log after a failure, which a failed start only points to.
journal() {
    local status=$?
    [ "$status" = 0 ] || journalctl -n 80 --no-pager _SYSTEMD_USER_UNIT=docker.service || true
    [ "$status" = 0 ] || dmesg | grep -i apparmor | tail -n 20 || true
    return "$status"
}
trap journal EXIT

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

as() {
    runuser -u "$user" -- env -C "$home" HOME="$home" XDG_RUNTIME_DIR="/run/user/$uid" "$@"
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

# Judge the deb's path with Ubuntu's restriction on, as on a real machine, whatever the image sets.
key=/proc/sys/kernel/apparmor_restrict_unprivileged_userns
[ ! -e "$key" ] || echo 1 >"$key"

# The image's Docker predates what --net=host means on a rootless daemon, and ws must install its
# own. The purge leaves docker0, which the route check accepts. The image's source of Docker goes
# too, as apt refuses one repository signed by two keys, and mise adds its own. So do the plugins
# that the image put in Docker's directory past dpkg, which would keep it and warn.
apt-get update
rm -rf /usr/libexec/docker/cli-plugins
apt-get purge -y docker-ce docker-ce-cli containerd.io
rm -rf /etc/docker /var/lib/docker /var/lib/containerd
for f in /etc/apt/sources.list.d/*; do
    if [ -f "$f" ] && grep -q download.docker.com "$f"; then rm -f "$f"; fi
done
apt-get install -y --no-install-recommends netcat-openbsd

# The account as Ubuntu's installer leaves it, with the groups ws must take away.
groupadd --force docker
groupadd --force lxd
useradd --create-home --shell /bin/bash --groups sudo,docker,lxd "$user"
uid=$(id -u "$user")
password=$(openssl rand -hex 16)
echo "::add-mask::$password"
chpasswd <<<"$user:$password"
keys=$(mktemp -d)
ssh-keygen -q -t ed25519 -N '' -f "$keys/key"
install -d -m 700 -o "$user" -g "$user" "$home/.ssh"
install -m 600 -o "$user" -g "$user" "$keys/key.pub" "$home/.ssh/authorized_keys"

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
trap 'kill "$lab_pid"' EXIT

install -d -m 755 /etc/ws
# The swap file is small, as the runner's disk has no room for one the size of its RAM.
printf '%s\n' --- "WS_USER: $user" 'WS_BASE_SWAP_SIZE_MB: 64' >/etc/ws/values.yml
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
must "sudo's password is cached before the drop-in" as bash -c "$sudo_form" bash "$password"

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

groups=$(id -nG "$user")
must "the user keeps sudo" grep -qw sudo <<<"$groups"
refused "the user is in docker" grep -qw docker <<<"$groups"
refused "the user is in lxd" grep -qw lxd <<<"$groups"
refused "sudo's password stays cached" as bash -c "$sudo_form" bash "$password"
refused "the system's socket answers the user" \
    as curl --silent --max-time 5 --unix-socket /var/run/docker.sock http://localhost/_ping

# The pulls ahead of the runs, the user's and root's, as a run of an image not yet pulled warns.
must "the user's docker pulls the image" as docker pull -q "$image"
docker pull -q "$image"
must "the user's docker runs a container" as docker run --rm "$image" true
resolver=$(as docker run --rm "$image" cat /etc/resolv.conf)
must "a container asks slirp4netns's resolver" grep -qx 'nameserver 10.0.2.3' <<<"$resolver"
must "a container reaches the internet" \
    as docker run --rm "$image" wget -qO /dev/null --timeout=10 http://example.com

cgroup=/sys/fs/cgroup/user.slice/user-$uid.slice/user@$uid.service
slice=$(as systemctl --user show --property=Slice --value docker.service)
must "the daemon runs in docker.slice" test "$slice" = docker.slice
must "the user's systemd starts the daemon by its link" as systemctl --user is-enabled --quiet docker.service
must "docker reaches the daemon by the rootless context" test "$(as docker context show)" = rootless
must "the user's systemd has cpuset" grep -qw cpuset "$cgroup/cgroup.controllers"
must "the user's systemd has io" grep -qw io "$cgroup/cgroup.controllers"
id=$(as docker run -d --rm "$image" sleep 60)
must "a container runs inside docker.slice" test -d "$cgroup/docker.slice/docker-$id.scope"
as docker stop -t 0 "$id" >/dev/null

if out=$(as docker run --rm --privileged --net=host -v /:/host "$image" \
    chroot /host /usr/sbin/nft flush ruleset 2>&1); then
    fail "a privileged container flushed the ruleset"
fi
must "the kernel refuses the flush" grep -q 'Operation not permitted' <<<"$out"
ruleset=$(nft list chain inet ws output)
must "inet ws stays" grep -q 'jump private' <<<"$ruleset"

probe='touch /host/etc/ws-probe; touch /host/usr/bin/ws-probe; echo usr=$?
cat /host/etc/shadow >/dev/null; echo shadow=$?'
out=$(as docker run --rm -v /:/host "$image" sh -c "$probe" 2>&1)
echo "$out"
must "a container can't write /usr" grep -qx 'usr=1' <<<"$out"
must "a container can't read shadow" grep -qx 'shadow=1' <<<"$out"
refused "a container wrote the real /etc" test -e /etc/ws-probe
refused "a container wrote the real /usr" test -e /usr/bin/ws-probe

refused "a bridged container reaches a private address" \
    as docker run --rm "$image" wget -qO /dev/null --timeout=5 "http://$target/"
refused "a host-network container reaches a private address" \
    as docker run --rm --net=host "$image" wget -qO /dev/null --timeout=5 "http://$target/"
refused "the user reaches a private address" as curl --silent --fail --max-time 5 "http://$target/"

as docker run -d --rm --name ws-web -p 0.0.0.0:8082:80 "$image" sh -c "$web" >/dev/null
must "a port on 0.0.0.0 answers on the host" up http://127.0.0.1:8082/
refused "a port on 0.0.0.0 answers past ufw" ip netns exec lab curl --silent --fail --max-time 5 \
    --output /dev/null http://10.250.0.1:8082/
ufw --force disable
must "the same port answers with ufw off" ip netns exec lab curl --silent --fail --max-time 5 \
    --output /dev/null http://10.250.0.1:8082/
ufw --force enable
as docker stop -t 0 ws-web >/dev/null

docker run -d --rm --name ws-probe -p 127.0.0.1:18080:80 "$image" sh -c "$web" >/dev/null
address=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ws-probe)
must "root reaches a system container" up "http://$address/"
refused "the user reaches a system container" as curl --silent --fail --max-time 5 \
    --output /dev/null "http://$address/"
must "the user reaches a published port" as curl --silent --fail --max-time 5 \
    --output /dev/null http://127.0.0.1:18080/
docker stop -t 0 ws-probe >/dev/null

# The shell's line refuses a link at ~/.bashrc. A dry run is enough, as the refusal is in the plan.
# useradd made ~/.bashrc, from /etc/skel.
as mv "$home/.bashrc" "$home/.bashrc.orig"
as ln -s /etc/hostname "$home/.bashrc"
if out=$(WS_UNITS=roles/server/ubuntu ws unit write --dry-run file/shell-bashrc-line 2>&1); then
    fail "ws unit write planned through a link at ~/.bashrc"
fi
must "a link at ~/.bashrc is refused" grep -q 'is a link to /etc/hostname' <<<"$out"
as rm "$home/.bashrc"
as mv "$home/.bashrc.orig" "$home/.bashrc"

# ws's secrets: the identity is a credential that the host's key encrypts, which ws unit decrypts
# by systemd-creds as root. A container's systemd can't make one.
ws server ubuntu secrets init
must "the credential is root's 600, in root's 700" test "$(stat -c '%U %a' \
    /etc/credstore.encrypted /etc/credstore.encrypted/ws-secrets)" = $'root 700\nroot 600'
must "the file is root's 600" test "$(stat -c '%U %a' /etc/ws/secrets.age)" = 'root 600'
value=$(openssl rand -hex 16)
echo "::add-mask::$value"
must "set takes a value on stdin" ws server ubuntu secrets set env WS_CI <<<"$value"
names=$(ws server ubuntu secrets keys)
must "keys names the entry" grep -qx 'env WS_CI' <<<"$names"
refused "keys prints the value" grep -q "$value" <<<"$names"
refused "the user runs ws server ubuntu secrets" as ws server ubuntu secrets keys

# The user's decrypt prints nothing, so that one that got through would show no identity in the log.
refused "the user decrypts the credential" as sh -c 'systemd-creds decrypt --name=ws-secrets \
    /etc/credstore.encrypted/ws-secrets - </dev/null >/dev/null 2>&1'
out=$(WS_UNITS=roles/server/ubuntu ws unit write --dry-run file/shell-env 2>&1) ||
    fail "ws unit can't plan /etc/ws/env from the file"
must "ws unit plans /etc/ws/env from the file" grep -q 'would change: file/shell-env' <<<"$out"
refused "ws unit prints the value" grep -q "$value" <<<"$out"

must "unset drops the entry" ws server ubuntu secrets unset env WS_CI
refused "keys names the entry after unset" grep -q WS_CI <<<"$(ws server ubuntu secrets keys)"
sums=$(sha256sum /etc/ws/secrets.age /etc/credstore.encrypted/ws-secrets)
ws server ubuntu secrets init
must "a second init keeps both" test "$sums" = \
    "$(sha256sum /etc/ws/secrets.age /etc/credstore.encrypted/ws-secrets)"
