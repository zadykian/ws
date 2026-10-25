#!/bin/sh
# The stage Hook on ws, on the Mac that 46-gh.sh leaves. WS_HOOK names a repository that the
# stand-in ws's gh clones, and WS_HOOK_SECRET an item that a throwaway keychain, the default one,
# lacks. --check must say it would add the item, clone the repository and run its apply.sh, and do
# none. A run with no terminal must stop, naming security add-generic-password. --update at a
# terminal of script's must add it, its secret typed at security's prompts, clone the repository,
# and run apply.sh once over plain ssh. Its stdin must be the secret, and no terminal. It must
# print no secret, and the next run nothing. Where apply.sh exits 1, --update must stop, and the
# next run, once it exits 0, run it again. With WS_HOOK_SECRET empty, its stdin must be empty.
# mac.conf, the clone, the stamp and the keychain go back, and a run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
hook=$ws/root/root/repository/hook
stamp=$ws/root/var/lib/ws-wizard/hook-hook
keychain=$RUNNER_TEMP/hook.keychain-db
before=$RUNNER_TEMP/hook.conf
set='s|^WS_HOOK=$|WS_HOOK=example/hook|'
secret='a stand-in secret, for apply.sh'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard [ARG]: a run of the wizard with no terminal. It prints the output, and sets out and status.
wizard() {
    out=$(sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# applied prints how many times apply.sh ran, then what it logged of its last stdin.
applied() {
    echo "$(cat "$ws/apply.log" 2>/dev/null | wc -l | tr -d ' ') $(tail -n 1 "$ws/apply.log")"
}

sha=$(printf '%s\n' "$secret" | shasum -a 256 | cut -d ' ' -f 1)
empty=$(shasum -a 256 </dev/null | cut -d ' ' -f 1)
cp "$conf" "$before"
{ sed "$set" "$before" && echo WS_HOOK_SECRET=ws-ci-hook; } >"$conf"
list=$(security list-keychains -d user | tr -d '"')
default=$(security default-keychain -d user | tr -d '" ')
security create-keychain -p ws "$keychain"
security set-keychain-settings "$keychain"
security unlock-keychain -p ws "$keychain"
# shellcheck disable=SC2086 # the keychains' paths hold no blank
security list-keychains -d user -s "$keychain" $list
security default-keychain -d user -s "$keychain"

wizard --check
add="security add-generic-password -a $USER -s ws-ci-hook -w"
[ "$status" = 0 ] && printf '%s\n' "$out" | grep -qF "Hook on ws: would run $add, which asks " &&
    printf '%s\n' "$out" | grep -qxF "wizard.sh: Hook on ws: would clone example/hook into \
/root/repository/hook on ws, where missing, then run its apply.sh there, its stdin the Keychain \
item ws-ci-hook" && [ ! -e "$hook" ] || error "wizard.sh --check exited $status, or cloned"
wizard
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF "no terminal is there for $add: run it" &&
    [ ! -e "$hook" ] || error "wizard.sh without the item exited $status, or cloned"
ran=$(grep -cx 'root: /root/repository/hook/apply.sh' "$ws/log")
# script gives the wizard a terminal, which apply.sh's ssh must not take. The secret goes in at
# each of security's two prompts, whose echo is off, or after 120 s; script's stdin ends 5 s after.
# shellcheck disable=SC2094 # the loop reads the prompts that script writes to tty, by design
for n in 1 2; do
    for _ in $(seq 120); do
        [ "$(sed -n '/, twice/,$p' "$RUNNER_TEMP/tty" | grep -c '[a-z]')" -le "$n" ] || break
        sleep 1
    done
    printf '%s\n' "$secret"
    sleep 5
done 2>/dev/null | script -q /dev/null sh "$wizard" --update >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
[ "$status" = 0 ] && [ "$(applied)" = "1 $sha" ] && [ -f "$stamp" ] &&
    [ "$(grep -cx 'root: /root/repository/hook/apply.sh' "$ws/log")" = $((ran + 1)) ] &&
    [ "$(cat "$ws/clone.log")" = example/hook ] ||
    error "wizard.sh --update at a terminal exited $status, or ran apply.sh other than once with it"
! grep -qF "$secret" "$RUNNER_TEMP/tty" || error 'wizard.sh printed the secret'
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(applied)" = "1 $sha" ] ||
    error "the next run exited $status"
echo 1 >"$ws/apply.status"
wizard --update
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'Hook on ws: the clone of example/hook' &&
    [ "$(applied)" = "2 $sha" ] && [ ! -e "$stamp" ] ||
    error "wizard.sh --update with apply.sh exiting 1 exited $status, or kept the stamp"
rm "$ws/apply.status"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "3 $sha" ] && [ -f "$stamp" ] ||
    error "the run after it exited $status"
{ sed "$set" "$before" && echo WS_HOOK_SECRET=; } >"$conf"
sudo rm "$stamp"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "4 $empty" ] ||
    error "wizard.sh without WS_HOOK_SECRET exited $status, with apply.sh's log '$(applied)'"

cp "$before" "$conf"
sudo rm -rf "$hook" "$stamp"
rm -f "$ws/clone.log" "$ws/apply.log" "$before" "$RUNNER_TEMP/tty"
# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
