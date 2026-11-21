#!/bin/bash
# The install job's cases of ws update. Run it from the repository's root, as a user with sudo.
# ws update writes /usr/local/bin/ws as root, which each case replaces. So it runs on GitHub's
# runner, or on a spare machine where WS_SPARE_MACHINE is 1, and never on a server.
#
# Only the ws under test is real. Each is a build of this tree. A tag in a copy of the tree stamps
# it as a release, and none makes it a trial's. serve-releases serves stubs as the releases. A stub
# is a shell script that prints its tag as ws version, and records a call of server setup. That
# is all that ws update runs of the new ws. A real setup is the trial's to run (#1385).
#
#     v0.2.0, v0.4.0-rc.1  stubs, served; v0.2.0 is the latest
#     v0.5.0               a stub that its ws.sha256 does not fit
#     builds               real, never served: v0.1.0, v0.3.0, v0.4.0-rc.1 and a trial's, at no tag
#
# An update leaves a stub, which runs no update, so each case installs a build again.

set -euo pipefail

root=$PWD

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

# build NAME [TAG] builds this tree's ws as $tmp/builds/NAME, in a copy of the tree that holds
# one commit. With TAG the commit is tagged while it builds, so that ws prints TAG. Without one,
# it prints a pseudo-version, as a trial's build does.
build() {
    (cd "$tmp/src" && { [ -z "${2:-}" ] || git tag "$2"; } &&
        .tools/build linux amd64 "$tmp/builds/$1" && { [ -z "${2:-}" ] || git tag -d "$2" >/dev/null; })
    [ -z "${2:-}" ] || [ "$("$tmp/builds/$1" version)" = "$2" ] || fail "the build of $2 prints another version"
}

# stub TAG makes release TAG's directory with a stub of ws and its ws.sha256.
stub() {
    local dir=$tmp/serve/download/$1
    mkdir -p "$dir"
    cat >"$dir/ws-linux-amd64" <<SH
#!/bin/sh
case \$1 in
version) echo $1 ;;
server) echo "$1 \$*" >>$log ;;
esac
SH
    chmod 755 "$dir/ws-linux-amd64"
    (cd "$dir" && sha256sum ws-linux-amd64 >ws.sha256)
}

# install_ws BUILD makes $tmp/builds/BUILD the ws of the machine, as install.sh does, and clears the
# setups that the log holds.
install_ws() {
    as_root install -o 0 -g 0 -m 0755 "$tmp/builds/$1" /usr/local/bin/.ws.case
    as_root mv -f /usr/local/bin/.ws.case /usr/local/bin/ws
    : >"$log"
}

# update [ARGS...] runs ws update as root against the served releases.
update() {
    as_root env WS_RELEASES_URL="$url" /usr/local/bin/ws update "$@"
}

# is_ws WANT fails unless /usr/local/bin/ws is root's, 0755, and prints WANT as ws version.
is_ws() {
    [ "$(as_root stat -c '%u:%g %a' /usr/local/bin/ws)" = "0:0 755" ] || fail "ws is not 0:0 and 755"
    [ "$(/usr/local/bin/ws version)" = "$1" ] || fail "ws is not $1: $(/usr/local/bin/ws version)"
}

# setups WANT fails unless the log holds the lines of WANT, the calls of server setup so far.
setups() {
    [ "$(cat "$log")" = "$1" ] || fail "the setups are '$(cat "$log")', not '$1'"
}

# nothing_left fails where a run left a download in /usr/local/bin.
nothing_left() {
    ! compgen -G '/usr/local/bin/.ws.*' >/dev/null || fail "a run left a download"
}

# refused ERROR ARGS... fails unless ws update with ARGS exits non-zero and says ERROR.
refused() {
    local error=$1
    shift
    if update "$@" 2>"$tmp/refusal"; then
        fail "ws update $* went through"
    fi
    grep -q -- "$error" "$tmp/refusal" || fail "ws update $* was refused, but not by '$error': $(cat "$tmp/refusal")"
}

[ "$(id -u)" != 0 ] || [ -n "${WS_SPARE_MACHINE:-}" ] || fail "run it as a user with sudo"
[ -n "${GITHUB_ACTIONS:-}${WS_SPARE_MACHINE:-}" ] || fail "it replaces /usr/local/bin/ws: run it on a runner or a spare machine"
[ "$(uname -m)" = x86_64 ] || fail "the builds are for linux amd64"
[ -z "$(git status --porcelain)" ] || fail "the tree is not clean: the builds would say +dirty"
# Case 6 runs a ws that is its release already, whose setup is a real one where the system is
# Ubuntu 26.04. The runner's system refuses it.
# shellcheck disable=SC1091 # the system's own file.
[ "$(. /etc/os-release && echo "$VERSION_ID")" != 26.04 ] || fail "the runner is Ubuntu 26.04, where ws's setup is real: change case 6"

