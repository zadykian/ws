#!/bin/bash
# The install job's cases of install.sh, with the real mise and a small repository in place of the
# release's tree. Run it from the repository's root, as a user with sudo. install.sh runs as root
# and writes /usr/local/bin/mise, /etc/mise and /opt/ws, which each case removes. So it
# runs on GitHub's runner, or on a spare machine where WS_SPARE_MACHINE is 1, and never on a
# server.
#
# mise's binary, which install.sh pins, is downloaded once into a directory that serve-releases
# serves on loopback. A copy of it with a byte added stands for a wrong one.
#
#     /mise/good/v$V/  the real binary          /mise/bad/v$V/  a copy with a byte added
#     /latest          v0.2.0                   /mise/none/     no binary
#
# The repository v0.2.0 holds a file of mise's config for each env, and each writes one file. The
# files written name the envs that the miserc loaded.

set -euo pipefail

root=$PWD
tmp=$(mktemp -d)
trap 'kill "${server:-}" 2>/dev/null || true; reset; rm -rf "$tmp"' EXIT

fail() {
    echo "::error::$*"
    exit 1
}

# as_root runs the command $@ as root, through sudo where the user is not root.
as_root() {
    if [ "$(id -u)" = 0 ]; then
        "$@"
    else
        sudo "$@"
    fi
}

# reset removes what a run of install.sh leaves, so that each case starts as a new machine.
reset() {
    as_root rm -rf /usr/local/bin/mise /usr/local/bin/.mise.* /etc/mise /opt/ws /etc/ws-install-check \
        /usr/share/mise
}

# install_sh [ENV=VALUE...] runs install.sh as root with the settings, from a stdin that is a pipe,
# as curl's is. The runner's system stands as Ubuntu 26.04, whatever its release.
install_sh() {
    echo | as_root env WS_RELEASES_URL="$url" WS_MISE_URL="$url/mise/good" WS_REPO="file://$tmp/repo" \
        WS_OS_RELEASE="$tmp/os-release" "$@" sh "$root/install.sh"
}

# written ENV fails unless the bootstrap wrote the file of ENV, with ENV in it.
written() {
    [ "$(cat "/etc/ws-install-check/$1" 2>/dev/null)" = "$1" ] || fail "the bootstrap wrote no file of env $1"
}

[ "$(id -u)" != 0 ] || [ -n "${WS_SPARE_MACHINE:-}" ] || fail "run it as a user with sudo"
[ -n "${GITHUB_ACTIONS:-}${WS_SPARE_MACHINE:-}" ] || fail "it removes /etc/mise and /opt/ws: run it on a runner or a spare machine"

# GitHub's runner keeps a default ACL on /opt, which lets anyone write what root makes there,
# whatever the umask. install.sh refuses such a tree, and a server's /opt has no such ACL. Its
# image lets others write /opt and /usr/share too, which install.sh refuses as mise would ignore
# its plugin below them, and a server's are root's alone.
! command -v setfacl >/dev/null || as_root setfacl -k /opt
as_root chmod go-w /opt /usr/share

printf '%s\n' 'NAME="Ubuntu"' 'ID=ubuntu' 'VERSION_ID="26.04"' >"$tmp/os-release"
version=$(sed -n 's/^mise_version=//p' install.sh)
case $(uname -m) in
x86_64) asset=linux-x64 ;;
aarch64) asset=linux-arm64 ;;
*) fail "mise is pinned for x86_64 and aarch64 alone" ;;
esac
good=$tmp/serve/mise/good/v$version
bad=$tmp/serve/mise/bad/v$version
mkdir -p "$good" "$bad" "$tmp/serve/mise/none"
curl -fsSL --proto '=https' --tlsv1.2 -o "$good/mise-v$version-$asset" \
    "https://github.com/jdx/mise/releases/download/v$version/mise-v$version-$asset"
cp "$good/mise-v$version-$asset" "$bad/"
printf x >>"$bad/mise-v$version-$asset"
echo v0.2.0 >"$tmp/serve/latest"

