#!/bin/sh
# A cloud VM's login is root by default, which the stand-in ws takes by root's own authorized_keys,
# with the account's keys there. ws client macos apply on a cloud target that names no WS_USER must
# write ssh's User as root. --check after it must read ws's status as root, with no sudo behind it,
# and ask for no password. Else Setup is pending, and --check skips the stages that read ws. ws
# client macos apply on mac.conf then gives ssh the runner again, and the key file goes.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

WS_CONF=$RUNNER_TEMP/confs/cloud/mac.conf
export WS_CONF
conf_print ~/.config/ws/mac.conf | sed -e 's/^WS_TARGET=.*/WS_TARGET=cloud/' -e '/^WS_USER=/d' | conf_write "$WS_CONF"
sudo install -d -m 700 ~root/.ssh
sudo install -m 600 "$RUNNER_TEMP/gh/keys" ~root/.ssh/authorized_keys
"$wsbin" client macos apply </dev/null || error "ws client macos apply on a cloud target exited $?"
ssh -G ws | grep -qx 'user root' || error "ws client macos apply left ssh's User as another than root"
reads=$(grep -c '^root: sh -s -- status$' "$ws/log")
sudos=$(wc -l <"$ws/sudo.log")
out=$("$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "ws client macos wizard --check as root on a cloud VM exited $status"
[ "$(grep -c '^root: sh -s -- status$' "$ws/log")" -gt "$reads" ] ||
    error "ws client macos wizard read ws's status as no root login"
[ "$(wc -l <"$ws/sudo.log")" -eq "$sudos" ] || error "ws client macos wizard ran sudo on ws as root"
printf '%s\n' "$out" | grep -q '^ws client macos wizard: Sudo: ' && error "ws client macos wizard spoke of sudo as root"
sudo rm -f ~root/.ssh/authorized_keys
rm -rf "${WS_CONF%/*}"
unset WS_CONF
"$wsbin" client macos apply </dev/null || error "ws client macos apply back on mac.conf exited $?"
exit "$fail"
