#!/bin/bash
# The install job's cases of install.sh, with stubs in place of ws's releases. Run it from the
# repository's root, as a user with sudo. install.sh runs as root and writes /usr/local/bin/ws,
# which each case removes, and the cases move /usr/local/bin aside and change its owner. So it
# runs on GitHub's runner, or on a spare machine where WS_SPARE_MACHINE is 1, and never on a
# server.
#
# serve-releases serves the releases on loopback, as GitHub does. A stub of ws is a shell script
# that prints its release as ws version, as the real ws does, and its target as ws target. A fake uname on the PATH of a case stands
# for another target, as the runner is Linux on x86_64.
#
#     /download/v0.2.0/ws-TARGET  a stub, for each target        /latest  v0.2.0
#     /download/v0.2.0/ws.sha256  its SHA-256s, sha256sum's form
#     /download/v0.5.0/           a ws-linux-amd64 that its sums do not fit
#     /download/v0.6.0/           the sums of v0.2.0, and no file
#     /download/v0.7.0/           a ws-linux-amd64 that prints v0.0.1, its sum right

set -euo pipefail

root=$PWD
tmp=$(mktemp -d)
aside=/usr/local/bin.aside
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

# reset removes what a run of install.sh leaves, and puts back what a case changed in /usr/local,
# so that each case starts as a new machine.
reset() {
    if [ -e "$aside" ]; then
        as_root rm -rf /usr/local/bin
        as_root mv "$aside" /usr/local/bin
    fi
    as_root chown 0:0 /usr/local /usr/local/bin
    as_root chmod 755 /usr/local /usr/local/bin
    as_root rm -rf /usr/local/bin/ws /usr/local/bin/.ws.*
}

# install_sh [ENV=VALUE...] runs install.sh as root with the settings, from a stdin that is a pipe,
# as curl's is.
install_sh() {
    echo | as_root env WS_RELEASES_URL="$url" "$@" sh "$root/install.sh"
}

# fake_uname OS ARCH makes a uname that answers OS and ARCH, in $tmp/fake/OS-ARCH, and prints the
# directory.
fake_uname() {
    mkdir -p "$tmp/fake/$1-$2"
    cat >"$tmp/fake/$1-$2/uname" <<SH
#!/bin/sh
case \$1 in
-s) echo $1 ;;
-m) echo $2 ;;
esac
SH
    chmod 755 "$tmp/fake/$1-$2/uname"
    echo "$tmp/fake/$1-$2"
}

# installed WANT fails unless /usr/local/bin/ws is root's, 0755, and prints WANT as ws version.
installed() {
    [ "$(as_root stat -c '%u:%g %a' /usr/local/bin/ws)" = "0:0 755" ] || fail "ws is not 0:0 and 755"
    [ "$(/usr/local/bin/ws version) $(/usr/local/bin/ws target)" = "$1" ] ||
        fail "ws is not the stub $1: $(/usr/local/bin/ws version) $(/usr/local/bin/ws target)"
}

# nothing_left fails where a refused run left ws or a download in /usr/local/bin.
nothing_left() {
    [ ! -e /usr/local/bin/ws ] || fail "a refused install left ws"
    ! compgen -G '/usr/local/bin/.ws.*' >/dev/null || fail "a refused install left a download"
}

# stub RELEASE TARGET [VERSION] writes the stub of ws for TARGET into the release's directory. It
# prints VERSION, which is the release by default, as ws version.
stub() {
    # shellcheck disable=SC2016 # The case's $1 is the stub's own, to expand when it runs.
    printf '%s\n' '#!/bin/sh' 'case $1 in' "version) echo ${3:-$1} ;;" "target) echo $2 ;;" 'esac' \
        >"$tmp/serve/download/$1/ws-$2"
}

# sums RELEASE writes the release's ws.sha256 for the files in its directory.
sums() {
    (cd "$tmp/serve/download/$1" && sha256sum ws-* >ws.sha256)
}

[ "$(id -u)" != 0 ] || [ -n "${WS_SPARE_MACHINE:-}" ] || fail "run it as a user with sudo"
[ -n "${GITHUB_ACTIONS:-}${WS_SPARE_MACHINE:-}" ] || fail "it replaces /usr/local/bin/ws: run it on a runner or a spare machine"
[ "$(uname -m)" = x86_64 ] || fail "the cases take x86_64 as the machine's own target"

for release in v0.2.0 v0.3.0 v0.4.0-rc.1 v0.5.0 v0.6.0 v0.7.0; do
    mkdir -p "$tmp/serve/download/$release"
done
for target in linux-amd64 linux-arm64 darwin-arm64; do
    for release in v0.2.0 v0.3.0 v0.4.0-rc.1; do
        stub "$release" "$target"
    done
done
sums v0.2.0
sums v0.3.0
sums v0.4.0-rc.1
# A release whose sums are not its file's, and one that lists a file that it lacks.
stub v0.5.0 linux-amd64
printf '%064d  ws-linux-amd64\n' 0 >"$tmp/serve/download/v0.5.0/ws.sha256"
cp "$tmp/serve/download/v0.2.0/ws.sha256" "$tmp/serve/download/v0.6.0/ws.sha256"
# A release whose ws prints another version, with sums that fit it.
stub v0.7.0 linux-amd64 v0.0.1
sums v0.7.0
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

# The image of a runner lets others write /usr/local and its bin, as GitHub's ubuntu image does, and
# a server's are root's alone.
as_root chmod go-w /usr/local /usr/local/bin
reset

