#!/bin/bash
# ws macos remove lab drops the second server from the Mac. Its directory, daemon, plist,
# state, log, ssh host and known_hosts entry go, and ws's stay, with its daemon still shaking hands.
# remove runs mise bootstrap, as setup runs it. The bootstrap takes lab's daemon and files away, and
# its host out of /etc/ws/ssh_config.
# The runner has no fnox key file, so remove leaves lab's key in the secrets to the user. remove
# refuses a name with no file, and the Mac's last server. Then the plan exits 0.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

lab=${LABEL%.ws}.lab
servers=/etc/ws/servers

# fails WHY ARG...: ws macos remove ARG... must exit 1, with WHY in what it prints.
fails() {
    why=$1
    shift
    status=0
    out=$(ws macos remove "$@" 2>&1) || status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF -e "$why"; then
        echo "::error::ws macos remove $* exited $status, printing '$out', where 1 and '$why' were due"
        exit 1
    fi
}
fails 'no server named box in' box

# known_hosts holds an entry for each server, as ssh's first connection writes it.
key=$(cut -d ' ' -f 1,2 ~/.ssh/id_ed25519.pub)
printf '%s %s\n' lab "$key" ws "$key" >>~/.ssh/known_hosts

ws macos remove lab </dev/null | tee "$RUNNER_TEMP/remove.out" || {
    echo '::error::ws macos remove lab failed'
    exit 1
}
grep -qx 'removed lab' "$RUNNER_TEMP/remove.out" || {
    echo '::error::ws macos remove lab did not say it removed lab'
    exit 1
}
# The runner has no fnox key file, so the secrets' part of remove is left by hand, with lab's key.
grep -q "^By hand: this Mac has no fnox key, .*: from a Mac that has one, drop lab's host key, $key, from the secrets" \
    "$RUNNER_TEMP/remove.out" || {
    echo "::error::ws macos remove lab did not leave the secrets' key by hand"
    exit 1
}

fail=0
gone() {
    echo "::error::$1"
    fail=1
}
[ ! -e "$servers/lab" ] || gone "$servers/lab is still there"
[ -f "$servers/ws/values.yml" ] || gone "the server ws's values.yml is gone"
[ ! -e "/Library/LaunchDaemons/$lab.plist" ] || gone "lab's plist is still there"
! launchctl print "system/$lab" >/dev/null 2>&1 || gone "lab's daemon is still loaded"
for path in /etc/ws/tunnel/lab /var/run/ws-tunnel.lab.state /var/log/ws-tunnel.lab.log; do
    ! sudo test -e "$path" || gone "$path is still there"
done
! grep -qx 'Host lab' /etc/ws/ssh_config || gone "/etc/ws/ssh_config still has a Host lab"
grep -qx 'Host ws' /etc/ws/ssh_config || gone "/etc/ws/ssh_config lost its Host ws"
! ssh-keygen -F lab >/dev/null || gone "known_hosts still has lab"
ssh-keygen -F ws >/dev/null || gone "known_hosts lost ws"
launchctl print "system/$LABEL" >/dev/null || gone "ws's daemon is not loaded"
# shaken: ws's daemon has a handshake. It has 60 s for one, as the daemon may be trying its paths
# again, with a new peer (92-second.sh).
shaken() {
    status=$(sudo "$TUNNEL" macos tunnel status --server ws 2>&1) || true
    printf '%s\n' "$status" | grep -Eq '^latest handshake: [0-9]+ s ago$'
}
for _ in $(seq 1 60); do
    shaken && break
    sleep 1
done
shaken || gone "ws's daemon has no handshake in 60 s; its status gives: $status"
[ "$(ws macos servers | awk 'NR > 1 { print $1 }')" = ws ] || gone "ws macos servers lists more than ws"
[ "$fail" = 0 ] || exit 1

fails "ws is the Mac's one server: with none, the bootstrap prunes no daemon, so remove leaves it" ws
client_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan with ws alone exited $?, not 0"
    exit 1
}
