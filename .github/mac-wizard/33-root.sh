#!/bin/sh
# The Root's key stage on the Mac that 31-host.sh leaves, where root on the stand-in ws takes the
# Mac's key. Once root's line for the key carries a forced command, as cloud-init writes it, beside
# another key, --check must say that a run would stop, and a run with no terminal must stop,
# naming the Mac's key. None may change root's authorized_keys. The wizard adds no key, so the case
# replaces that file by the stand-in gh's keys, as the README's step copies the installer's user's
# to root: a run must then say nothing of the stage, and keep the file. With another host key under
# ws in known_hosts, a run must stop, naming ssh-keygen -R ws, and change nothing, with
# StrictHostKeyChecking no in ~/.ssh/config too.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
keys=~root/.ssh/authorized_keys
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
        error "wizard.sh exited $status, where $1 and '$2' were due"
    sudo cmp -s "$3" "$4" || error "wizard.sh changed $3"
}

ssh-keygen -q -t ed25519 -N '' -C other -f "$RUNNER_TEMP/other" </dev/null
other=$(cut -d ' ' -f 1-2 "$RUNNER_TEMP/other.pub")
printf '%s\n' "$other other" "no-port-forwarding,command=\"echo no;exit 142\" $pub mac" \
    >"$RUNNER_TEMP/root.keys"
sudo cp "$RUNNER_TEMP/root.keys" "$keys"
out=$(sh "$wizard" --check </dev/null 2>&1)
status=$?
expect 0 "wizard.sh: Root's key: root on ws doesn't log in by the Mac's key; a run would stop" \
    "$keys" "$RUNNER_TEMP/root.keys"
stop="wizard.sh: Root's key: root on ws doesn't log in by the Mac's key, $HOME/.ssh/id_ed25519.pub,"
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
expect 1 "$stop" "$keys" "$RUNNER_TEMP/root.keys"
sudo install -m 600 "$RUNNER_TEMP/gh/keys" "$keys"
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
if [ "$status" != 0 ] || printf '%s\n' "$out" | grep -q "^wizard\.sh: Root's key: "; then
    error "wizard.sh exited $status, or said of Root's key, with the account's keys in root's file"
fi
sudo cmp -s "$keys" "$RUNNER_TEMP/gh/keys" || error "wizard.sh changed $keys"

cp "$hosts" "$RUNNER_TEMP/hosts"
sed "s|^ws .*|ws $other|" "$RUNNER_TEMP/hosts" >"$hosts"
cp "$hosts" "$RUNNER_TEMP/hosts.other"
cp ~/.ssh/config "$RUNNER_TEMP/ssh.config"
printf '%s\n' 'Host *' '    StrictHostKeyChecking no' >>~/.ssh/config
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
cp "$RUNNER_TEMP/ssh.config" ~/.ssh/config
expect 1 'offers another host key than known_hosts holds under ws: where ws was installed again,' \
    "$hosts" "$RUNNER_TEMP/hosts.other"
cp "$RUNNER_TEMP/hosts" "$hosts"
rm -f "$RUNNER_TEMP/other" "$RUNNER_TEMP/other.pub" "$RUNNER_TEMP"/root.keys "$RUNNER_TEMP"/hosts* \
    "$RUNNER_TEMP/ssh.config"
exit "$fail"
