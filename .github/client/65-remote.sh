#!/bin/bash
# With nothing at WS_LAN_ADDRESS, the remote tier.
# WS_LAN_ADDRESS becomes 127.0.0.2, inside WS_LAN as values.yml's check requires, where nothing
# answers. macOS's lo0 has 127.0.0.1 alone, and a handshake on the LAN tier fails the step. mise
# bootstrap writes the new tunnel.conf, and launchd's unit sends the daemon a SIGHUP, which it takes
# in place: it keeps its pid and logs a reload. A daemon that died of the SIGHUP, and that launchd
# started again, would log a start.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

pid() { sudo launchctl print "system/$LABEL" | awk '$1 == "pid" { print $3; exit }'; }
was=$(pid) || true
lines=$(sudo awk 'END { print NR }' /var/log/ws-tunnel.ws.log)
conf_set /etc/ws/values.yml 'WS_LAN_ADDRESS: 127.0.0.2'
client_bootstrap || {
    echo '::error::mise bootstrap with a new WS_LAN_ADDRESS failed'
    exit 1
}
client_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan after the bootstrap with a new WS_LAN_ADDRESS exited $?, not 0"
    exit 1
}
grep -qx 'WS_LAN_ADDRESS=127.0.0.2' /etc/ws/tunnel/ws/tunnel.conf ||
    {
        echo '::error::mise bootstrap wrote no new tunnel.conf'
        exit 1
    }
remote() {
    status=$(sudo "$TUNNEL" macos tunnel status 2>&1) || true
    printf '%s\n' "$status" | grep -qx 'tier: remote' &&
        printf '%s\n' "$status" | grep -qx 'endpoint: 127.0.0.1:51820' &&
        printf '%s\n' "$status" | grep -Eq '^latest handshake: [0-9]+ s ago$'
}
for _ in $(seq 1 60); do
    remote && break
    sleep 1
done
printf '%s\n' "$status"
remote || {
    echo '::error::no handshake on the remote tier in 60 s'
    exit 1
}
new=$(sudo tail -n "+$((lines + 1))" /var/log/ws-tunnel.ws.log)
now=$(pid) || true
if [ -z "$was" ] || [ "$now" != "$was" ]; then
    echo "::error::the daemon was pid '$was' before the SIGHUP, and is pid '$now'"
    exit 1
fi
grep -q 'read new settings' <<<"$new" ||
    {
        echo '::error::the daemon logged no reload of tunnel.conf'
        exit 1
    }
! grep -q ' started on ' <<<"$new" ||
    {
        echo '::error::the daemon started again on the SIGHUP'
        exit 1
    }
! grep -q 'the handshake came from' <<<"$new" ||
    {
        echo '::error::something answered 127.0.0.2 on the LAN tier'
        exit 1
    }
