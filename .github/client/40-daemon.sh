#!/bin/bash
# With ws's key, mise bootstrap installs the daemon, and its plan then exits 0.
# The daemon must run: launchd gives it a pid, and its state names the utun it set up.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

conf_set /etc/ws/values.yml "WS_PUBLIC_KEY: $(cat "$RUNNER_TEMP/ws.pub")"
client_bootstrap || {
    echo "::error::mise bootstrap with ws's key failed"
    exit 1
}
client_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan after the bootstrap with ws's key exited $?, not 0"
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
for want in "/etc/ws/tunnel/ws root:wheel 755" "/etc/ws/tunnel/ws/routes.d root:wheel 755" \
    "/etc/ws/tunnel/ws/tunnel.conf root:wheel 644"; do
    got=$(stat -f '%N %Su:%Sg %Lp' "${want%% *}")
    [ "$got" = "$want" ] || {
        echo "::error::$got, not $want"
        exit 1
    }
done
pid() { sudo launchctl print "system/$LABEL" | awk '$1 == "pid" { print $3; exit }'; }
for _ in $(seq 1 30); do
    now=$(pid) || true
    status=$(sudo "$TUNNEL" macos tunnel status 2>&1) || true
    if [ -n "$now" ] && printf '%s\n' "$status" | grep -Eqx 'interface: utun[0-9]+'; then
        printf 'pid %s\n%s\n' "$now" "$status"
        exit 0
    fi
    sleep 1
done
echo "::error::no daemon runs in 30 s: launchd's pid is '$now'; ws macos tunnel status gives:"
printf '%s\n' "$status"
exit 1
