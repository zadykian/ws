#!/bin/bash
# The server job's cases, as root on a runner that booted (.github/workflows/server.md). They check
# the setup on a booted machine, what it wrote, Docker's forward policy and its published ports.
# Each case that expects a refusal has a control, as a broken setup refuses everything. Run as
# `server-checks.sh`, with .tools/install-build's ws in /usr/local/bin and the tree in /opt/ws.
set -euo pipefail

image=alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
web='while true; do printf "HTTP/1.0 200 OK\r\n\r\nhi\n" | nc -l -p 80; done'
# The runner's sudo keeps the runner's environment, which a real sudo drops, and its
# /etc/environment, which PAM gives each session, names the runner's own directories. They would
# send root's settings to the runner's home.
unset XDG_CONFIG_HOME
sed -i '/^XDG_/d' /etc/environment
# The runner's apt takes the amd64v3 variant, which a machine turns on by hand. The git-core PPA's
# index of that variant holds no git. So the cases run with no variant, as a machine has none.
# apt reads this file after apt.conf.d, so the line wins wherever the runner set the variant.
echo '#clear APT::Architecture-Variants;' >>/etc/apt/apt.conf
export HOME=/root DEBIAN_FRONTEND=noninteractive

fail() {
    echo "::error::$*"
    exit 1
}

# finish prints Docker's log on a failure.
finish() {
    local status=$?
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
# past dpkg, which would keep it and warn. The same holds for the image's source of the git-core PPA,
# which ws adds with its own key.
rm -rf /usr/libexec/docker/cli-plugins
apt-get purge -y docker-ce docker-ce-cli containerd.io
rm -rf /etc/docker /var/lib/docker /var/lib/containerd
for f in /etc/apt/sources.list.d/*; do
    if [ -f "$f" ] && grep -q -e download.docker.com -e git-core/ppa "$f"; then rm -f "$f"; fi
done
apt-get update
# The image's git comes from that PPA already, so it would pass the git check below whatever the
# setup does. Git and git-man go down to the archive's version, as on a server from Ubuntu's image,
# which the setup must move up to the PPA's.
archive_git=$(apt-cache madison git | awk '$3 !~ /ppa/ && !seen++ { print $3 }')
[ -n "$archive_git" ] || fail "no archive version of git"
apt-get install -y --allow-downgrades --no-install-recommends "git=$archive_git" "git-man=$archive_git"
git_version=$(git version | cut -d' ' -f3)
[ "$(printf '%s\n' "$git_version" 2.54 | sort -V | head -n 1)" != 2.54 ] ||
    fail "git is $git_version after the downgrade, so the git check proves nothing"
apt-get install -y --no-install-recommends netcat-openbsd

# Root logs in by key, as a cloud VM's does.
keys=$(mktemp -d)
ssh-keygen -q -t ed25519 -N '' -f "$keys/key"
install -d -m 700 /root/.ssh
install -m 600 "$keys/key.pub" /root/.ssh/authorized_keys

# A second network, in a namespace of its own: the runner has no other. The cases below publish
# ports and reach them from there.
ip netns add lab
ip link add veth0 type veth peer name veth1
ip link set veth1 netns lab
ip addr add 10.250.0.1/24 dev veth0
ip link set veth0 up
ip netns exec lab ip addr add 10.250.0.2/24 dev veth1
ip netns exec lab ip link set veth1 up
ip netns exec lab ip link set lo up

install -d -m 755 /etc/ws
# The swap file is small, as the runner's disk has no room for one the size of its RAM. The runner is
# no container and has no host, so the guest runs SigNoz and its tunnel itself, as a cloud VM does.
printf '%s\n' --- 'WS_BASE_SWAP_SIZE_MB: 64' 'WS_SELF_CONTAINED: true' >/etc/ws/values.yml
chmod 600 /etc/ws/values.yml
values_kept=$(mktemp)
cp /etc/ws/values.yml "$values_kept"

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

# A cloud VM starts with forwarding off and the policy at accept. The first run starts Docker, which
# turns forwarding on and sets the policy to drop. That cuts wg0 off ikeward0 until a reboot.
sysctl -qw net.ipv4.ip_forward=0
iptables -P FORWARD ACCEPT

# As on a new machine: ws ubuntu guest setup installs mise and runs the bootstrap. The build is already
# /usr/local/bin/ws. It runs twice, and the second run must change nothing. Its plan must exit 2
# before the first run, which has mise, the plugin and the bootstrap to do, and 0 after each run.
# The tree must be root's alone for setup's guard. So must /opt and /usr/share, as on a machine,
# for mise's. The runner's image lets others write both, and mise refuses a system plugin below
# either.
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
log=$(mktemp)
since=$(date +%s)
if ! "$(dirname "$0")/bootstrap.sh" --in-place > >(tee "$log") 2>&1; then
    apt_quiet "$log" "$since" || true
    fail "the first setup"
fi
must "no upgrade ran, and no apt command waited for dpkg's lock" apt_quiet "$log" "$since"
must "the first setup leaves nothing to do" setup --plan
must "the first setup prints no warning" "$check" "$log"
# docker/README.md#the-forward-policy: the policy is read with no reboot after the first run.
forward_policy() {
    local policy
    policy=$(iptables -S FORWARD | head -n 1)
    echo "$policy, ip_forward $(cat /proc/sys/net/ipv4/ip_forward)"
    [ "$policy" = '-P FORWARD ACCEPT' ] && [ "$(cat /proc/sys/net/ipv4/ip_forward)" = 1 ]
}
must "Docker's forward policy is accept after the first run, with forwarding on" forward_policy
must "the second setup runs" setup
must "the second setup leaves nothing to do" setup --plan

# Hooks in git's config came in git 2.54, which Ubuntu 26.04 lacks. The git above was Ubuntu's, so
# ws's git comes from the git-core PPA, whose source mise wrote, and its version has the PPA's suffix.
git_from_ppa() {
    local version
    version=$(git version | sed -n 's/^git version \([0-9][0-9]*\.[0-9][0-9]*\).*/\1/p')
    echo "git $version"
    [ "$(printf '%s\n' "$version" 2.54 | sort -V | head -n 1)" = 2.54 ] &&
        test -f /etc/apt/sources.list.d/mise-git-core.sources &&
        dpkg-query -W git | grep -q ppa
}
must "git is 2.54 or above, from the git-core PPA" git_from_ppa

