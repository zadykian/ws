#!/bin/sh
# The stage gh on the Mac, through standin.sh's gh, on the Mac that 40-again.sh leaves. Without the
# login, a run with no terminal must stop at the stage, naming gh auth login, before setup.sh's, and
# --check must say it would run it. With a login that lacks the scopes, or the signing one alone,
# as a 404 from gh api shows, each must do the same for gh auth refresh. None may change the login's
# keys. With the login back, a run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
state=$RUNNER_TEMP/gh
flow='-h github.com -s write:public_key,write:ssh_signing_key'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STATUS WANT ARG...: wizard.sh ARGs, with no terminal, must exit STATUS, with WANT in its
# lines of the stage. It must run no setup.sh, and leave the login's keys as they were.
expect() {
    want_status=$1
    want=$2
    shift 2
    out=$(sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep '^wizard\.sh: gh on the Mac: ' | grep -qF -e "$want" &&
        [ "$status" = "$want_status" ] || error "wizard.sh $* exited $status, with no '$want'"
    ! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error "wizard.sh $* ran setup.sh"
    cmp -s "$state/keys" "$RUNNER_TEMP/gh.keys" || error "wizard.sh $* changed the login's keys"
}

cp "$state/keys" "$RUNNER_TEMP/gh.keys"
rm "$state/login"
expect 1 "no terminal is there for gh auth login -p https -w $flow"
expect 0 "would run gh auth login -p https -w $flow" --check
touch "$state/login" "$state/noscope"
expect 1 "no terminal is there for gh auth refresh $flow"
expect 0 "would run gh auth refresh $flow" --check
rm "$state/noscope"
touch "$state/nosign"
expect 1 "no terminal is there for gh auth refresh $flow"
expect 0 "would run gh auth refresh $flow" --check
rm "$state/nosign" "$RUNNER_TEMP/gh.keys"
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh with the login back exited $status: '$out'"
exit "$fail"
