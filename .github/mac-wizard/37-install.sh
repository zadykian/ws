#!/bin/sh
# The ws on ws stage, on the Mac that 36-share.sh leaves. The first run installed this ws's release,
# v0.0.0, on the stand-in ws, once. It ran the release's install.sh at the address of the release
# stand-in. The stand-in ws prints that version. Where ws holds a newer one, v0.0.9, the run must
# stop, naming --update, and install and start nothing.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error::, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

[ "$(wc -l <"$ws/install.log")" -eq 1 ] || error "the first run did not install once: $(cat "$ws/install.log")"
grep -qx 'http://127.0.0.1:22080/download/v0.0.0/install.sh' "$ws/install.log" || error "the first run did not install v0.0.0"
[ "$(cat "$ws/ws.version")" = v0.0.0 ] || error "ws.version is not v0.0.0"

echo v0.0.9 | sudo tee "$ws/ws.version" >/dev/null
units=$(wc -l <"$ws/run.log")
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
[ "$status" != 0 ] || error "ws client macos wizard exited 0 with the newer v0.0.9 on ws"
printf '%s\n' "$out" | grep -qF 'ws client macos wizard --update' || error "the run with v0.0.9 on ws did not name --update: $out"
[ "$(wc -l <"$ws/install.log")" -eq 1 ] && [ "$(cat "$ws/ws.version")" = v0.0.9 ] ||
    error "the run with v0.0.9 on ws installed: $(cat "$ws/install.log")"
[ "$(wc -l <"$ws/run.log")" -eq "$units" ] || error "the run with v0.0.9 on ws started a unit"
echo v0.0.0 | sudo tee "$ws/ws.version" >/dev/null
exit "$fail"