must "dotnet lists devtools' SDKs alone, which roslyn-language-server finds" \
    "$(dirname "$0")/dotnet-checks.sh"

# What the runs wrote, which needs no container. The plans that must fail, from /opt/ws and with a
# unit's line dropped. The values file, which no run changed. Zed's wrapper and tmux.conf against
# tmux, which takes about a minute and catches a tmux release that changes how list-sessions or
# rename-session behave.
must "the plans that must fail do" "$(dirname "$0")/server-plan.sh"
must "the values file is root's at 600" test "$(stat -c '%u:%g %a' /etc/ws/values.yml)" = '0:0 600'
must "no run changed the values file" cmp "$values_kept" /etc/ws/values.yml
must "the Zed terminal wrapper holds against tmux" \
    sh "$(dirname "$0")/zed-terminal-checks.sh" /usr/local/lib/ws/zed

# Fresh loads a malformed config with its defaults and exits 0, so this reads what it loaded.
fresh_config() {
    local shown
    shown=$(mktemp)
    fresh --cmd config show >"$shown" &&
        "$(dirname "$0")/fresh-config-check.sh" /root/.config/fresh/config.json "$shown"
}
must "Fresh loads the config that ws wrote" fresh_config

# An empty commit, made in a new repository, has git sign and verify with the key's own path, and
# carries the one trailer that ws's hook adds. It reads the git role's key, ~/.gitconfig's include
# of /etc/ws/gitconfig, and /etc/ws/allowed_signers as git reads them. The file's text alone cannot
# tell that git opens a path, or runs the hook.
signed_commit() {
    local repo status trailers
    repo=$(mktemp -d)
    git -C "$repo" -c init.defaultBranch=main init -q
    git -C "$repo" commit -q --allow-empty -m check
    status=$(git -C "$repo" log -1 --format=%G?)
    trailers=$(git -C "$repo" log -1 --format=%B | grep -c '^Co-authored-by: zadykian <me@zadykian.dev>$' || true)
    rm -rf "$repo"
    echo "signature $status, $trailers trailer(s)"
    [ "$status" = G ] && [ "$trailers" = 1 ]
}
must "root signs a commit, git verifies it, and it holds one co-author trailer" signed_commit

# The signing key is made by ws ubuntu ssh keygen, so list shows it, from a file, and exits 0.
signing_key_listed() {
    local keys
    keys=$(ws ubuntu ssh list) || return 1
    echo "$keys"
    awk -F'\t' '$1 == "ws-sign" && $4 == "file" { found = 1 } END { exit !found }' <<<"$keys"
}
must "ws ubuntu ssh list shows the signing key ws-sign, from a file" signing_key_listed

# The pulls ahead of the runs, as a run of an image not yet pulled warns.
must "docker pulls the image" docker pull -q "$image"
must "docker runs a container" docker run --rm "$image" true
must "a container reaches the internet" \
    docker run --rm "$image" wget -qO /dev/null --timeout=10 http://example.com
# Docker's containers send OTLP/gRPC to the bridge address, and the collector takes it
# (ubuntu/roles/otelcol/README.md#dockers-containers).
must "a container's OTLP reaches the collector on the bridge address" "$(dirname "$0")/otlp-checks.sh" "$image"

# A port published with no address goes on loopback, as a guest holds no filter
# (ubuntu/roles/docker/README.md#published-ports). It answers on the host, not on the
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
# ws unit picks its side by the directory mise runs in, so the call runs in the guest's.
if out=$(cd /opt/ws/ubuntu/guest && ws unit write --dry-run file/shell-bashrc-line 2>&1); then
    fail "ws unit write planned through a link at ~/.bashrc"
fi
must "a link at ~/.bashrc is refused" grep -q 'is a link to /etc/hostname' <<<"$out"
rm /root/.bashrc
mv /root/.bashrc.orig /root/.bashrc
