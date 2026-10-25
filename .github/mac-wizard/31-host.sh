#!/bin/sh
# The Host key stage on the Mac that 30-run.sh leaves, whose known_hosts holds ws's key under ws.
# Without that line, a run with no terminal must stop where WS_HOST_KEY is missing or names another
# key, and --check must show the line it would add. With a mac.conf of no way to ws, a run must
# stop, naming docs/router.md. None may change known_hosts. WS_HOST_KEY=yes, with the Mac's own
# mac.conf, must then add the line back as it was. With a mac.conf whose way to ws doesn't answer,
# ws's address now must come from WS_FIRST_ADDRESS. The Mac's own mac.conf then puts the way back.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
hosts=$HOME/.ssh/known_hosts
line="ws $(cut -d ' ' -f 1-2 "$RUNNER_TEMP/ws/host_key.pub")"
fp=$(cat "$RUNNER_TEMP/ws/fingerprint")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STATUS WANT: the run before, whose output is out, must have exited STATUS, with WANT in
# out, and left known_hosts without ws's line.
expect() {
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep -qF -e "$2" && [ "$status" = "$1" ] ||
        error "wizard.sh exited $status, where $1 and '$2' were due"
    cmp -s "$hosts" "$RUNNER_TEMP/hosts.without" || error "wizard.sh changed $hosts"
}

cp "$hosts" "$RUNNER_TEMP/hosts"
grep -vxF "$line" "$RUNNER_TEMP/hosts" >"$RUNNER_TEMP/hosts.without"
cp "$RUNNER_TEMP/hosts.without" "$hosts"
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 'wizard.sh: Host key: ws at localhost:22022 offers the ed25519 key '
out=$(WS_HOST_KEY=SHA256:x sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 "wizard.sh: Host key: 'SHA256:x' is neither yes nor $fp"
out=$(sh "$wizard" --check </dev/null 2>&1)
status=$?
expect 0 "+$line"
none=$RUNNER_TEMP/none.conf
sed -e 's/^WS_REMOTE=.*/WS_REMOTE=/' -e 's/^WS_REMOTE_PORT=.*/WS_REMOTE_PORT=/' \
    ~/.config/ws/mac.conf >"$none"
out=$(WS_CONF=$none sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 'and no WS_REMOTE_PORT; set those in mac.conf as docs/router.md has them'
out=$(WS_HOST_KEY=yes sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "wizard.sh with WS_HOST_KEY=yes exited $status"
cmp -s "$hosts" "$RUNNER_TEMP/hosts" || error "wizard.sh with WS_HOST_KEY=yes wrote another $hosts"

conf=$RUNNER_TEMP/first.conf
sed 's/^WS_REMOTE=.*/WS_REMOTE=192.0.2.1/' ~/.config/ws/mac.conf >"$conf"
cp "$RUNNER_TEMP/hosts.without" "$hosts"
out=$(WS_CONF=$conf sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 "doesn't answer at 192.0.2.1:22022 with an ed25519 key, and no terminal is there to ask"
out=$(WS_CONF=$conf WS_FIRST_ADDRESS=ws.test sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 'wizard.sh: Host key: WS_FIRST_ADDRESS=ws.test is not an IPv4 address'
out=$(WS_CONF=$conf WS_FIRST_ADDRESS=127.0.0.1 WS_HOST_KEY=$fp sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "wizard.sh with WS_FIRST_ADDRESS=127.0.0.1 exited $status"
cmp -s "$hosts" "$RUNNER_TEMP/hosts" || error "wizard.sh with WS_FIRST_ADDRESS wrote another $hosts"
sh "$wizard" </dev/null || error "wizard.sh with the Mac's own mac.conf exited $?"
grep -qx '    HostName localhost' ~/.ssh/config.d/ws || error 'wizard.sh put no way to ws back'
rm -f "$RUNNER_TEMP/hosts" "$RUNNER_TEMP/hosts.without" "$none" "$conf"
exit "$fail"
