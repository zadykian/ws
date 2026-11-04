#!/bin/bash
# The install job's cases: install.sh and ws update against a release laid out in a temporary
# directory and served on loopback as GitHub serves it. Run it from the repository's root;
# tools/build gets the Go.
# The releases are of HEAD, built at tags that a copy of it makes and removes again.
#
#   v0.1.0 and v0.2.0  good; v0.2.0 is the latest
#   v0.3.0             ws.sha256 holds another binary's sum
#   v0.4.0             a binary that prints v0.1.0, with the sum it should have
#
# The root case writes /usr/local/bin/ws, so it runs on GitHub's runner alone, with sudo.

set -euo pipefail

root=$PWD
tmp=$(mktemp -d)
trap 'kill "${server:-}" 2>/dev/null || true; rm -rf "$tmp"' EXIT

# src is HEAD in a repository of its own, for the tags: a release's run holds one at HEAD already.
src=$tmp/src
mkdir "$src"
git archive HEAD | tar -x -C "$src"
git -C "$src" init -q
git -C "$src" add -A
git -C "$src" -c user.name=ws -c user.email=ws@localhost -c commit.gpgsign=false commit -q -m tree
# The build shares this tree's .cache, with the pinned Go in it. The link is excluded, as git's
# ignore of a directory does not match a link: else the build is +dirty and prints another version.
mkdir -p "$root/.cache"
ln -s "$root/.cache" "$src/.cache"
echo /.cache >>"$src/.git/info/exclude"

# build_release TAG makes release TAG's directory, with the tree's ws as the tag stamps it.
build_release() {
    local dir=$tmp/releases/download/$1
    mkdir -p "$dir"
    (cd "$src" && git tag "$1" && tools/build linux amd64 "$dir/ws-linux-amd64" && git tag -d "$1" >/dev/null)
    [ "$("$dir/ws-linux-amd64" version)" = "$1" ] || fail "the build at $1 prints another version"
    (cd "$dir" && sha256sum ws-linux-amd64 >ws.sha256)
}

fail() {
    echo "::error::$*"
    exit 1
}

# installs HOME_DIR [ENV=VALUE...] runs install.sh with HOME_DIR as the home, and the settings.
installs() {
    local home=$1
    shift
    env HOME="$home" WS_RELEASES_URL="$url" "$@" sh "$root/install.sh"
}

# has_version HOME_DIR TAG fails unless the ws installed for HOME_DIR prints TAG.
has_version() {
    local got
    got=$("$1/.local/bin/ws" version)
    [ "$got" = "$2" ] || fail "ws in $1 is $got, not $2"
}

# only_ws HOME_DIR fails where its bin directory holds more than ws, such as a download left.
only_ws() {
    local left
    left=$(ls -A "$1/.local/bin")
    [ "$left" = ws ] || fail "$1/.local/bin holds '$left', not ws alone"
}

# refused HOME_DIR TAG... fails where install.sh succeeds, or installs anything.
refused() {
    local home=$1
    shift
    if installs "$home" "$@"; then
        fail "install.sh went through with $*"
    fi
}

# updates HOME_DIR runs the ws installed for HOME_DIR as ws update.
updates() {
    env HOME="$1" WS_RELEASES_URL="$url" "$1/.local/bin/ws" update
}

# update_refused HOME_DIR TAG fails where ws update, with TAG the latest release, goes through or
# changes the v0.1.0 installed for HOME_DIR.
update_refused() {
    echo "$2" >"$tmp/releases/latest"
    if updates "$1"; then
        fail "ws update went through at $2"
    fi
    echo v0.2.0 >"$tmp/releases/latest"
    has_version "$1" v0.1.0
    only_ws "$1"
}

[ "$(id -u)" != 0 ] || fail "run it as a user: as root, install.sh installs to /usr/local/bin"
[ -z "$(git status --porcelain)" ] || fail "the tree is not clean: the builds would say +dirty"

