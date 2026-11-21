#!/bin/bash
# install.sh on macOS, as a Mac runs it through sudo. serve-releases serves this clone's build of
# ws as the release v0.2.0, with its ws.sha256, as install-checks.sh serves its stubs. ws is removed
# first, so that the run writes it. The file must be root:wheel and 755, as launchd needs for the
# tunnel's daemon. The last line must name ws client setup, and no download may stay. The build
# is the one that the job's step put there, so the cases after it run as before.

set -eo pipefail

tmp=$(mktemp -d)
trap 'kill "${server:-}" 2>/dev/null || true; rm -rf "$tmp"' EXIT

release=$tmp/serve/download/v0.2.0
mkdir -p "$release"
sh .tools/build darwin arm64 "$release/ws-darwin-arm64"
(cd "$release" && shasum -a 256 ws-darwin-arm64 >ws.sha256)
echo v0.2.0 >"$tmp/serve/latest"
sh .tools/build darwin arm64 "$tmp/serve-releases" ./.tools/serve-releases
"$tmp/serve-releases" "$tmp/serve" 0 "$tmp/port" &
server=$!
for _ in $(seq 50); do
    [ -s "$tmp/port" ] && break
    sleep 0.1
done
url=http://127.0.0.1:$(cat "$tmp/port")

sudo rm -f /usr/local/bin/ws
out=$(echo | sudo env WS_RELEASES_URL="$url" sh install.sh 2>&1) || {
    echo "$out"
    echo "::error::install.sh failed"
    exit 1
}
echo "$out"
[ "${out##*$'\n'}" = "install.sh: next, run: ws client setup" ] || {
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
