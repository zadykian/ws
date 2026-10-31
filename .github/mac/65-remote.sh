#!/bin/bash
# With nothing at WS_LAN_ADDRESS, the remote tier.
# WS_LAN_ADDRESS becomes 127.0.0.2, inside WS_LAN as setup.sh requires, where nothing answers:
# macOS's lo0 has 127.0.0.1 alone, and a handshake on the LAN tier fails the step. The daemon
# takes the new tunnel.conf on SIGHUP, in place: it keeps its pid and logs a reload. A daemon
# that died of the SIGHUP, and that launchd started again, would log a start.

set -eo pipefail

pid() { sudo launchctl print "system/$LABEL" | awk '$1 == "pid" { print $3; exit }'; }
was=$(pid) || true
lines=$(sudo awk 'END { print NR }' /var/log/ws-tunnel.log)
sed -i '' 's/^WS_LAN_ADDRESS=.*/WS_LAN_ADDRESS=127.0.0.2/' ~/.config/ws/mac.conf
sh mac/setup.sh 2>&1 | tee "$RUNNER_TEMP/hup.out" ||
    {
        echo '::error::setup.sh with a new WS_LAN_ADDRESS failed'
        exit 1
    }
grep -q "ran launchctl kill SIGHUP system/$LABEL" "$RUNNER_TEMP/hup.out" ||
    {
        echo '::error::setup.sh sent the daemon no SIGHUP for a new tunnel.conf'
        exit 1
    }
remote() {
    status=$(sudo "$TUNNEL" tunnel status 2>&1) || true
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
new=$(sudo tail -n "+$((lines + 1))" /var/log/ws-tunnel.log)
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
