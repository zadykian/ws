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
export HOME=/root DEBIAN_FRONTEND=noninteractive

fail() {
    echo "::error::$*"
    exit 1
}

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

