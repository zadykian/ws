#!/bin/sh
# The stage Signing key, through standin.sh's gh and ws, on the Mac that 41-remote.sh leaves. With
# ws's signing key off the Mac's gh login, --check must say it would run gh ssh-key add --type
# signing, titled with ws's hostname, and change nothing. A run must add the key to the signing keys
# alone, and list what stays by hand, with ws's gh login, as mac.conf names no WS_HOOK. The next run
# must print nothing. Where root's git config on ws names no signing key, a run must print nothing,
# with the key off the login.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
state=$RUNNER_TEMP/gh
ws=$RUNNER_TEMP/ws
key=$(cut -d ' ' -f 1-2 "$ws/signing.pub")
login="- ws's gh login: gh auth login -h github.com --with-token"
add="gh ssh-key add --type signing --title $(hostname)"
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# quiet: a run with no terminal must print nothing.
quiet() {
    out=$(sh "$wizard" </dev/null 2>&1)
    status=$?
    [ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh $1 exited $status: '$out'"
}

: >"$state/signing"
out=$(sh "$wizard" --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qxF "wizard.sh: Signing key: would run $add for ws's signing key" &&
    [ "$status" = 0 ] && [ ! -s "$state/signing" ] || error "wizard.sh --check exited $status"
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qxF "wizard.sh: Signing key: ran $add for ws's signing key" &&
    printf '%s\n' "$out" | grep -qx 'What stays by hand:' &&
    printf '%s\n' "$out" | grep -qF -e "$login" &&
    [ "$status" = 0 ] &&
    [ "$(cat "$state/signing")" = "$key" ] && [ "$(grep -cxF "$key" "$state/keys")" = 0 ] ||
    error "wizard.sh exited $status, adding other than the signing key to the signing keys"
quiet 'after the key was added'
: >"$state/signing"
sudo git config -f "$ws/root/root/.gitconfig" --unset user.signingkey
quiet 'with no signing key on ws'
sudo git config -f "$ws/root/root/.gitconfig" user.signingkey \
    "$ws/root/root/.ssh/id_ed25519_signing.pub"
echo "$key" >"$state/signing"
exit "$fail"
