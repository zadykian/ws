#!/bin/sh
# The stage gh on the Mac, through standin.sh's gh, on the Mac that 40-again.sh leaves. Without the
# login, a run with no terminal must stop at the stage, naming gh auth login, before ws mac apply's, and
# --check must say it would run it. With a login that lacks the scopes, or the signing one alone,
# as a 404 from gh api shows, each must do the same for gh auth refresh. None may change the login's
# keys. With the login back, a run must print nothing.

set -u

wsbin=$HOME/.local/bin/ws
state=$RUNNER_TEMP/gh
flow='-h github.com -s write:public_key,write:ssh_signing_key'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# expect STATUS WANT ARG...: ws mac wizard ARGs, with no terminal, must exit STATUS, with WANT in its
# lines of the stage. It must run no ws mac apply, and leave the login's keys as they were.
expect() {
    want_status=$1
    want=$2
    shift 2
    out=$("$wsbin" mac wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    printf '%s\n' "$out" | grep '^ws mac wizard: gh on the Mac: ' | grep -qF -e "$want" &&
        [ "$status" = "$want_status" ] || error "ws mac wizard $* exited $status, with no '$want'"
    ! printf '%s\n' "$out" | grep -qE '^ws mac wizard: (ws mac apply: would apply|wrote [^ ]*/\.ssh/config\.d/ws|ran launchctl |made the Mac.s WireGuard key)' || error "ws mac wizard $* ran ws mac apply"
    cmp -s "$state/keys" "$RUNNER_TEMP/gh.keys" || error "ws mac wizard $* changed the login's keys"
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
out=$("$wsbin" mac wizard </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "ws mac wizard with the login back exited $status: '$out'"
exit "$fail"
