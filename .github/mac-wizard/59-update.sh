#!/bin/sh
# --update on the Mac, on the Mac that the cases before leave, whose ws is the job's build, v0.0.0,
# the latest release of standin.sh's server. A clone of the tree, tagged v0.0.1 and built, becomes
# the latest release there. Then ws client macos wizard --update must install it as ~/.local/bin/ws,
# and run again as the new ws. That one installs v0.0.1 on the stand-in ws, so that ws holds it as
# well. The case puts v0.0.0 back, the binary, the stand-in ws's version and the latest release.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
releases=$RUNNER_TEMP/releases
src=$RUNNER_TEMP/update-src
fail=0

# error MESSAGE... prints an ::error::, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# The clone shares the tree's .cache, which holds the pinned Go. The link is excluded, as git's
# ignore of a directory does not match a link: else the build is +dirty. The clone's tag v0.0.0
# goes, for the build to print the tag it sits at.
git clone -q ~/repository/ws "$src" || exit 1
ln -s ~/repository/ws/.cache "$src/.cache"
echo /.cache >>"$src/.git/info/exclude"
git -C "$src" tag -d v0.0.0 >/dev/null
git -C "$src" tag v0.0.1
mkdir -p "$releases/download/v0.0.1"
sh "$src/tools/build" darwin arm64 "$releases/download/v0.0.1/ws-darwin-arm64" || exit 1
cp "$src/install.sh" "$releases/download/v0.0.1/install.sh"
(cd "$releases/download/v0.0.1" && shasum -a 256 ws-darwin-arm64 >ws.sha256)
[ "$("$releases/download/v0.0.1/ws-darwin-arm64" version)" = v0.0.1 ] || error "the build at v0.0.1 prints another version"
cp "$wsbin" "$RUNNER_TEMP/ws.v0.0.0"
echo v0.0.1 >"$releases/latest"

"$wsbin" client macos wizard --update </dev/null
status=$?
[ "$status" = 0 ] || error "ws client macos wizard --update exited $status with v0.0.1 the latest release"
[ "$("$wsbin" version)" = v0.0.1 ] || error "ws client macos wizard --update left ws $("$wsbin" version) on the Mac, not v0.0.1"
[ "$(cat "$ws/ws.version")" = v0.0.1 ] || error "ws client macos wizard --update left ws $(cat "$ws/ws.version") on the stand-in ws, not v0.0.1"

cp "$RUNNER_TEMP/ws.v0.0.0" "$wsbin"
echo v0.0.0 >"$releases/latest"
echo v0.0.0 | sudo tee "$ws/ws.version" >/dev/null
exit "$fail"
