#!/bin/sh
# After the first run, mac.conf goes back as standin.sh wrote it, and the first run's answers are
# typed at a terminal of script's, after a bad one. The run must exit 0, write the mac.conf that the
# environment's answers did, and ask the bad one again.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
fail=0

cp "$conf" "$RUNNER_TEMP/mac.conf.env"
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 >"$conf"
# script gives the wizard a terminal of its own, its stdin typed there, and its output on stdout.
# macOS's script writes its stdin's end to the terminal ahead of the lines it still holds, so the
# end comes 5 s after them.
{
    printf '%s\n' '' 10.77.0.0/30 none none '' '' ''
    sleep 5
} | script -q /dev/null sh "$wizard" >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
if [ "$status" != 0 ]; then
    echo "::error::wizard.sh exited $status with the answers typed"
    fail=1
fi
if ! cmp -s "$conf" "$RUNNER_TEMP/mac.conf.env"; then
    echo "::error::wizard.sh wrote another mac.conf from the answers typed than the environment's:"
    diff "$RUNNER_TEMP/mac.conf.env" "$conf"
    fail=1
fi
if ! tr -d '\r' <"$RUNNER_TEMP/tty" | grep -qF 'WS_LAN=10.77.0.0/30 is not an IPv4 network'; then
    echo '::error::wizard.sh took WS_LAN=10.77.0.0/30, or asked it no more'
    fail=1
fi
exit "$fail"
