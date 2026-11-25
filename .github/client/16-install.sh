#!/bin/bash
# install.sh on macOS, as a Mac runs it through sudo. serve-releases serves this clone's build of
# ws as a release, with its ws.sha256, as install-checks.sh serves its stubs. install.sh refuses a
# ws that prints another version than its release, and Go stamps the tag of HEAD as ws version. So
# the release is the one that the job's build prints, as on a tag. On a pull request it prints
# none. The case then tags HEAD v0.2.0 in the clone for its build. At its end it puts the job's
# build back, so that the cases after it run as before. ws is removed first, so that the run writes
# it. The file must be root:wheel and 755, as launchd needs for the tunnel's daemon. The last line
# must name ws macos setup, and no download may stay.

set -eo pipefail

tmp=$(mktemp -d)
tagged=
trap 'kill "${server:-}" 2>/dev/null || true
[ -z "$tagged" ] || git update-ref -d "refs/tags/$tagged"
rm -rf "$tmp"' EXIT

# update-ref makes the tag with no signature, whatever git's config asks of a tag. Its empty old
# value fails where the clone has the tag.
tag=$(/usr/local/bin/ws version)
if ! [[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]]; then
    tag=v0.2.0
    git update-ref "refs/tags/$tag" HEAD ''
    tagged=$tag
fi

release=$tmp/serve/download/$tag
mkdir -p "$release"
sh .tools/build darwin arm64 "$release/ws-darwin-arm64"
(cd "$release" && shasum -a 256 ws-darwin-arm64 >ws.sha256)
# A pre-release is never the latest, so WS_VERSION names it.
version=
case $tag in
*-rc.*) version=$tag ;;
*) echo "$tag" >"$tmp/serve/latest" ;;
esac
sh .tools/build darwin arm64 "$tmp/serve-releases" ./.tools/serve-releases
"$tmp/serve-releases" "$tmp/serve" 0 "$tmp/port" &
server=$!
for _ in $(seq 50); do
    [ -s "$tmp/port" ] && break
    sleep 0.1
done
settings=(WS_RELEASES_URL="http://127.0.0.1:$(cat "$tmp/port")")
[ -z "$version" ] || settings+=(WS_VERSION="$version")

sudo rm -f /usr/local/bin/ws
out=$(echo | sudo env "${settings[@]}" sh install.sh 2>&1) || {
    echo "$out"
    echo "::error::install.sh failed"
    exit 1
}
echo "$out"
[ "${out##*$'\n'}" = "install.sh: next, run: ws macos setup" ] || {
    echo "::error::install.sh's last line is not the next command"
    exit 1
}
got=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/ws)
[ "$got" = "root:wheel 755" ] || {
    echo "::error::/usr/local/bin/ws is $got, not root:wheel 755"
    exit 1
}
cmp "$release/ws-darwin-arm64" /usr/local/bin/ws
/usr/local/bin/ws version
! compgen -G '/usr/local/bin/.ws.*' >/dev/null || {
    echo "::error::install.sh left a download in /usr/local/bin"
    exit 1
}

if [ -n "$tagged" ]; then
    git update-ref -d "refs/tags/$tagged"
    tagged=
    sh .tools/install-build darwin arm64
fi
