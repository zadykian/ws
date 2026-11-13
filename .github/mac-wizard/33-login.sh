#!/bin/sh
# The Login stage on the Mac that 31-host.sh leaves, where the runner on the stand-in ws takes the
# Mac's key. With another key alone in the user's file, a run must stop, naming the Mac's key, and
# end with ssh's error, its \r dropped. Then the user's line for the Mac's key carries a forced
# command, as cloud-init writes it, beside the other key. --check must say that a run would stop,
# and a run must stop, with no terminal and at one of script's. None may change the user's
# authorized_keys. The wizard adds no key, so the case replaces that file by the stand-in gh's
# keys, as Ubuntu's installer imports them for its user. A run must then say nothing of the stage,
# and keep the file. With another host key under ws in known_hosts, a run must stop, naming
# ssh-keygen -R ws, and change nothing, with StrictHostKeyChecking no in ~/.ssh/config too.

set -u

wsbin=$HOME/.local/bin/ws
keys=~/.ssh/authorized_keys
hosts=$HOME/.ssh/known_hosts
pub=$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STATUS WANT FILE KEPT: the run before, whose output is out, must have exited STATUS, with
# WANT in out, and left FILE as KEPT holds it.
expect() {
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep -qF -e "$2" && [ "$status" = "$1" ] ||
        error "ws client macos wizard exited $status, where $1 and '$2' were due"
    cmp -s "$3" "$4" || error "ws client macos wizard changed $3"
}

ssh-keygen -q -t ed25519 -N '' -C other -f "$RUNNER_TEMP/other" </dev/null
other=$(cut -d ' ' -f 1-2 "$RUNNER_TEMP/other.pub")
stop="ws client macos wizard: Login: $(id -un) on ws doesn't log in by the Mac's key, $HOME/.ssh/id_ed25519.pub,"
echo "$other other" >"$RUNNER_TEMP/user.keys"
cp "$RUNNER_TEMP/user.keys" "$keys"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 "$stop" "$keys" "$RUNNER_TEMP/user.keys"
printf '%s\n' "$out" | grep -F "$stop" |
    grep -q "again\. ssh's error: .*Permission denied (.*)\.\$" ||
    error "ws client macos wizard's stop at Login ended in no error of ssh's, or kept its \\r"
printf '%s\n' "$other other" "no-port-forwarding,command=\"echo no;exit 142\" $pub mac" \
    >"$RUNNER_TEMP/user.keys"
cp "$RUNNER_TEMP/user.keys" "$keys"
out=$("$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
expect 0 "ws client macos wizard: Login: $(id -un) on ws doesn't log in by the Mac's key; a run would stop" \
    "$keys" "$RUNNER_TEMP/user.keys"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
expect 1 "$stop" "$keys" "$RUNNER_TEMP/user.keys"
# script gives the wizard a terminal of its own, at which the stage asks nothing.
{ sleep 5; } | script -q /dev/null "$wsbin" --plain client macos wizard >"$RUNNER_TEMP/tty" 2>&1
status=$?
out=$(tr -d '\r' <"$RUNNER_TEMP/tty")
expect 1 "$stop" "$keys" "$RUNNER_TEMP/user.keys"
install -m 600 "$RUNNER_TEMP/gh/keys" "$keys"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
if [ "$status" != 0 ] || printf '%s\n' "$out" | grep -q "^ws client macos wizard: Login: "; then
    error "ws client macos wizard exited $status, or said of Login, with the account's keys in the user's file"
fi
cmp -s "$keys" "$RUNNER_TEMP/gh/keys" || error "ws client macos wizard changed $keys"

cp "$hosts" "$RUNNER_TEMP/hosts"
sed "s|^ws .*|ws $other|" "$RUNNER_TEMP/hosts" >"$hosts"
cp "$hosts" "$RUNNER_TEMP/hosts.other"
cp ~/.ssh/config "$RUNNER_TEMP/ssh.config"
printf '%s\n' 'Host *' '    StrictHostKeyChecking no' >>~/.ssh/config
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
cp "$RUNNER_TEMP/ssh.config" ~/.ssh/config
expect 1 'offers another host key than known_hosts holds under ws: where ws was installed again,' \
    "$hosts" "$RUNNER_TEMP/hosts.other"
cp "$RUNNER_TEMP/hosts" "$hosts"
rm -f "$RUNNER_TEMP/other" "$RUNNER_TEMP/other.pub" "$RUNNER_TEMP"/user.keys "$RUNNER_TEMP"/hosts* \
    "$RUNNER_TEMP/ssh.config" "$RUNNER_TEMP/tty"
exit "$fail"
