#!/bin/sh
# gh's and claude's logins on the stand-in ws, on the Mac that 41-remote.sh leaves. standin.sh
# logged both in, so no run of the wizard may have opened a session on ws for a login. Without gh's,
# a run with no terminal must stop at gh on ws, naming gh auth login, and --check must say it would
# run it. With a login that lacks the write:ssh_signing_key scope, each must name gh auth refresh.
# With a token that GitHub refuses, a run must name gh auth login. Where the list of the account's
# keys fails otherwise, each must stop, naming gh's error, if any. With ws's signing key off the
# account, a run with no terminal must run bootstrap.sh --tags devtools, which adds it, then list
# what stays by hand. Without claude's login, a run must stop at claude on ws, naming claude auth
# login, and --check must say it would run it. The next run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
ws=$RUNNER_TEMP/ws
login='gh auth login -h github.com -p https -w -s write:ssh_signing_key'
refresh='gh auth refresh -h github.com -s write:ssh_signing_key'
devtools='bootstrap.sh --tags devtools'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STAGE STATUS WANT ARG...: wizard.sh ARGs, with no terminal, must exit STATUS, with WANT in
# its lines of STAGE.
expect() {
    stage=$1
    want_status=$2
    want=$3
    shift 3
    out=$(sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep "^wizard\.sh: $stage: " | grep -qF -e "$want" &&
        [ "$status" = "$want_status" ] || error "wizard.sh $* exited $status, with no '$want'"
}

# sessions: no session on ws may have run a login.
sessions() {
    ! grep -q -e '^root: gh auth ' -e '^root: exec bash -lc "claude auth login"$' "$ws/log" ||
        error "a run opened a session on ws for a login"
}

sessions
rm "$ws/gh"
expect 'gh on ws' 1 "no terminal is there for $login"
expect 'gh on ws' 0 "would open github.com/login/device on the Mac, run $login on ws," --check
touch "$ws/gh" "$ws/gh.nosign"
expect 'gh on ws' 1 "no terminal is there for $refresh"
expect 'gh on ws' 0 "run $refresh on ws, then run $devtools on ws" --check
rm "$ws/gh.nosign"
echo 'gh: Bad credentials (HTTP 401)' >"$ws/gh.fail"
expect 'gh on ws' 1 "no terminal is there for $login"
echo 'gh: Server Error (HTTP 502)' >"$ws/gh.fail"
down='gh api user/ssh_signing_keys failed as root on ws: gh: Server Error (HTTP 502); run the'
expect 'gh on ws' 1 "$down"
expect 'gh on ws' 1 "$down" --check
: >"$ws/gh.fail"
expect 'gh on ws' 1 'failed as root on ws and printed nothing; run the wizard again'
rm "$ws/gh.fail"
: >"$ws/gh.keys"
# Where root's git config names no signing key, as shell_git_signing off leaves it, gh on ws is
# done, and a run prints nothing.
sudo git config -f "$ws/root/root/.gitconfig" --unset user.signingkey
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh with no signing key exited $status: '$out'"
sudo git config -f "$ws/root/root/.gitconfig" user.signingkey \
    "$ws/root/root/.ssh/id_ed25519_signing.pub"
expect 'gh on ws' 0 "would run $devtools on ws, which adds ws's signing key to GitHub" --check
expect 'gh on ws' 0 "$devtools runs on ws in a unit of its own"
printf '%s\n' "$out" | grep -qx 'What stays by hand:' &&
    grep -qxF "$(cut -d ' ' -f 1-2 "$ws/signing.pub")" "$ws/gh.keys" &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags devtools' ] ||
    error "wizard.sh ran no $devtools that added ws's signing key, or listed nothing by hand"
rm "$ws/claude"
expect 'claude on ws' 1 'no terminal is there for claude auth login'
expect 'claude on ws' 0 'would run claude auth login as root on ws' --check
touch "$ws/claude"
sessions
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status: '$out'"
exit "$fail"
