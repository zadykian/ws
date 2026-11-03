#!/bin/sh
# The stage Hook on ws, on the Mac that 46-gh.sh leaves. WS_HOOK names a release, example/hook
# @v1.0.0, that the stand-in gh on the Mac downloads. Without the Mac's clone, --check must say Hook
# on ws is after Hook on the Mac, and make no clone. A run with no terminal must make the clone,
# send the build once, and run hook apply. ws must keep the build at bin/hook-v1.0.0 in the
# wizard's state, mode 700 and root's, make no clone and use no gh. The command must run once over
# plain ssh. Its stdin must be empty, and no terminal. The next run must
# print nothing. Where the command exits 1, --update must stop, and the next run, once it
# exits 0, run it again. With the command held and no stamp, --check must say it would send the
# build and run hook apply, and run the command with --check, once. With v1.1.0, the build must
# replace v1.0.0 on ws, and the stamp hold the value. A release whose sums list another SHA-256
# must stop the stage, naming them, with nothing sent, and so must a tag with no release.
# mac.conf, the clone and the stamp go back, and a run must print nothing.

set -u

wsbin=$HOME/.local/bin/ws
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
hook=$ws/root/root/repository/hook
bin=$ws/root/var/lib/ws-wizard/bin
mac=$HOME/repository/hook
stamp=$ws/root/var/lib/ws-wizard/hook-hook
before=$RUNNER_TEMP/hook.conf
set='s|^WS_HOOK=$|WS_HOOK=example/hook@v1.0.0|'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard [ARG]: a run of the wizard with no terminal. It prints the output, and sets out and status.
wizard() {
    out=$("$wsbin" mac wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# sessions prints how many sessions on ws were to run the command.
sessions() {
    grep -c ' /var/lib/ws-wizard/bin/hook-v[0-9.]* apply$' "$ws/log"
}

# sends prints how many builds the wizard sent to ws since this case began, by the ssh that writes
# one. An earlier case's send stays in the log, so the count starts from what the log held here.
sends() {
    echo $(($(grep -c '^root: umask 077 && cat >.*/hook-hook\.bin$' "$ws/log") - sent))
}

# tagged TAG: mac.conf names the release TAG.
tagged() {
    sed "s|^WS_HOOK=\$|WS_HOOK=example/hook@$1|" "$before" >"$conf"
}

# applied prints how many times the command ran, then what it logged of its last stdin.
applied() {
    echo "$(cat "$ws/apply.log" 2>/dev/null | wc -l | tr -d ' ') $(tail -n 1 "$ws/apply.log")"
}

logged=$(wc -l <"$ws/log")
sent=$(grep -c '^root: umask 077 && cat >.*/hook-hook\.bin$' "$ws/log")
empty=$(shasum -a 256 </dev/null | cut -d ' ' -f 1)
cp "$conf" "$before"
sed "$set" "$before" >"$conf"

wizard --check
[ "$status" = 0 ] &&
    printf '%s\n' "$out" | grep -qxF 'ws mac wizard: Hook on ws: after Hook on the Mac' &&
    [ ! -e "$mac" ] && [ ! -e "$hook" ] || error "ws mac wizard --check exited $status, or cloned"
ran=$(sessions)
wizard
[ "$status" = 0 ] && [ "$(applied)" = "1 $empty" ] && [ -f "$stamp" ] && [ -d "$mac/.git" ] &&
    [ "$(sessions)" = $((ran + 1)) ] && [ "$(sends)" = 1 ] ||
    error "ws mac wizard exited $status, or ran the command other than once with an empty stdin"
[ "$(sudo stat -f '%Lp %Su' "$bin/hook-v1.0.0")" = '700 root' ] && [ "$(sudo stat -f %Lp "$bin")" = 700 ] &&
    [ ! -e "$hook" ] && ! sudo test -e "$ws/root/var/lib/ws-wizard/hook-hook.bin" &&
    [ -f "$stamp" ] || error "the command is not root's alone at bin, or ws made a clone"
! tail -n "+$((logged + 1))" "$ws/log" | grep -q 'clone\|gh ' || error 'ws made a clone, or used gh'
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(applied)" = "1 $empty" ] ||
    error "the next run exited $status"
echo 1 >"$ws/apply.status"
wizard --update
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'Hook on ws: the release of example/hook@v1.0.0' &&
    [ "$(applied)" = "2 $empty" ] && [ ! -e "$stamp" ] ||
    error "ws mac wizard --update with the command exiting 1 exited $status, or kept the stamp"
rm "$ws/apply.status"
sudo rm -f "$stamp"
wizard --check
[ "$status" = 0 ] && [ "$(tail -n 1 "$ws/hook.args")" = 'apply --check' ] &&
    printf '%s\n' "$out" | grep -qxF "ws mac wizard: Hook on ws: would download example/hook's \
release v1.0.0 with the Mac's gh, check its SHA-256, send it to ws, then run hook apply there as \
root, its stdin empty" &&
    [ "$(wc -l <"$ws/check.log" | tr -d ' ')" = 1 ] && [ "$(applied)" = "2 $empty" ] ||
    error "ws mac wizard --check with the command held exited $status, or ran it other than once with --check"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "3 $empty" ] && [ -f "$stamp" ] ||
    error "the run after it exited $status"
sudo rm "$stamp"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "4 $empty" ] ||
    error "ws mac wizard with no stamp exited $status, with the command's log '$(applied)'"
[ "$(sends)" = 4 ] || error "ws mac wizard sent the build other than once for each run"

# A new tag: the build of v1.1.0 replaces v1.0.0 on ws, and the stamp names the tag.
tagged v1.1.0
wizard
[ "$status" = 0 ] && [ "$(applied)" = "5 $empty" ] && [ "$(sends)" = 5 ] &&
    sudo test -x "$bin/hook-v1.1.0" && ! sudo test -e "$bin/hook-v1.0.0" &&
    [ "$(cut -d ' ' -f 3 "$stamp")" = example/hook@v1.1.0 ] ||
    error "ws mac wizard with v1.1.0 exited $status, or ws holds other than its build"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "the next run with the tag exited $status"
# Sums that list another SHA-256 stop the stage with nothing sent, and so does a tag with no release.
touch "$RUNNER_TEMP/gh/badsum"
tagged v1.2.0
wizard
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'is not the one that hook.sha256 lists' &&
    ! sudo test -e "$bin/hook-v1.2.0" && [ "$(sends)" = 5 ] || error "ws mac wizard with bad sums exited $status, or sent a build"
rm "$RUNNER_TEMP/gh/badsum"
touch "$RUNNER_TEMP/gh/norelease"
wizard
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'gh cannot download v1.2.0 of example/hook' &&
    [ "$(sends)" = 5 ] || error "ws mac wizard with no release exited $status, or sent a build"
rm "$RUNNER_TEMP/gh/norelease"

cp "$before" "$conf"
sudo rm -rf "$bin" "$stamp"
rm -rf "$mac" "$RUNNER_TEMP/gh/github.com"
rm -f "$ws/apply.log" "$ws/check.log" "$ws/hook.args" "$before" "$RUNNER_TEMP/tty"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "ws mac wizard after it exited $status"
exit "$fail"