echo "::group::A user is refused, and nothing is installed"
if WS_RELEASES_URL="$url" sh "$root/install.sh" </dev/null; then
    fail "install.sh went through as a user"
fi
nothing_left
echo "::endgroup::"

echo "::group::The latest release installs ws for each target"
install_sh
installed "v0.2.0 linux-amd64"
for target in linux-arm64:Linux:aarch64 darwin-arm64:Darwin:arm64; do
    IFS=: read -r want os arch <<<"$target"
    reset
    out=$(echo | as_root env PATH="$(fake_uname "$os" "$arch"):$PATH" WS_RELEASES_URL="$url" sh "$root/install.sh" 2>&1)
    installed "v0.2.0 $want"
    case $want in
    darwin-*) next="ws client setup" ;;
    *) next="ws server setup" ;;
    esac
    [ "${out##*$'\n'}" = "install.sh: next, run: $next" ] || fail "the last line for $want is not the next command: ${out##*$'\n'}"
done
echo "::endgroup::"

echo "::group::A system that has no build is refused, and nothing is installed"
reset
for target in Linux:armv7l Darwin:x86_64 FreeBSD:amd64; do
    IFS=: read -r os arch <<<"$target"
    if echo | as_root env PATH="$(fake_uname "$os" "$arch"):$PATH" WS_RELEASES_URL="$url" sh "$root/install.sh"; then
        fail "install.sh went through on $os $arch"
    fi
    nothing_left
done
echo "::endgroup::"

echo "::group::A wrong SHA-256 and a missing file are refused, and nothing is left"
for version in 0.5.0 0.6.0; do
    if install_sh WS_VERSION="$version"; then
        fail "install.sh went through with the release $version"
    fi
    nothing_left
done
# A refusal keeps the ws that stands, and leaves no download.
install_sh
for version in 0.5.0 0.6.0; do
    if install_sh WS_VERSION="$version"; then
        fail "install.sh went through with the release $version over a ws"
    fi
    installed "v0.2.0 linux-amd64"
    ! compgen -G '/usr/local/bin/.ws.*' >/dev/null || fail "a refused install left a download"
done
echo "::endgroup::"

echo "::group::A ws that prints another version than the release is refused, and nothing is left"
if install_sh WS_VERSION=0.7.0 2>"$tmp/refusal"; then
    fail "install.sh went through with a ws that prints another version"
fi
grep -q 'is version "v0.0.1", not v0.7.0' "$tmp/refusal" || fail "install.sh refused the version, but not by its message"
nothing_left
echo "::endgroup::"

echo "::group::A newer latest, and WS_VERSION, install their own release"
echo v0.3.0 >"$tmp/serve/latest"
install_sh
installed "v0.3.0 linux-amd64"
install_sh WS_VERSION=0.2.0
installed "v0.2.0 linux-amd64"
echo "::endgroup::"

echo "::group::A pre-release installs where WS_VERSION names it alone"
install_sh WS_VERSION=v0.4.0-rc.1
installed "v0.4.0-rc.1 linux-amd64"
# GitHub never redirects releases/latest to a pre-release: install.sh refuses one that it finds there.
reset
echo v0.4.0-rc.1 >"$tmp/serve/latest"
if install_sh 2>"$tmp/refusal"; then
    fail "install.sh took a pre-release as the latest release"
fi
grep -q 'GitHub redirects to' "$tmp/refusal" || fail "install.sh refused a pre-release as the latest, but not by its message"
nothing_left
echo v0.3.0 >"$tmp/serve/latest"
echo "::endgroup::"

echo "::group::A version that is no version is refused"
for version in 0.2 0.4.0-rc.0 0.4.0-rc.01 0.4.0-beta.1; do
    if install_sh WS_VERSION="$version"; then
        fail "install.sh went through with WS_VERSION=$version"
    fi
    nothing_left
done
echo "::endgroup::"

echo "::group::/usr/local/bin is made where it is missing, root's and 755"
as_root mv /usr/local/bin "$aside"
install_sh
installed "v0.3.0 linux-amd64"
[ "$(as_root stat -c '%u:%g %a' /usr/local/bin)" = "0:0 755" ] || fail "/usr/local/bin was not made 0:0 and 755"
reset
echo "::endgroup::"

echo "::group::A /usr/local/bin that another user owns, or others may write, is refused, and keeps its owner"
for change in "chown nobody" "chmod o+w" "chmod g+w"; do
    # shellcheck disable=SC2086 # CHANGE is a command and its operand, to split.
    as_root $change /usr/local/bin
    before=$(as_root stat -c '%U %a' /usr/local/bin)
    if install_sh 2>"$tmp/refusal"; then
        fail "install.sh went through with /usr/local/bin after $change"
    fi
    grep -q 'is not root.s alone' "$tmp/refusal" || fail "install.sh refused /usr/local/bin after $change, but not by its message"
    [ "$(as_root stat -c '%U %a' /usr/local/bin)" = "$before" ] || fail "install.sh changed /usr/local/bin after $change"
    reset
done
as_root chmod o+w /usr/local
if install_sh 2>"$tmp/refusal"; then
    fail "install.sh went through with a /usr/local that others may write"
fi
grep -q 'is not root.s alone' "$tmp/refusal" || fail "install.sh refused /usr/local, but not by its message"
nothing_left
echo "::endgroup::"
