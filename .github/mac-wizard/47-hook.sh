#!/bin/sh
# The stage Hook on ws, on the Mac that 46-gh.sh leaves. WS_HOOK names a release, example/hook
# @v1.0.0, that the stand-in gh on the Mac downloads, and WS_HOOK_SECRET an item that a throwaway
# keychain, the default one, lacks. Without the Mac's clone, --check must say Hook on ws is after
# Hook on the Mac, and a run with no terminal make the clone. Then --check must say it would
# download the release, send its build and run hook apply, and do none. A run with no terminal
# must stop, naming security add-generic-password. --update at a terminal of script's must add the
# item, its secret typed at security's prompts, and send the build once. ws must keep it at
# bin/hook-v1.0.0 in the wizard's state, mode 700 and root's, make no clone and use no gh. The
# command must run once over plain ssh. Its stdin must be the secret, and no terminal. It must
# print no secret, and the next run nothing. Where security can't read the item, --update must
# stop, naming its error. It must stop before the download, the stamp kept: ws reads the secret
# once, before it sends anything. Where the command exits 1, --update must stop, and the next run,
# once it exits 0, run it again. With the command held and no stamp, --check must run it with
# --check, once. With WS_HOOK_SECRET empty, its stdin must be empty. With v1.1.0, the build
# must replace v1.0.0 on ws, and the stamp hold the value. A release whose sums list another SHA-256
# must stop the stage, naming them, with nothing sent, and so must a tag with no release.
# mac.conf, the clone, the stamp and the keychain go back, and a run must print nothing.

set -u

wsbin=$HOME/.local/bin/ws
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
hook=$ws/root/root/repository/hook
bin=$ws/root/var/lib/ws-wizard/bin
mac=$HOME/repository/hook
stamp=$ws/root/var/lib/ws-wizard/hook-hook
keychain=$RUNNER_TEMP/hook.keychain-db
before=$RUNNER_TEMP/hook.conf
set='s|^WS_HOOK=$|WS_HOOK=example/hook@v1.0.0|'
secret='a stand-in secret, for the command'
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

# sessions prints how many sessions on ws were to run the command, alone or through the guard of
# ws mac wizard.
sessions() {
    grep -c ' /var/lib/ws-wizard/bin/hook-v[0-9.]* apply$' "$ws/log"
}

# sends prints how many builds the wizard sent to ws since this case began, by the ssh that writes
# one. An earlier case's send stays in the log, so the count starts from what the log held here.
sends() {
    echo $(($(grep -c '^root: umask 077 && cat >.*/hook-hook\.bin$' "$ws/log") - sent))
}

# tagged TAG: mac.conf names the release TAG, with no secret.
tagged() {
    {
        sed "s|^WS_HOOK=\$|WS_HOOK=example/hook@$1|" "$before"
        echo WS_HOOK_SECRET=
    } >"$conf"
}

# unread N: --update, with security failing each read of the secret from the Nth on, must stop,
# naming its error, and run no command.
unread() {
    echo 0 >"$RUNNER_TEMP/security.reads"
    echo "$1" >"$RUNNER_TEMP/security.from"
    wizard --update
    [ "$status" = 1 ] && printf '%s\n' "$out" | grep -qxF "ws mac wizard: Hook on ws: security can't \
read the item ws-ci-hook from the Keychain, so the command didn't run; unlock or allow it, then run \
the wizard again: security: SecKeychainItemCopyContent: User interaction is not allowed." &&
        [ "$(applied)" = "1 $sha" ] ||
        error "ws mac wizard --update with read $1 of the item failing exited $status, or ran the command"
}

# applied prints how many times the command ran, then what it logged of its last stdin.
applied() {
    echo "$(cat "$ws/apply.log" 2>/dev/null | wc -l | tr -d ' ') $(tail -n 1 "$ws/apply.log")"
}

logged=$(wc -l <"$ws/log")
sent=$(grep -c '^root: umask 077 && cat >.*/hook-hook\.bin$' "$ws/log")
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
    printf '%s\n' "$out" | grep -qxF 'ws mac wizard: Hook on ws: after Hook on the Mac' &&
    [ ! -e "$mac" ] && [ ! -e "$hook" ] || error "ws mac wizard --check exited $status, or cloned"
wizard
add="security add-generic-password -a $USER -s ws-ci-hook -w"
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF "no terminal is there for $add: run it" &&
    [ -d "$mac/.git" ] && [ ! -e "$hook" ] && [ "$(sends)" = 0 ] ||
    error "ws mac wizard without the item exited $status, made no clone on the Mac, or sent one"
wizard --check
[ "$status" = 0 ] && printf '%s\n' "$out" | grep -qF "Hook on ws: would run $add, which asks " &&
    printf '%s\n' "$out" | grep -qxF "ws mac wizard: Hook on ws: would download example/hook's \
release v1.0.0 with the Mac's gh, check its SHA-256, send it to ws, then run hook apply there as \
root, its stdin the Keychain item ws-ci-hook" && [ ! -e "$bin" ] && [ "$(sends)" = 0 ] ||
    error "ws mac wizard --check with the Mac's clone exited $status, or sent a build"
