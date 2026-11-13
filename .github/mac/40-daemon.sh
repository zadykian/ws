#!/bin/bash
# With ws's key, ws client macos apply installs the daemon.
# The daemon must run: launchd gives it a pid, and its state names the utun it set up.

set -eo pipefail

echo "WS_PUBLIC_KEY=$(cat "$RUNNER_TEMP/ws.pub")" >>~/.config/ws/mac.conf
ws client macos apply || {
    echo "::error::ws client macos apply with ws's key failed"
    exit 1
}
launchctl print "system/$LABEL" >/dev/null ||
    {
        echo '::error::the daemon is not loaded'
        exit 1
    }
[ "$(stat -f '%Su:%Sg %Lp' "/Library/LaunchDaemons/$LABEL.plist")" = "root:wheel 644" ] ||
    {
        echo '::error::the plist is not root:wheel 644'
        exit 1
    }
plutil -lint "/Library/LaunchDaemons/$LABEL.plist"
pid() { sudo launchctl print "system/$LABEL" | awk '$1 == "pid" { print $3; exit }'; }
for _ in $(seq 1 30); do
    now=$(pid) || true
    status=$(sudo "$TUNNEL" client macos tunnel status 2>&1) || true
    if [ -n "$now" ] && printf '%s\n' "$status" | grep -Eqx 'interface: utun[0-9]+'; then
        printf 'pid %s\n%s\n' "$now" "$status"
        exit 0
    fi
    sleep 1
done
echo "::error::no daemon runs in 30 s: launchd's pid is '$now'; ws client macos tunnel status gives:"
printf '%s\n' "$status"
exit 1