# The release's tree: a config with an entry for each env's file, and a directory for them. The
# secrets' file sits in a role, behind the side's link, as a role's do, and the root holds none.
# On such a tree mise trust --all never ends, so the case of the latest release proves install.sh's
# trust. A task stands for the secrets task: mise asks to trust a task that
# holds a template's {{, {% or {#, and the secrets task's ${#host} holds {#.
repo=$tmp/repo
mkdir -p "$repo/roles/server/ubuntu/part" "$repo/roles/server/mise" "$repo/roles/server/mise-tasks"
ln -s ../ubuntu "$repo/roles/server/mise/conf.d"
echo '# the install job'"'"'s tree' >"$repo/mise.toml"
# shellcheck disable=SC2016 # the task's own shell expands it, not this one.
printf '%s\n' '#!/bin/sh' 'echo "${#0}"' >"$repo/roles/server/mise-tasks/check"
chmod 755 "$repo/roles/server/mise-tasks/check"
for env in server booted secrets; do
    config=$repo/mise.$env.toml
    [ "$env" != secrets ] || config=$repo/roles/server/ubuntu/part/mise.$env.toml
    cat >"$config" <<TOML
[bootstrap.directories."/etc/ws-install-check"]
mode = "0755"

[bootstrap.files."/etc/ws-install-check/$env"]
content = "$env\n"
TOML
done
git -C "$repo" init -q
git -C "$repo" add -A
git -C "$repo" -c user.name=ws -c user.email=ws@localhost -c commit.gpgsign=false commit -q -m tree
git -C "$repo" tag v0.2.0

serve=$tmp/serve-releases
"$("$root/.tools/run" -path go)" build -o "$serve" ./.tools/serve-releases
"$serve" "$tmp/serve" 0 "$tmp/port" &
server=$!
for _ in $(seq 50); do
    [ -s "$tmp/port" ] && break
    sleep 0.1
done
url=http://127.0.0.1:$(cat "$tmp/port")

reset
echo "::group::A user is refused, and nothing is installed"
if WS_OS_RELEASE="$tmp/os-release" sh "$root/install.sh" </dev/null; then
    fail "install.sh went through as a user"
fi
[ ! -e /usr/local/bin/mise ] || fail "a refused install left mise"
echo "::endgroup::"

echo "::group::A system other than Ubuntu 26.04 is refused, and nothing is installed"
printf '%s\n' 'ID=ubuntu' 'VERSION_ID="24.04"' >"$tmp/os-release-24.04"
printf '%s\n' 'ID=debian' 'VERSION_ID="13"' >"$tmp/os-release-debian"
for system in 24.04 debian none; do
    if install_sh WS_OS_RELEASE="$tmp/os-release-$system"; then
        fail "install.sh went through on $system"
    fi
    [ ! -e /usr/local/bin/mise ] || fail "a refused install on $system left mise"
done
echo "::endgroup::"

echo "::group::A wrong SHA-256 is refused, and nothing is left"
if install_sh WS_MISE_URL="$url/mise/bad"; then
    fail "install.sh went through with a wrong mise"
fi
[ ! -e /usr/local/bin/mise ] || fail "a refused install left mise"
! compgen -G '/usr/local/bin/.mise.*' >/dev/null || fail "a refused install left a download"
echo "::endgroup::"

echo "::group::The latest release installs mise and runs its bootstrap"
install_sh
[ "$(/usr/local/bin/mise --version | cut -d' ' -f1)" = "$version" ] || fail "mise is not $version"
# The miserc is a template, which mise renders: the files show what it loaded.
want=(server)
[ ! -d /run/systemd/system ] || want+=(booted)
for env in "${want[@]}"; do
    written "$env"
done
# install.sh trusts the tree, so that a plan by hand, with no MISE_TRUSTED_CONFIG_PATHS, runs and
# finds no change. This tree holds no tool, so the plan needs no MISE_DATA_DIR either.
as_root /usr/local/bin/mise -C /opt/ws/roles/server bootstrap plan --detailed-exitcode </dev/null >/dev/null ||
    fail "a plan by hand after install.sh found a change or failed"
# A run by hand loads the tree's tasks, and mise refuses one that install.sh left untrusted. Where
# CI or MISE_YES is set, mise takes an untrusted config, so neither is passed.
as_root env -u CI -u MISE_YES /usr/local/bin/mise -C /opt/ws/roles/server bootstrap </dev/null >/dev/null ||
    fail "a run by hand after install.sh refused the tree's task or failed"
# The set ws leaves secrets out, which a run adds by -E ws,secrets. A plain plan after it keeps the
# env's file, as a run without the env changes none of it.
[ ! -e /etc/ws-install-check/secrets ] || fail "install.sh's run loaded the env secrets"
as_root /usr/local/bin/mise -C /opt/ws/roles/server -E ws,secrets bootstrap </dev/null >/dev/null ||
    fail "a run with -E ws,secrets failed"
for env in "${want[@]}" secrets; do
    written "$env"
done
as_root /usr/local/bin/mise -C /opt/ws/roles/server bootstrap plan --detailed-exitcode </dev/null >/dev/null ||
    fail "a plan by hand after the env secrets found a change or failed"