build_release v0.1.0
build_release v0.2.0
build_release v0.3.0
build_release v0.4.0
cp "$tmp/releases/download/v0.1.0/ws-linux-amd64" "$tmp/releases/download/v0.4.0/ws-linux-amd64"
(cd "$tmp/releases/download/v0.4.0" && sha256sum ws-linux-amd64 >ws.sha256)
sed -i 's/^[0-9a-f]\{64\}/0000000000000000000000000000000000000000000000000000000000000000/' \
    "$tmp/releases/download/v0.3.0/ws.sha256"

echo v0.2.0 >"$tmp/releases/latest"
serve=$tmp/serve-releases
"$(sh "$root/tools/run" -path go)" build -o "$serve" ./tools/serve-releases
"$serve" "$tmp/releases" 0 "$tmp/port" &
server=$!
for _ in $(seq 50); do
    [ -s "$tmp/port" ] && break
    sleep 0.1
done
url=http://127.0.0.1:$(cat "$tmp/port")

echo "::group::WS_VERSION installs that release"
mkdir -p "$tmp/home1"
installs "$tmp/home1" WS_VERSION=0.1.0
has_version "$tmp/home1" v0.1.0
only_ws "$tmp/home1"
echo "::endgroup::"

echo "::group::A second run, for the latest, updates"
installs "$tmp/home1"
has_version "$tmp/home1" v0.2.0
only_ws "$tmp/home1"
echo "::endgroup::"

echo "::group::The latest installs into a new home"
installs "$tmp/home2"
has_version "$tmp/home2" v0.2.0
echo "::endgroup::"

echo "::group::A wrong ws.sha256 is refused, and ws is as it was"
refused "$tmp/home1" WS_VERSION=0.3.0
has_version "$tmp/home1" v0.2.0
only_ws "$tmp/home1"
mkdir -p "$tmp/home3"
refused "$tmp/home3" WS_VERSION=v0.3.0
[ -z "$(ls -A "$tmp/home3/.local/bin")" ] || fail "a refused install left files"
echo "::endgroup::"

echo "::group::A binary of another version is refused, and ws is as it was"
refused "$tmp/home1" WS_VERSION=0.4.0
has_version "$tmp/home1" v0.2.0
only_ws "$tmp/home1"
echo "::endgroup::"

echo "::group::A version that is no version is refused"
refused "$tmp/home1" WS_VERSION=0.2
echo "::endgroup::"

echo "::group::ws update moves ws to the latest release, and a second run stays"
installs "$tmp/home4" WS_VERSION=0.1.0
updates "$tmp/home4"
has_version "$tmp/home4" v0.2.0
only_ws "$tmp/home4"
updates "$tmp/home4"
has_version "$tmp/home4" v0.2.0
echo "::endgroup::"

echo "::group::ws update replaces the file a link leads to"
installs "$tmp/home5" WS_VERSION=0.1.0
mkdir "$tmp/home5/other"
mv "$tmp/home5/.local/bin/ws" "$tmp/home5/other/ws"
ln -s ../../other/ws "$tmp/home5/.local/bin/ws"
updates "$tmp/home5"
[ -L "$tmp/home5/.local/bin/ws" ] || fail "ws update replaced the link"
[ "$("$tmp/home5/other/ws" version)" = v0.2.0 ] || fail "ws update left the link's file as it was"
echo "::endgroup::"

echo "::group::ws update refuses what install.sh refuses, and ws is as it was"
installs "$tmp/home6" WS_VERSION=0.1.0
update_refused "$tmp/home6" v0.3.0
update_refused "$tmp/home6" v0.4.0
echo "::endgroup::"

if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::group::Root installs to /usr/local/bin"
    sudo env WS_RELEASES_URL="$url" sh "$root/install.sh"
    [ "$(/usr/local/bin/ws version)" = v0.2.0 ] || fail "root's ws is not v0.2.0"
    echo "::endgroup::"
fi
