#!/bin/sh
# The ws on ws stage, on the Mac that 36-share.sh leaves. The first run installed this ws's release,
# v0.0.0, on the stand-in ws, once, through the release's install.sh at the address of the release
# stand-in, and the stand-in ws prints that version. Where ws holds another version, v0.0.9, with a
# full run of its own, a run must install v0.0.0 again and run ws server apply again.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
stamp=$ws/root/var/lib/ws-wizard/run
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
echo "v0.0.9 $(sudo cut -d ' ' -f 2- "$stamp")" | sudo tee "$stamp" >/dev/null
units=$(wc -l <"$ws/run.log")
"$wsbin" mac wizard </dev/null
status=$?
[ "$status" = 0 ] || error "ws mac wizard exited $status with v0.0.9 on ws"
[ "$(wc -l <"$ws/install.log")" -eq 2 ] && [ "$(cat "$ws/ws.version")" = v0.0.0 ] ||
    error "the run with v0.0.9 on ws did not install v0.0.0: $(cat "$ws/install.log")"
[ "$(wc -l <"$ws/run.log")" -eq $((units + 1)) ] || error "the run with v0.0.9 on ws started no unit"
sudo grep -q '^v0.0.0 0 ' "$stamp" || error "the run with v0.0.9 on ws left the stamp $(sudo cat "$stamp")"
exit "$fail"
