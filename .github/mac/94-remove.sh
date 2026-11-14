#!/bin/bash
# ws client macos remove lab drops the second server from the Mac. Its directory, daemon, plist,
# state, log, ssh host and known_hosts entry go, and ws's stay, with its daemon still shaking hands.
# remove refuses a name with no file, and the Mac's last server.

set -eo pipefail

lab=${LABEL%.ws}.lab
servers=$HOME/.config/ws/servers

# fails WHY ARG...: ws client macos remove ARG... must exit 1, with WHY in what it prints.
fails() {
    why=$1
    shift
    status=0
    out=$(ws client macos remove "$@" 2>&1) || status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$why"; then
        echo "::error::ws client macos remove $* exited $status, printing '$out', where 1 and '$why' were due"
        exit 1
    fi
}
fails 'no server named box in' box

# known_hosts holds an entry for each server, as ssh's first connection writes it.
key=$(cut -d ' ' -f 1,2 ~/.ssh/id_ed25519.pub)
printf '%s %s\n' lab "$key" ws "$key" >>~/.ssh/known_hosts

ws client macos remove lab | tee "$RUNNER_TEMP/remove.out" || {
    echo '::error::ws client macos remove lab failed'
    exit 1
}
grep -qx 'removed lab' "$RUNNER_TEMP/remove.out" || {
    echo '::error::ws client macos remove lab did not say it removed lab'
    exit 1
}

fail=0
gone() {
    echo "::error::$1"
    fail=1
}
[ ! -e "$servers/lab" ] || gone "$servers/lab is still there"
[ -f "$servers/ws/server.conf" ] || gone "ws's server.conf is gone"
[ ! -e "/Library/LaunchDaemons/$lab.plist" ] || gone "lab's plist is still there"
! launchctl print "system/$lab" >/dev/null 2>&1 || gone "lab's daemon is still loaded"
for path in /etc/ws-tunnel/lab /var/run/ws-tunnel.lab.state /var/log/ws-tunnel.lab.log; do
    ! sudo test -e "$path" || gone "$path is still there"
done
! grep -qx 'Host lab' ~/.ssh/config.d/ws || gone "config.d/ws still has a Host lab"
grep -qx 'Host ws' ~/.ssh/config.d/ws || gone "config.d/ws lost its Host ws"
! ssh-keygen -F lab >/dev/null || gone "known_hosts still has lab"
ssh-keygen -F ws >/dev/null || gone "known_hosts lost ws"
launchctl print "system/$LABEL" >/dev/null || gone "ws's daemon is not loaded"
status=$(sudo "$TUNNEL" client macos tunnel status --server ws 2>&1) || true
printf '%s\n' "$status" | grep -Eq '^latest handshake: [0-9]+ s ago$' ||
    gone "ws's daemon has no handshake; its status gives: $status"
[ "$(ws client macos servers | awk 'NR > 1 { print $1 }')" = ws ] || gone "ws client macos servers lists more than ws"
[ "$fail" = 0 ] || exit 1

fails "ws is the Mac's one server: apply needs one, so remove leaves it" ws
out=$(ws client macos apply --check 2>&1) || {
    echo "::error::ws client macos apply --check failed with ws alone:"
    printf '%s\n' "$out"
    exit 1
}
if [ -n "$out" ]; then
    echo '::error::ws client macos apply --check printed what it should not with ws alone:'
    printf '%s\n' "$out"
    exit 1
fi
