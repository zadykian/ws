#!/bin/sh
# Where sudo on the stand-in ws asks for a password, on the Mac that 33-login.sh leaves. A run with
# no terminal must stop at Sudo, naming the user, and --check must say that a run would ask. At a
# terminal of script's, a wrong password must be refused and the right one taken. The run must
# end well, with sudo's forms that carry a password in the stand-in's log. The password must show in
# no transcript and no log. sudo asks no password again after it.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
tty=$RUNNER_TEMP/tty
password='stand-in password'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# prompts N: waits up to a minute for the Nth prompt for the password in the transcript.
prompts() {
    for _ in $(seq 60); do
        [ "$(grep -o 'sudo password on ws' "$tty" 2>/dev/null | wc -l)" -ge "$1" ] && return 0
        sleep 1
    done
}

rm "$ws/sudo.open"
(umask 077 && printf '%s\n' "$password" >"$ws/sudo.password")
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qF "Sudo: $(id -un)'s sudo on ws asks for a password, and no terminal" &&
    [ "$status" = 1 ] ||
    error "ws client macos wizard exited $status, or did not stop at Sudo, with no terminal"
out=$("$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qF "Sudo: would ask for $(id -un)'s sudo password on ws" &&
    [ "$status" = 0 ] ||
    error "ws client macos wizard --check exited $status, or said not that it would ask"
: >"$ws/sudo.log"
{
    prompts 1
    printf '%s\n' 'wrong password'
    prompts 2
    printf '%s\n' "$password"
    sleep 5
} | script -q /dev/null "$wsbin" --plain client macos wizard >"$tty" 2>&1
status=$?
tr -d '\r' <"$tty"
tr -d '\r' <"$tty" | grep -qF 'Sudo: sudo on ws refused the password' &&
    [ "$status" = 0 ] ||
    error "ws client macos wizard at a terminal exited $status, or did not refuse the wrong password"
grep -qF "$password" "$tty" "$ws/log" "$ws/sudo.log" && error 'the password showed in a transcript or a log'
grep -q "^sudo -k -S -p '' sh -s -- status$" "$ws/sudo.log" || error 'no status was read through sudo -k -S -p'
rm -f "$tty" "$ws/sudo.password"
touch "$ws/sudo.open"
exit "$fail"
