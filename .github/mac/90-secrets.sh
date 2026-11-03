#!/bin/bash
# ws mac secrets, on a throwaway keychain that is the default one, so that the runner's own is
# left alone. init must make the item ws-secrets and ~/.config/ws/secrets.age, mode 600 in a
# directory of 700, and say so. A second init must keep both, change neither and make no new key.
# The output must hold no key. The keychain and the file go back.

set -u

wsbin=$HOME/.local/bin/ws
file=$HOME/.config/ws/secrets.age
keychain=$RUNNER_TEMP/secrets.keychain-db
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

if [ -e "$file" ]; then
    echo "::error::$file is there before the case"
    exit 1
fi

list=$(security list-keychains -d user | tr -d '"')
default=$(security default-keychain -d user | tr -d '" ')
security create-keychain -p ws "$keychain"
security set-keychain-settings "$keychain"
security unlock-keychain -p ws "$keychain"
# shellcheck disable=SC2086 # the keychains' paths hold no blank
security list-keychains -d user -s "$keychain" $list
security default-keychain -d user -s "$keychain"

out=$("$wsbin" mac secrets init 2>&1)
status=$?
echo "$out"
if [ "$status" != 0 ] || ! grep -qF 'made the identity' <<<"$out"; then
    error "ws mac secrets init exited $status, or did not say that it made the identity"
fi
security find-generic-password -s ws-secrets >/dev/null 2>&1 || error "init made no item"
[ "$(stat -f %Lp "$file")" = 600 ] && [ "$(stat -f %Lp "$(dirname "$file")")" = 700 ] ||
    error "init made $file with a mode other than 600, or its directory with one other than 700"
grep -qF 'AGE-SECRET-KEY' <<<"$out" && error "init printed a key"
sum=$(shasum -a 256 "$file")

out=$("$wsbin" mac secrets init 2>&1)
status=$?
echo "$out"
if [ "$status" != 0 ] || ! grep -qF 'is there, and stays' <<<"$out"; then
    error "the second init exited $status, or did not say that the item stays"
fi
[ "$(shasum -a 256 "$file")" = "$sum" ] || error "the second init changed $file"

rm -f "$file"
# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
exit "$fail"
