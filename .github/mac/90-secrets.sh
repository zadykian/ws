#!/bin/bash
# ws mac secrets, on a throwaway keychain that is the default one, so that the runner's own is
# left alone. init must make the item ws-secrets and ~/.config/ws/secrets.age, mode 600 in a
# directory of 700, and say so. A second init must keep both, change neither and make no new key.
# The output must hold no key. set must take a value from stdin, and keys, which decrypts the file,
# must list it by kind and name alone: no output may hold the value, nor the file in plain text.
# unset must drop an entry. The keychain and the file go back.

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

value='a stand-in value, for the case'
out=$(printf '%s\n' "$value" | "$wsbin" mac secrets set env MY_TOKEN 2>&1) || error "set env exited $?"
out+=$(printf '%s\n' "$value" | "$wsbin" mac secrets set token github 2>&1) || error "set token exited $?"
keys=$("$wsbin" mac secrets keys 2>&1) || error "keys exited $?, so the file did not decrypt"
echo "$keys"
[ "$keys" = $'env MY_TOKEN\ntoken github' ] || error "keys listed other entries than env MY_TOKEN and token github"
grep -qF "$value" <<<"$out$keys" && error "a command printed the value"
grep -qF "$value" "$file" && error "the file holds the value in plain text"
"$wsbin" mac secrets unset env MY_TOKEN || error "unset exited $?"
[ "$("$wsbin" mac secrets keys 2>&1)" = 'token github' ] || error "unset did not drop env MY_TOKEN"
status=0
"$wsbin" mac secrets unset env MY_TOKEN 2>/dev/null || status=$?
[ "$status" = 1 ] || error "unset of an entry that is gone exited $status, not 1"
[ "$(stat -f %Lp "$file")" = 600 ] || error "a set left $file with a mode other than 600"

rm -f "$file"
# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
exit "$fail"