echo "::endgroup::"

echo "::group::A second run downloads no mise"
install_sh WS_MISE_URL="$url/mise/none"
written server
echo "::endgroup::"

# at TAG fails unless /opt/ws stands at the commit of TAG.
at() {
    [ "$(as_root git -C /opt/ws rev-parse HEAD)" = "$(git -C "$repo" rev-parse "$1^{commit}")" ] ||
        fail "/opt/ws is not at $1"
}

echo "::group::A newer release checks its tag out, and drops the files git does not track"
cat >>"$repo/mise.server.toml" <<'TOML'

[bootstrap.files."/etc/ws-install-check/release"]
content = "v0.3.0\n"
TOML
git -C "$repo" -c user.name=ws -c user.email=ws@localhost -c commit.gpgsign=false commit -q -am release
git -C "$repo" tag v0.3.0
echo v0.3.0 >"$tmp/serve/latest"
# A trial's file, as .tools/serve-build writes it, would pin another ws.
printf '[bootstrap.files."/etc/ws-install-check/local"]\ncontent = "local\\n"\n' |
    as_root tee /opt/ws/mise.local.toml >/dev/null
install_sh WS_MISE_URL="$url/mise/none"
at v0.3.0
[ "$(cat /etc/ws-install-check/release 2>/dev/null)" = v0.3.0 ] || fail "the bootstrap ran no tree of v0.3.0"
[ ! -e /opt/ws/mise.local.toml ] || fail "a file git does not track stayed in /opt/ws"
[ ! -e /etc/ws-install-check/local ] || fail "the bootstrap loaded a file git does not track"
echo "::endgroup::"

echo "::group::WS_VERSION checks its release's tag out"
install_sh WS_VERSION=0.2.0 WS_MISE_URL="$url/mise/none"
at v0.2.0
echo "::endgroup::"

echo "::group::A pre-release runs where WS_VERSION names it alone"
git -C "$repo" -c user.name=ws -c user.email=ws@localhost -c commit.gpgsign=false commit -q --allow-empty -m rc
git -C "$repo" tag v0.4.0-rc.1
install_sh WS_VERSION=0.4.0-rc.1 WS_MISE_URL="$url/mise/none"
at v0.4.0-rc.1
# GitHub never redirects releases/latest to a pre-release: install.sh refuses one that it finds there.
echo v0.4.0-rc.1 >"$tmp/serve/latest"
if install_sh WS_MISE_URL="$url/mise/none" 2>"$tmp/refusal"; then
    fail "install.sh took a pre-release as the latest release"
fi
grep -q 'GitHub redirects to' "$tmp/refusal" || fail "install.sh refused a pre-release as the latest, but not by its message"
at v0.4.0-rc.1
echo v0.3.0 >"$tmp/serve/latest"
echo "::endgroup::"

echo "::group::A version that is no version is refused"
reset
for version in 0.2 0.4.0-rc.0 0.4.0-rc.01 0.4.0-beta.1; do
    if install_sh WS_VERSION="$version"; then
        fail "install.sh went through with WS_VERSION=$version"
    fi
    [ ! -e /usr/local/bin/mise ] || fail "a refused install of $version left mise"
done
echo "::endgroup::"

echo "::group::WS_IN_PLACE runs the tree in /opt/ws as it stands"
as_root cp -r "$repo" /opt/ws
install_sh WS_IN_PLACE=1 WS_REPO="file://$tmp/nothing"
written server
echo "::endgroup::"

echo "::group::A tree that is not root's alone is refused"
as_root chown nobody /opt/ws/mise.server.toml
as_root rm -f /etc/ws-install-check/server
if install_sh WS_IN_PLACE=1 WS_REPO="file://$tmp/nothing"; then
    fail "install.sh ran a tree with a file that root does not own"
fi
[ ! -e /etc/ws-install-check/server ] || fail "a refused tree ran"
echo "::endgroup::"

echo "::group::An /opt that others may write is refused, as mise would ignore the plugin ws there"
reset
as_root cp -r "$repo" /opt/ws
as_root chmod o+w /opt
if install_sh WS_IN_PLACE=1 WS_REPO="file://$tmp/nothing" 2>"$tmp/refusal"; then
    fail "install.sh ran with an /opt that others may write"
fi
as_root chmod o-w /opt
grep -q 'mise ignores the plugin ws' "$tmp/refusal" || fail "install.sh refused /opt, but not by its message"
[ ! -e /etc/ws-install-check/server ] || fail "a refused run went on to the bootstrap"
echo "::endgroup::"
