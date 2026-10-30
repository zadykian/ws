#!/bin/sh
# The Mac key stage on a fresh Mac, with a mac.conf of its own that names no key for ws, so the keys
# are the YubiKey's. --check must say it would write their .pubs, and with the YubiKey out, that a
# run would wait for it. A run must wait, and once one from the factory, 30000002, comes in, stop,
# naming ykman's commands, as no terminal is there for its prompts. --check must name the slots of
# both keys it would make there, and with the job's YubiKey in too, a run must stop, naming both.
# With the job's alone, a run with no terminal or login of gh's must write the .pubs, pass the stage
# once ws's agent lists the keys, and stop at gh's, before setup.sh's. A second must say nothing of
# the stage.
# None may make a file key, change that mac.conf, or leave a lock or file in $TMPDIR.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
standin=$HOME/repository/ws/.github/yubikey-standin.sh
conf=$RUNNER_TEMP/key.conf
keys=$HOME/.config/ws/keys
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# keys ARG...: wizard.sh ARGs, with that mac.conf and no terminal. It prints the output, and sets
# status, and mac to the Mac key stage's lines.
keys() {
    out=$(WS_CONF=$conf sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    mac=$(printf '%s\n' "$out" | grep '^wizard\.sh: Mac key: ')
}

# stopped WANT: the run before must have exited 1 at the Mac key stage, with WANT in its lines, and
# run no setup.sh.
stopped() {
    if [ "$status" != 1 ] || ! printf '%s\n' "$mac" | grep -qF -e "wizard.sh: Mac key: $1" ||
        printf '%s\n' "$out" | grep -q '^setup\.sh: '; then
        error "wizard.sh exited $status, with '$mac', not '$1'"
    fi
}

printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 WS_TARGET=lan WS_LAN= WS_GIT_NAME= \
    WS_TUNNEL_NETWORKS= WS_RESOLVERS= WS_HOOK= >"$conf"
cp "$conf" "$RUNNER_TEMP/key.conf.before"
keys --check
[ "$status" = 0 ] && printf '%s\n' "$out" | grep -qxF "+++ $keys/ws.pub" && [ ! -e "$keys" ] ||
    error "wizard.sh --check exited $status, showed no $keys/ws.pub it would write, or wrote it"
sh "$standin" unplug 30000001
keys --check
want="wizard.sh: Mac key: no YubiKey is in; a run would wait up to 5 minutes for one, then find or"
want="$want make the keys ws and git on it"
if [ "$status" != 0 ] || ! printf '%s\n' "$mac" | grep -qxF "$want" ||
    ! printf '%s\n' "$out" | grep -qxF 'wizard.sh: setup.sh: after Mac key'; then
    error "wizard.sh --check with no YubiKey exited $status, with '$mac'"
fi
: >"$RUNNER_TEMP/yubikey/log"
WS_CONF=$conf sh "$wizard" </dev/null >"$RUNNER_TEMP/key.out" 2>&1 &
pid=$!
sleep 10
sh "$standin" new 30000002
wait "$pid"
status=$?
out=$(cat "$RUNNER_TEMP/key.out")
printf '%s\n' "$out"
mac=$(printf '%s\n' "$out" | grep '^wizard\.sh: Mac key: ')
stopped "no YubiKey is in: plug in the one with ws's keys, or a new one; the wizard waits up to"
stopped "no terminal for ykman to make ws's key, as it asks the PIN, and the management key unless"
! grep -q ' generate ' "$RUNNER_TEMP/yubikey/log" || error 'wizard.sh ran ykman generate'
keys --check
make='ykman piv keys generate -a ED25519 --pin-policy NEVER --touch-policy NEVER'
for slot in "ws's key in slot 82" "git's key in slot 83"; do
    printf '%s\n' "$mac" | grep -qF "Mac key: would make $slot of YubiKey 30000002: $make" &&
        [ "$status" = 0 ] || error "wizard.sh --check exited $status, making no $slot: '$mac'"
done
sh "$standin" plug 30000001
keys
stopped 'several YubiKeys are in, 30000001 and 30000002: leave in the one'
sh "$standin" unplug 30000002
rm "$RUNNER_TEMP/gh/login"
for run in 1 2; do
    keys
    printf '%s\n' "$out" | grep -q '^wizard\.sh: gh on the Mac: gh lacks a login' &&
        [ "$status" = 1 ] || error "wizard.sh run $run exited $status, not at gh's stage"
    [ "$run" = 1 ] || [ -z "$mac" ] || error "wizard.sh run 2 said of the Mac key stage: '$mac'"
    ! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error "wizard.sh run $run ran setup.sh"
done
touch "$RUNNER_TEMP/gh/login"
[ -s "$keys/ws.pub" ] && [ -s "$keys/git.pub" ] || error "wizard.sh wrote no .pubs in $keys"
for left in ~/.ssh/id_ed25519 ~/.ssh/id_ed25519.pub "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
cmp -s "$conf" "$RUNNER_TEMP/key.conf.before" || error "wizard.sh changed $conf"
rm -f "$conf" "$RUNNER_TEMP"/key.*
exit "$fail"
