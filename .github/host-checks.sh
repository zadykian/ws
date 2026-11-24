#!/bin/bash
# The host job's cases, as root on a runner that booted (.github/workflows/README.md#the-host-job).
# They check the host's setup on a booted machine. It runs twice, and the second changes nothing.
# Fans stay off where the board is not theirs. ufw holds SSH and Docker's rules for its resolver
# and the collector, sshd takes keys alone, and a container resolves a name and sends OTLP. Run as
# `host-checks.sh`, with .tools/install-build's ws in /usr/local/bin and the tree in /opt/ws, as
# server-checks.sh has them.
set -euo pipefail

image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
# The runner's sudo keeps its environment, and its /etc/environment names its own directories, which
# would send root's settings to the runner's home. Its apt takes a variant that no machine has.
unset XDG_CONFIG_HOME
sed -i '/^XDG_/d' /etc/environment
echo '#clear APT::Architecture-Variants;' >>/etc/apt/apt.conf
export HOME=/root DEBIAN_FRONTEND=noninteractive WS_SIDE=host

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

# The image's Docker and its sources are not ws's, as in server-checks.sh: ws installs its own.
rm -rf /usr/libexec/docker/cli-plugins
apt-get purge -y docker-ce docker-ce-cli containerd.io
rm -rf /etc/docker /var/lib/docker /var/lib/containerd
for f in /etc/apt/sources.list.d/*; do
    if [ -f "$f" ] && grep -q -e download.docker.com -e git-core/ppa "$f"; then rm -f "$f"; fi
done
apt-get update

# Root logs in by key, as the installer's host does, so the lockout guard passes.
keys=$(mktemp -d)
ssh-keygen -q -t ed25519 -N '' -f "$keys/key"
install -d -m 700 /root/.ssh
install -m 600 "$keys/key.pub" /root/.ssh/authorized_keys

# The swap file is small, as the runner's disk has no room for one the size of its RAM.
install -d -m 755 /etc/ws
printf '%s\n' --- 'WS_BASE_SWAP_SIZE_MB: 64' >/etc/ws/values.yml
chmod 600 /etc/ws/values.yml
# apt's daily timers keep the stamps of the image's build, which would start an upgrade under the
# bootstrap, as server-checks.sh explains. A new machine has none.
rm -fv /var/lib/systemd/timers/stamp-apt-daily{,-upgrade}.timer

# The tree must be root's alone, as setup's guard needs, and /opt and /usr/share too, for mise's.
chown -R 0:0 /opt/ws
chmod -R go-w /opt/ws
chmod go-w /opt /usr/share
check=$(dirname "$0")/warnings-check.sh
setup() { "$check" -- "$(dirname "$0")/bootstrap.sh" --in-place "$@" </dev/null; }
exits() {
    local want=$1 status=0
    shift
    "$@" || status=$?
    [ "$status" = "$want" ]
}
must "the plan before the first run has changes to make" exits 2 setup --plan
must "the first setup runs" setup
must "the first setup leaves nothing to do" setup --plan
must "the second setup runs" setup
must "the second setup leaves nothing to do" setup --plan

# The runner's board is no fans' board, so the fans' units are off by their fact: no service, no
# config and no driver.
must "no ws-fans service" test ! -e /etc/systemd/system/ws-fans.service
must "no fans config" test ! -e /etc/ws/fans.json
must "no fans driver" bash -c "! (dkms status 2>/dev/null | grep -q nct6687)"

must "sshd takes keys alone" bash -c "sshd -T | grep -qx 'passwordauthentication no'"
must "ufw is on, denying what comes in and allowing what goes out" \
    bash -c "ufw status verbose | grep -E 'Status: active|Default: deny \(incoming\), allow \(outgoing\)' | wc -l | grep -qx 2"
must "ufw lets SSH in" bash -c "ufw status | grep -qE '^22/tcp +ALLOW'"
# ufw's added rules leave out a source of any, so a rule reads `allow in on docker0 to ADDRESS`.
must "ufw lets Docker's networks ask the resolver" \
    bash -c "ufw show added | grep -qF 'ufw allow in on docker0 to 172.17.0.1 port 53'"
must "ufw lets Docker's networks send OTLP to the collector's socket" \
    bash -c "ufw show added | grep -qE 'docker0.*172\.17\.0\.1.*4317'"

must "docker pulls the image" docker pull -q "$image"
must "a container resolves a name through the bridge" docker run --rm "$image" nslookup example.com
must "a container's OTLP reaches the collector on the bridge address" "$(dirname "$0")/otlp-checks.sh" "$image"
