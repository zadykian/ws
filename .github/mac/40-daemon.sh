#!/bin/bash
# With ws's key, setup.sh installs the daemon.
# The daemon must run: launchd gives it a pid, and its state names the utun it set up.

set -eo pipefail

echo "WS_PUBLIC_KEY=$(cat "$RUNNER_TEMP/ws.pub")" >>~/.config/ws/mac.conf
sh mac/setup.sh || {
    echo "::error::setup.sh with ws's key failed"
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
    status=$(sudo "$TUNNEL" tunnel status 2>&1) || true
    if [ -n "$now" ] && printf '%s\n' "$status" | grep -Eqx 'interface: utun[0-9]+'; then
        printf 'pid %s\n%s\n' "$now" "$status"
        exit 0
    fi
    sleep 1
done
echo "::error::no daemon runs in 30 s: launchd's pid is '$now'; ws tunnel status gives:"
printf '%s\n' "$status"
exit 1
