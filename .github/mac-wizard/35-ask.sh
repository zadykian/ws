#!/bin/sh
# After the first run, mac.conf goes back as standin.sh wrote it. The first run's answers are
# typed at a terminal of script's, after a bad one. The run must exit 0, write the mac.conf that the
# environment's answers did, and ask the bad one again.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
conf=$HOME/.config/ws/mac.conf
fail=0

conf_print "$conf" >"$RUNNER_TEMP/mac.conf.env"
printf '%s\n' "WS_USER=$(id -un)" WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub' |
    conf_write "$conf"
# script gives the wizard a terminal of its own, its stdin typed there, and its output on stdout.
# macOS's script writes its stdin's end to the terminal ahead of the lines it still holds, so the
# end comes 5 s after them.
{
    printf '%s\n' '' 10.77.0.0/30 none none '' '' ''
    sleep 5
} | script -q /dev/null "$wsbin" --plain client macos wizard >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
if [ "$status" != 0 ]; then
    echo "::error::ws client macos wizard exited $status with the answers typed"
    fail=1
fi
if ! conf_print "$conf" | cmp -s - "$RUNNER_TEMP/mac.conf.env"; then
    echo "::error::ws client macos wizard wrote another mac.conf from the answers typed than the environment's:"
    conf_print "$conf" >"$RUNNER_TEMP/mac.conf.got"
    diff "$RUNNER_TEMP/mac.conf.env" "$RUNNER_TEMP/mac.conf.got"
    fail=1
fi
if ! tr -d '\r' <"$RUNNER_TEMP/tty" | grep -qF 'WS_LAN=10.77.0.0/30 is not an IPv4 network'; then
    echo '::error::ws client macos wizard took WS_LAN=10.77.0.0/30, or asked it no more'
    fail=1
fi
exit "$fail"