ran=$(sessions)
# script gives the wizard a terminal, which the command's ssh must not take. The secret goes in at
# each of security's two prompts, whose echo is off, or after 120 s; script's stdin ends 5 s after.
# shellcheck disable=SC2094 # the loop reads the prompts that script writes to tty, by design
for n in 1 2; do
    for _ in $(seq 120); do
        [ "$(sed -n '/, twice/,$p' "$RUNNER_TEMP/tty" | grep -c '[a-z]')" -le "$n" ] || break
        sleep 1
    done
    printf '%s\n' "$secret"
    sleep 5
done 2>/dev/null | script -q /dev/null "$wsbin" mac wizard --update >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
[ "$status" = 0 ] && [ "$(applied)" = "1 $sha" ] && [ -f "$stamp" ] &&
    [ "$(sessions)" = $((ran + 1)) ] && [ "$(sends)" = 1 ] ||
    error "ws mac wizard --update at a terminal exited $status, or ran apply.sh other than once with it"
[ "$(sudo git -C "$hook" rev-parse HEAD)" = "$(git -C "$up" rev-parse HEAD)" ] &&
    [ "$(sudo git -C "$hook" symbolic-ref --short HEAD)" = main ] &&
    [ "$(sudo git -C "$hook" remote get-url origin)" = https://github.com/example/hook.git ] &&
    [ ! -e "$ws/root/var/lib/ws-wizard/hook-hook.bundle" ] ||
    error "ws's clone is not the Mac's origin/main on branch main, or the bundle stayed"
! grep -qF "$secret" "$RUNNER_TEMP/tty" || error 'ws mac wizard printed the secret'
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
    [ "$(sessions)" = "$ran" ] || error 'ws mac wizard with an unreadable item went to ws'
rm /opt/homebrew/bin/security "$RUNNER_TEMP/security.reads" "$RUNNER_TEMP/security.from"
echo 1 >"$ws/apply.status"
wizard --update
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'Hook on ws: the clone of example/hook' &&
    [ "$(applied)" = "2 $sha" ] && [ ! -e "$stamp" ] ||
    error "ws mac wizard --update with apply.sh exiting 1 exited $status, or kept the stamp"
rm "$ws/apply.status"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "3 $sha" ] && [ -f "$stamp" ] ||
    error "the run after it exited $status"
{ sed "$set" "$before" && echo WS_HOOK_SECRET=; } >"$conf"
sudo rm "$stamp"
wizard
[ "$status" = 0 ] && [ "$(applied)" = "4 $empty" ] ||
    error "ws mac wizard without WS_HOOK_SECRET exited $status, with apply.sh's log '$(applied)'"
[ "$(sends)" = 1 ] || error "ws mac wizard sent the clone other than once"

# A tag: ws's clone moves from main to the stand-in gh's v1.0.0, which is not its HEAD.
tagged() {
    {
        sed "s|^WS_HOOK=\$|WS_HOOK=example/hook@$1|" "$before"
        echo WS_HOOK_SECRET=
    } >"$conf"
}
tagged v1.0.0
wizard
[ "$status" = 0 ] && [ "$(applied)" = "5 $empty" ] && [ "$(sends)" = 2 ] &&
    [ "$(sudo git -C "$hook" describe --tags --exact-match)" = v1.0.0 ] &&
    ! sudo git -C "$hook" symbolic-ref -q HEAD >/dev/null &&
    [ "$(cut -d ' ' -f 3 "$stamp")" = example/hook@v1.0.0 ] ||
    error "ws mac wizard with the tag exited $status, or ws's clone is not detached at it"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "the next run with the tag exited $status"
# A new tag with a change in the clone: the clone is refused, and stays at v1.0.0 with its change.
git -C "$up" -c user.name=ws -c user.email=ws@example.com commit -q --allow-empty -m tagged
git -C "$up" tag v1.1.0
echo changed | sudo tee -a "$hook/Brewfile" >/dev/null
tagged v1.1.0
wizard
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'Hook on ws: the clone of example/hook@v1.1.0' &&
    [ "$(sudo git -C "$hook" describe --tags --exact-match)" = v1.0.0 ] &&
    [ -n "$(sudo git -C "$hook" status --porcelain --untracked-files=no)" ] ||
    error "ws mac wizard with a changed clone exited $status, or changed it"
sudo git -C "$hook" checkout -q -- Brewfile
wizard
[ "$status" = 0 ] && [ "$(sudo git -C "$hook" describe --tags --exact-match)" = v1.1.0 ] ||
    error "ws mac wizard with the tag v1.1.0 exited $status, or ws's clone is not at it"
# GitHub moves v1.1.0, which the Mac fetched: ws, with no clone, stops the stage.
git -C "$up" -c user.name=ws -c user.email=ws@example.com commit -q --allow-empty -m moved
git -C "$up" tag -f v1.1.0 >/dev/null
sudo rm -rf "$hook" "$stamp"
wizard
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'git cannot fetch the tag v1.1.0 of example/hook' &&
    [ ! -e "$hook" ] || error "ws mac wizard with a moved tag exited $status, or made a clone"

cp "$before" "$conf"
sudo rm -rf "$hook" "$stamp"
rm -rf "$mac" "$RUNNER_TEMP/gh/github.com"
rm -f "$ws/apply.log" "$before" "$RUNNER_TEMP/tty"
# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "ws mac wizard after it exited $status"
exit "$fail"
