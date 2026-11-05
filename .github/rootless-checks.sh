#!/bin/bash
# The rootless job's cases, as root on a runner that booted (docs/checks-rootless.md). It proves
# what a container can't: the rootless daemon, and the firewall around it. Each case that expects
# a refusal has a control, as a broken setup refuses everything.
set -euo pipefail

user=zadykian
home=/home/$user
image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
target=10.250.0.2:8000
web='while true; do printf "HTTP/1.0 200 OK\r\n\r\nhi\n" | nc -l -p 80; done'
# shellcheck disable=SC2016 # the bash that as starts expands these, not this one.
sudo_form='printf "%s\n" "$1" | sudo -S -p "" -v; sudo -n true; rc=$?; exit "$rc"'
# The runner's sudo keeps the runner's environment, which a real sudo drops, and its
# /etc/environment, which PAM gives each session, names the runner's own directories. They would
# send Docker's tool to the runner's home, and the user's systemd to the runner's runtime directory.
unset XDG_CONFIG_HOME
sed -i '/^XDG_/d' /etc/environment
export HOME=/root DEBIAN_FRONTEND=noninteractive

fail() {
    echo "::error::$*"
    exit 1
}

# journal prints the user's daemon's log after a failure, which Docker's tool only points to.
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
    if "$@"; then
        fail "$name"
    fi
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
if [ -e "$key" ]; then
    echo 1 >"$key"
fi

# The image's Docker predates what --net=host means on a rootless daemon, and ws must install its
# own. The purge leaves docker0, which the route check accepts.
apt-get update
apt-get purge -y docker-ce docker-ce-cli containerd.io
rm -rf /etc/docker /var/lib/docker /var/lib/containerd
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
printf '%s\n' --- "ws_user: $user" >/etc/ws/host_vars.yml
chmod 600 /etc/ws/host_vars.yml

# The controls: each of these holds before ws, and the opposite after it.
must "the private address answers before the table" up "http://$target/"
must "sudo's password is cached before the drop-in" as bash -c "$sudo_form" bash "$password"

log=$(mktemp)
ws server apply --only security,docker
ws server apply --only security,docker 2>&1 | tee "$log"
if [ "$(tail -n 1 "$log")" != 'ws server apply: changed: none' ]; then
    fail "The second run of ws server apply changed something."
fi

groups=$(id -nG "$user")
must "the user keeps sudo" grep -qw sudo <<<"$groups"
refused "the user is in docker" grep -qw docker <<<"$groups"
refused "the user is in lxd" grep -qw lxd <<<"$groups"
refused "sudo's password stays cached" as bash -c "$sudo_form" bash "$password"
refused "the system's socket answers the user" \
    as curl --silent --max-time 5 --unix-socket /var/run/docker.sock http://localhost/_ping

must "the user's docker runs a container" as docker run --rm "$image" true
resolver=$(as docker run --rm "$image" cat /etc/resolv.conf)
must "a container asks slirp4netns's resolver" grep -qx 'nameserver 10.0.2.3' <<<"$resolver"
must "a container reaches the internet" \
    as docker run --rm "$image" wget -qO /dev/null --timeout=10 http://example.com

cgroup=/sys/fs/cgroup/user.slice/user-$uid.slice/user@$uid.service
slice=$(as systemctl --user show --property=Slice --value docker.service)
must "the daemon runs in docker.slice" test "$slice" = docker.slice
must "the user's systemd has io" grep -qw io "$cgroup/cgroup.controllers"
id=$(as docker run -d --rm "$image" sleep 60)
must "a container runs inside docker.slice" test -d "$cgroup/docker.slice/docker-$id.scope"
as docker stop -t 0 "$id" >/dev/null

