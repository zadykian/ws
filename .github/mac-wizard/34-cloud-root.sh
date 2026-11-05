#!/bin/sh
# A cloud VM's login is root by default, which the stand-in ws takes by root's own authorized_keys,
# with the account's keys there. ws mac apply on a cloud target that names no WS_USER must write
# ssh's User as root, and --check after it must read ws's status as root, with no sudo behind it,
# and ask for no password. Else Setup is pending, and --check skips the stages that read ws. ws mac
# apply on mac.conf then gives ssh the runner again, and the key file goes.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

WS_CONF=$RUNNER_TEMP/cloud.conf
export WS_CONF
sed -e 's/^WS_TARGET=.*/WS_TARGET=cloud/' -e '/^WS_USER=/d' ~/.config/ws/mac.conf >"$WS_CONF"
sudo install -d -m 700 ~root/.ssh
sudo install -m 600 "$RUNNER_TEMP/gh/keys" ~root/.ssh/authorized_keys
"$wsbin" mac apply </dev/null || error "ws mac apply on a cloud target exited $?"
ssh -G ws | grep -qx 'user root' || error "ws mac apply left ssh's User as another than root"
reads=$(grep -c '^root: sh -s -- status$' "$ws/log")
sudos=$(wc -l <"$ws/sudo.log")
out=$("$wsbin" mac wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] || error "ws mac wizard --check as root on a cloud VM exited $status"
[ "$(grep -c '^root: sh -s -- status$' "$ws/log")" -gt "$reads" ] ||
    error "ws mac wizard read ws's status as no root login"
[ "$(wc -l <"$ws/sudo.log")" -eq "$sudos" ] || error "ws mac wizard ran sudo on ws as root"
printf '%s\n' "$out" | grep -q '^ws mac wizard: Sudo: ' && error "ws mac wizard spoke of sudo as root"
sudo rm -f ~root/.ssh/authorized_keys
rm -f "$WS_CONF"
unset WS_CONF
"$wsbin" mac apply </dev/null || error "ws mac apply back on mac.conf exited $?"
exit "$fail"
