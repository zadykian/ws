#!/bin/sh
# The Tunnel stage on the Mac that 43-terminal.sh leaves, whose tunnel has a handshake with the
# stand-in ws. mac.conf's WS_PUBLIC_KEY goes first. With a new key for the stand-in's wg0, as a ws
# installed again has, --check must show that line replaced, and change nothing. A run must replace
# it in place, have ws-tunnel take the key, and log in to ws through the tunnel. Without wg0's peer,
# as a tagged run that failed leaves it, a run must run bootstrap.sh --tags wireguard and restart
# ws-tunnel. Without the stand-in's mac.pub too, it must write mac.pub first. Where nothing answers
# on UDP 51820, a run must stop within a minute, naming it, the router's forward and the Background.
# Once it answers, a run must exit 0, and the next print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
peer=$ws/root/etc/wireguard/mac.pub
wg=/usr/local/libexec/ws-tunnel/wg
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard [ARG]: a run of the wizard with no terminal. It prints the output, and sets out and status.
wizard() {
    out=$(sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# logins prints how many logins to ws went through the tunnel.
logins() {
    cat "$ws/tunnel.log" 2>/dev/null | wc -l
}

# standin_set ARG...: wg set on the stand-in ws's wg0, as root.
standin_set() {
    sudo sh -c '"$0" set "$(cat "$1/utun")" "$2" "$3" ${4+"$4"}' "$wg" "$ws" "$@"
}

{ grep '^WS_PUBLIC_KEY=' "$conf" && grep -v '^WS_PUBLIC_KEY=' "$conf"; } >"$RUNNER_TEMP/first"
cp "$RUNNER_TEMP/first" "$conf"
old=$(cat "$ws/wg0")
sudo sh -c 'umask 077 && "$0" genkey >"$1/ws.key" && "$0" pubkey <"$1/ws.key" >"$1/wg0"' "$wg" "$ws"
new=$(cat "$ws/wg0")
standin_set private-key "$ws/ws.key"
wizard --check
printf '%s\n' "$out" | grep -qx -- "-WS_PUBLIC_KEY=$old" &&
    printf '%s\n' "$out" | grep -qx "+WS_PUBLIC_KEY=$new" && cmp -s "$conf" "$RUNNER_TEMP/first" &&
    [ "$status" = 0 ] ||
    error "--check exited $status, showed no WS_PUBLIC_KEY of wg0's new key, or changed mac.conf"
n=$(logins)
wizard
sed "1s|.*|WS_PUBLIC_KEY=$new|" "$RUNNER_TEMP/first" | cmp -s - "$conf" && [ "$status" = 0 ] &&
    printf '%s\n' "$out" | grep -q '^setup\.sh: ran launchctl kill SIGHUP ' &&
    [ "$(logins)" -eq $((n + 1)) ] ||
    error "a run with wg0's new key exited $status, or put no WS_PUBLIC_KEY in its line's place"

mac=$(sudo sh -c '"$0" pubkey <"$1"' "$wg" /etc/ws-tunnel/private.key)
tagged=$(grep -cx -- '--tags wireguard' "$ws/bootstrap.log")
standin_set peer "$mac" remove
wizard
printf '%s\n' "$out" | grep -q '^wizard\.sh: Tunnel: ran launchctl kickstart -k ' &&
    ! printf '%s\n' "$out" | grep -q '^wizard-remote\.sh: wrote ' && [ "$status" = 0 ] &&
    [ "$(grep -cx -- '--tags wireguard' "$ws/bootstrap.log")" -eq $((tagged + 1)) ] ||
    error "a run with mac.pub but without wg0's peer exited $status, or ran no --tags wireguard"

sudo rm "$peer"
standin_set peer "$mac" remove
wizard
printf '%s\n' "$out" | grep -q '^wizard\.sh: Tunnel: ran launchctl kickstart -k ' &&
    [ "$(sudo cat "$peer")" = "$mac" ] &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags wireguard' ] && [ "$status" = 0 ] ||
    error "a run without mac.pub on ws exited $status, or gave wg0 no peer and ws-tunnel no restart"

standin_set listen-port 51821
sudo launchctl kickstart -k system/com.github.zadykian.ws-tunnel
start=$(date +%s)
wizard
printf '%s\n' "$out" |
    grep -q 'within 30 s\. ws must take UDP 51820, .* router.s forward, .* in the Background' &&
    [ "$status" = 1 ] && [ $(($(date +%s) - start)) -lt 60 ] ||
    error "a run with nothing on UDP 51820 exited $status, or named no port, forward and Background"
standin_set listen-port 51820
wizard
[ "$status" = 0 ] || error "a run once UDP 51820 answers again exited $status"
wizard
[ "$status" = 0 ] && [ -z "$out" ] ||
    error "the last run exited $status, or printed what it should not"
rm -f "$RUNNER_TEMP/first"
exit "$fail"
