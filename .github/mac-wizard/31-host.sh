#!/bin/sh
# The Host key stage on the Mac that 30-run.sh leaves, whose known_hosts holds ws's key under ws.
# Without that line, a run with no terminal must stop where WS_HOST_KEY is missing or names another
# key. --check, under StrictHostKeyChecking accept-new in ~/.ssh/config, must show the line it would
# add. Where known_hosts is a link, it must stop, as a run must, which prints that line for its
# file. With a conf of no way to ws, a run must stop, naming
# roles/server/ubuntu/cable/router.md. None may change known_hosts. WS_HOST_KEY=yes, with the
# Mac's own conf, must then add the same line back. With a conf whose way to ws doesn't
# answer, ws's address now must come from WS_FIRST_ADDRESS. That run's closing list must name
# the server's WS_TUNNEL_PORT in the router's forwards, and ws's lease at no address, as its file has no
# LAN. As that port is not 51820, the run must name WS_TUNNEL_PORT in ws's values.yml before ws's
# run. A cloud VM's copy must name the server file's way to ws, and no lease. The Mac's own conf
# then puts the way back.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
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
        error "ws client macos wizard exited $status, where $1 and '$2' were due"
    cmp -s "$hosts" "$RUNNER_TEMP/hosts.without" || error "ws client macos wizard changed $hosts"
}

# linked WANT [ARG]: the run, with known_hosts a link to a file, must stop, naming that file, then
# WANT, and leave the link.
linked() {
    want=$1
    shift
    out=$(WS_HOST_KEY=yes "$wsbin" client macos wizard "$@" </dev/null 2>&1)
    status=$?
    expect 1 "Host key: $hosts is a link to $RUNNER_TEMP/hosts.file, which the wizard leaves: $want"
    [ -L "$hosts" ] || error "ws client macos wizard $* replaced the link"
}

cp "$hosts" "$RUNNER_TEMP/hosts"
grep -vxF "$line" "$RUNNER_TEMP/hosts" >"$RUNNER_TEMP/hosts.without"
cp "$RUNNER_TEMP/hosts.without" "$hosts"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 'ws client macos wizard: Host key: ws at localhost:22022 offers the ed25519 key '
out=$(WS_HOST_KEY=SHA256:x "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 "ws client macos wizard: Host key: 'SHA256:x' is neither yes nor $fp"
cp ~/.ssh/config "$RUNNER_TEMP/ssh.config"
printf '%s\n' 'Host *' '    StrictHostKeyChecking accept-new' >>~/.ssh/config
out=$("$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
cp "$RUNNER_TEMP/ssh.config" ~/.ssh/config
expect 0 "+$line"
mv "$hosts" "$RUNNER_TEMP/hosts.file"
ln -s "$RUNNER_TEMP/hosts.file" "$hosts"
linked "a run would print ws's line for its file" --check
! printf '%s\n' "$out" | grep -qF "$line" || error 'ws client macos wizard --check printed the line for the link'
linked 'add this line to its file, then run the wizard again'
printf '%s\n' "$out" | grep -qxF "$line" || error "ws client macos wizard printed no line for the link's file"
rm "$hosts"
mv "$RUNNER_TEMP/hosts.file" "$hosts"
none=$RUNNER_TEMP/confs/none/values.yml
conf_print /etc/ws/values.yml |
    sed -e 's/^WS_REMOTE:.*/WS_REMOTE: ""/' -e 's/^WS_REMOTE_PORT:.*/WS_REMOTE_PORT: ""/' |
    conf_write "$none"
out=$(WS_VALUES=$none "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 "and no WS_REMOTE_PORT; set those in the server's values.yml as roles/server/ubuntu/cable/router.md has them"
out=$(WS_HOST_KEY=yes "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "ws client macos wizard with WS_HOST_KEY=yes exited $status"
cmp -s "$hosts" "$RUNNER_TEMP/hosts" || error "ws client macos wizard with WS_HOST_KEY=yes wrote another $hosts"

conf=$RUNNER_TEMP/confs/first/values.yml
conf_print /etc/ws/values.yml | sed -e 's/^WS_REMOTE:.*/WS_REMOTE: 192.0.2.1/' \
    -e 's/^WS_TUNNEL_PORT:.*/WS_TUNNEL_PORT: 51821/' | conf_write "$conf"
cp "$RUNNER_TEMP/hosts.without" "$hosts"
out=$(WS_VALUES=$conf "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 "doesn't answer at 192.0.2.1:22022 with an ed25519 key, and no terminal is there to ask"
out=$(WS_VALUES=$conf WS_FIRST_ADDRESS=ws.test "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 'ws client macos wizard: Host key: WS_FIRST_ADDRESS=ws.test is not an IPv4 address'
out=$(WS_VALUES=$conf WS_FIRST_ADDRESS=127.0.0.1 WS_HOST_KEY=$fp "$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "ws client macos wizard with WS_FIRST_ADDRESS=127.0.0.1 exited $status"
cmp -s "$hosts" "$RUNNER_TEMP/hosts" || error "ws client macos wizard with WS_FIRST_ADDRESS wrote another $hosts"
sudo grep -qx 'WS_TUNNEL_PORT: 51821' "$RUNNER_TEMP/ws/root"/etc/ws/values.yml ||
    error "ws client macos wizard with WS_TUNNEL_PORT=51821 named no WS_TUNNEL_PORT in ws's values.yml"
lease="- ws's lease: static, as roles/server/ubuntu/cable/router.md has it, as this run reached ws at 127.0.0.1."
forwards="- The router's forwards to ws, UDP 51821 and TCP 22022, for 192.0.2.1,"
if ! printf '%s\n' "$out" | grep -qxF -e "$lease" ||
    ! printf '%s\n' "$out" | grep -qF -e "$forwards"; then
    error "ws client macos wizard with WS_FIRST_ADDRESS listed another lease or forwards by hand"
fi
conf_print "$conf" | sed 's/^WS_TARGET:.*/WS_TARGET: cloud/' | conf_write "$RUNNER_TEMP/confs/cloud/values.yml"
cp "$RUNNER_TEMP/hosts.without" "$hosts"
out=$(WS_VALUES=$RUNNER_TEMP/confs/cloud/values.yml WS_FIRST_ADDRESS=127.0.0.1 WS_HOST_KEY=$fp "$wsbin" client macos wizard \
    </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
way="- ws's address: WS_REMOTE in the server's values.yml must lead to ws, as this run reached it at 127.0.0.1"
if [ "$status" != 0 ] || ! printf '%s\n' "$out" | grep -qF -e "$way (roles/client/macos/values/README.md)." ||
    printf '%s\n' "$out" | grep -q "^- ws's lease: "; then
    error "ws client macos wizard on a cloud VM with WS_FIRST_ADDRESS exited $status, or named its lease"
fi
sudo sed -i '' -e '/^WS_SECURITY_OUTBOUND:/d' -e '/^WS_TUNNEL_PORT:/d' \
    "$RUNNER_TEMP/ws/root"/etc/ws/values.yml
sudo rm -f "$RUNNER_TEMP/ws/values.log"
rm -r "$RUNNER_TEMP/confs/cloud"
"$wsbin" client macos wizard </dev/null || error "ws client macos wizard with the Mac's own values.yml exited $?"
grep -qx '    HostName 10.99.0.1' /etc/ws/ssh_config || error 'ws client macos wizard wrote no /etc/ws/ssh_config for the tunnel'
rm -f "$RUNNER_TEMP/hosts" "$RUNNER_TEMP/hosts.without" "${none%/*}" "${conf%/*}" "$RUNNER_TEMP/ssh.config"
exit "$fail"
