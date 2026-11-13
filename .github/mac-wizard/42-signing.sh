#!/bin/sh
# The stage Signing key, through standin.sh's gh and ws, on the Mac that 41-remote.sh leaves. First
# ws's signing key is off the Mac's gh login. --check must then say it would run gh ssh-key add
# --type signing, titled with ws's hostname, and change nothing. A run must add the key to the
# signing keys alone, and list what stays by hand, which no longer holds ws's gh login. The next
# run must print nothing. Where the login's git config on ws names no signing key, a run must print
# nothing, with the key off the login.

set -u

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws client macos wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
state=$RUNNER_TEMP/gh
ws=$RUNNER_TEMP/ws
key=$(cut -d ' ' -f 1-2 "$ws/signing.pub")
login="- macOS's local network:"
add="gh ssh-key add --type signing --title $(hostname)"
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# quiet: a run with no terminal must print nothing.
quiet() {
    out=$("$wsbin" client macos wizard </dev/null 2>&1)
    status=$?
    [ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "ws client macos wizard $1 exited $status: '$out'"
}

: >"$state/signing"
out=$("$wsbin" client macos wizard --check </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qxF "ws client macos wizard: Signing key: would run $add for ws's signing key" &&
    [ "$status" = 0 ] && [ ! -s "$state/signing" ] || error "ws client macos wizard --check exited $status"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qxF "ws client macos wizard: Signing key: ran $add for ws's signing key" &&
    printf '%s\n' "$out" | grep -qx 'What stays by hand:' &&
    printf '%s\n' "$out" | grep -qF -e "$login" &&
    ! printf '%s\n' "$out" | grep -qF "ws's gh login" &&
    [ "$status" = 0 ] &&
    [ "$(cat "$state/signing")" = "$key" ] && [ "$(grep -cxF "$key" "$state/keys")" = 0 ] ||
    error "ws client macos wizard exited $status, adding other than the signing key to the signing keys"
quiet 'after the key was added'
: >"$state/signing"
sudo git config -f "$ws/root/home/$(id -un)/.gitconfig" --unset user.signingkey
quiet 'with no signing key on ws'
sudo git config -f "$ws/root/home/$(id -un)/.gitconfig" user.signingkey \
    "$ws/root/home/$(id -un)/.ssh/id_ed25519_signing.pub"
echo "$key" >"$state/signing"
exit "$fail"
