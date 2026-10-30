#!/bin/sh
# The Mac key stage on a fresh Mac, with a mac.conf of its own that names no key for ws, so the keys
# are the YubiKey's. --check must say it would write their .pubs, and with the YubiKey out, that a
# run would wait for it. With it in, a run with no terminal or login of gh's must write them, pass
# the stage once ws's agent lists the keys, and stop at gh's, before setup.sh's. A second must say
# nothing of the stage.
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
sh "$standin" plug 30000001
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
rm -f "$conf" "$RUNNER_TEMP/key.conf.before"
exit "$fail"