# The trap comes after the guards: a run they refuse must leave this machine's ws, which it deletes.
tmp=$(mktemp -d)
log=$tmp/setup.log
trap 'kill "${server:-}" 2>/dev/null || true; as_root rm -f /usr/local/bin/ws /usr/local/bin/.ws.*; rm -rf "$tmp"' EXIT

# src is HEAD in a repository of its own, for the tags: a release's run holds one at HEAD already.
mkdir -p "$tmp/src" "$tmp/builds"
git archive HEAD | tar -x -C "$tmp/src"
git -C "$tmp/src" init -q
git -C "$tmp/src" add -A
git -C "$tmp/src" -c user.name=ws -c user.email=ws@localhost -c commit.gpgsign=false commit -q -m tree
# The build shares this tree's .cache, with the pinned Go in it. The link is excluded, as git's
# ignore of a directory does not match a link: else the build is +dirty and prints another version.
mkdir -p "$root/.cache"
ln -s "$root/.cache" "$tmp/src/.cache"
echo /.cache >>"$tmp/src/.git/info/exclude"
build v0.1.0 v0.1.0
build v0.3.0 v0.3.0
build v0.4.0-rc.1 v0.4.0-rc.1
build trial

for release in v0.2.0 v0.4.0-rc.1 v0.5.0; do
    stub "$release"
done
# A release whose sums are not its file's.
printf '%064d  ws-linux-amd64\n' 0 >"$tmp/serve/download/v0.5.0/ws.sha256"
echo v0.2.0 >"$tmp/serve/latest"

serve=$tmp/serve-releases
"$("$root/.tools/run" -path go)" build -o "$serve" ./.tools/serve-releases
"$serve" "$tmp/serve" 0 "$tmp/port" &
server=$!
for _ in $(seq 50); do
    [ -s "$tmp/port" ] && break
    sleep 0.1
done
url=http://127.0.0.1:$(cat "$tmp/port")

echo "::group::1. A release's ws moves to the latest and runs its setup"
install_ws v0.1.0
update
is_ws v0.2.0
setups "v0.2.0 server setup"
nothing_left
echo "::endgroup::"

echo "::group::2. A trial's build is replaced, which ends the trial"
install_ws trial
update
is_ws v0.2.0
setups "v0.2.0 server setup"
nothing_left
echo "::endgroup::"

echo "::group::3. --version takes a pre-release, which the latest release is never"
install_ws v0.1.0
update --version 0.4.0-rc.1
is_ws v0.4.0-rc.1
setups "v0.4.0-rc.1 server setup"
# A real pre-release is after the latest release, which is then refused.
install_ws v0.4.0-rc.1
refused '--version'
is_ws v0.4.0-rc.1
setups ""
nothing_left
echo "::endgroup::"

echo "::group::4. A release older than this ws is refused, unless --version names it"
install_ws v0.3.0
refused '--version'
is_ws v0.3.0
setups ""
nothing_left
update --version 0.2.0
is_ws v0.2.0
setups "v0.2.0 server setup"
echo "::endgroup::"

echo "::group::5. A wrong sum and a missing release are refused, and ws stays as it was"
install_ws v0.1.0
refused 'does not match' --version 0.5.0
refused '404' --version 0.7.0
is_ws v0.1.0
setups ""
nothing_left
echo "::endgroup::"

echo "::group::6. A ws that is the release already downloads nothing and runs its own setup"
install_ws v0.3.0
# v0.3.0 is not served: a download would fail on its 404. Setup is the build's own, which the
# runner's system refuses.
refused 'Ubuntu 26.04 alone' --version 0.3.0
is_ws v0.3.0
nothing_left
echo "::endgroup::"

if [ "$(id -u)" != 0 ]; then
    echo "::group::7. A user is refused before any download"
    if WS_RELEASES_URL="$url" /usr/local/bin/ws update 2>"$tmp/refusal"; then
        fail "ws update went through as a user"
    fi
    grep -q 'run it as root' "$tmp/refusal" || fail "ws update refused a user, but not by its message"
    is_ws v0.3.0
    nothing_left
    echo "::endgroup::"
fi
