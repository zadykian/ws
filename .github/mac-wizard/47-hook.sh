#!/bin/sh
# The stage Hook on ws, on the Mac that 46-gh.sh leaves. WS_HOOK names a repository that the
# stand-in gh on the Mac clones, and WS_HOOK_SECRET an item that a throwaway keychain, the default
# one, lacks. Without the Mac's clone, --check must say Hook on ws is after Hook on the Mac, and a
# run with no terminal make the clone. Then --check must say it would add the item, send the clone
# and run its apply.sh, and do none. A run with no terminal must stop, naming security
# add-generic-password. --update at a terminal of script's must add it, its secret typed at
# security's prompts, fetch the repository's newer commit, and send the Mac's clone once, as a
# bundle of its origin/main. ws must clone it on branch main, with GitHub's URL as origin, keep no
# bundle and use no gh. apply.sh must run once over plain ssh. Its stdin must be the secret, and no
# terminal. It must print no secret, and the next run nothing. Where security can't read the item,
# --update must stop, naming its error. It must stop before the clone, the stamp kept, or, where
# the read for ssh fails, before apply.sh runs. Where apply.sh exits 1, --update must stop, and the
# next run, once it exits 0, run it again. With WS_HOOK_SECRET empty, its stdin must be empty.
# mac.conf, the clones, the stamp and the keychain go back, and a run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
hook=$ws/root/root/repository/hook
mac=$HOME/repository/hook
up=$RUNNER_TEMP/gh/github.com/example/hook
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

# sessions prints how many sessions on ws were to run apply.sh, alone or through wizard-hook.sh's
# guard.
sessions() {
    grep -c ' /root/repository/hook/apply\.sh$' "$ws/log"
}

# sends prints how many bundles the wizard sent to ws since this case began, by the ssh that writes
# one. An earlier case's send stays in the log, so the count starts from what the log held here.
sends() {
    echo $(($(grep -c '^root: umask 077 && cat >.*/hook-hook\.bundle$' "$ws/log") - sent))
}

# unread N: --update, with security failing each read of the secret from the Nth on, must stop,
# naming its error, and run no apply.sh.
unread() {
    echo 0 >"$RUNNER_TEMP/security.reads"
    echo "$1" >"$RUNNER_TEMP/security.from"
    wizard --update
    [ "$status" = 1 ] && printf '%s\n' "$out" | grep -qxF "wizard.sh: Hook on ws: security can't \
read the item ws-ci-hook from the Keychain, so apply.sh didn't run; unlock or allow it, then run \
the wizard again: security: SecKeychainItemCopyContent: User interaction is not allowed." &&
        [ "$(applied)" = "1 $sha" ] ||
        error "wizard.sh --update with read $1 of the item failing exited $status, or ran apply.sh"
}

# applied prints how many times apply.sh ran, then what it logged of its last stdin.
applied() {
    echo "$(cat "$ws/apply.log" 2>/dev/null | wc -l | tr -d ' ') $(tail -n 1 "$ws/apply.log")"
}

sent=$(grep -c '^root: umask 077 && cat >.*/hook-hook\.bundle$' "$ws/log")
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
[ "$status" = 0 ] &&
    printf '%s\n' "$out" | grep -qxF 'wizard.sh: Hook on ws: after Hook on the Mac' &&
    [ ! -e "$mac" ] && [ ! -e "$hook" ] || error "wizard.sh --check exited $status, or cloned"
wizard
add="security add-generic-password -a $USER -s ws-ci-hook -w"
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF "no terminal is there for $add: run it" &&
    [ -d "$mac/.git" ] && [ ! -e "$hook" ] && [ "$(sends)" = 0 ] ||
    error "wizard.sh without the item exited $status, made no clone on the Mac, or sent one"
wizard --check
[ "$status" = 0 ] && printf '%s\n' "$out" | grep -qF "Hook on ws: would run $add, which asks " &&
    printf '%s\n' "$out" | grep -qxF "wizard.sh: Hook on ws: would send the Mac's clone of \
example/hook to /root/repository/hook on ws, where missing, then run its apply.sh there, its stdin \
the Keychain item ws-ci-hook" && [ ! -e "$hook" ] && [ "$(sends)" = 0 ] ||
    error "wizard.sh --check with the Mac's clone exited $status, or sent it"
ran=$(sessions)
git -C "$up" -c user.name=ws -c user.email=ws@example.com commit -q --allow-empty -m newer
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
    [ "$(sessions)" = $((ran + 1)) ] && [ "$(sends)" = 1 ] ||
    error "wizard.sh --update at a terminal exited $status, or ran apply.sh other than once with it"
[ "$(sudo git -C "$hook" rev-parse HEAD)" = "$(git -C "$up" rev-parse HEAD)" ] &&
    [ "$(sudo git -C "$hook" symbolic-ref --short HEAD)" = main ] &&
    [ "$(sudo git -C "$hook" remote get-url origin)" = https://github.com/example/hook.git ] &&
    [ ! -e "$ws/root/var/lib/ws-wizard/hook-hook.bundle" ] ||
    error "ws's clone is not the Mac's origin/main on branch main, or the bundle stayed"
! grep -qF "$secret" "$RUNNER_TEMP/tty" || error 'wizard.sh printed the secret'
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(applied)" = "1 $sha" ] ||
    error "the next run exited $status"
# A stand-in security, first on the wizard's PATH, runs macOS's. But reads of a secret fail from
# the one that security.from counts on, as in a locked Keychain where no prompt can unlock it.
cat >/opt/homebrew/bin/security <<'EOF'
#!/bin/sh
case " $* " in
*' find-generic-password '*' -w '*)
    n=$(($(cat "$RUNNER_TEMP/security.reads") + 1))
    echo "$n" >"$RUNNER_TEMP/security.reads"
    if [ "$n" -ge "$(cat "$RUNNER_TEMP/security.from")" ]; then
        echo 'security: SecKeychainItemCopyContent: User interaction is not allowed.' >&2
        exit 36
    fi
    ;;
esac
exec /usr/bin/security "$@"
EOF
chmod 755 /opt/homebrew/bin/security
clones=$(grep -c 'sh -s -- clone example/hook$' "$ws/log")
ran=$(sessions)
unread 1
[ -f "$stamp" ] && [ "$(grep -c 'sh -s -- clone example/hook$' "$ws/log")" = "$clones" ] &&
    [ "$(sessions)" = "$ran" ] || error 'wizard.sh with an unreadable item went to ws'
unread 2
[ ! -e "$stamp" ] && [ "$(sessions)" = $((ran + 1)) ] ||
    error "wizard.sh with the read for ssh failing kept the stamp, or didn't reach ws's guard"
rm /opt/homebrew/bin/security "$RUNNER_TEMP/security.reads" "$RUNNER_TEMP/security.from"
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
[ "$(sends)" = 1 ] || error "wizard.sh sent the clone other than once"

cp "$before" "$conf"
sudo rm -rf "$hook" "$stamp"
rm -rf "$mac" "$RUNNER_TEMP/gh/github.com"
rm -f "$ws/apply.log" "$before" "$RUNNER_TEMP/tty"
# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
