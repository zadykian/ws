#!/bin/bash
# ws client macos apply writes Zed's keymap.json as roles/client/macos/mac/zed/keymap.json, whole. A
# file with another first line is left, with one warning. The file then goes, and the next run must
# write ws's back: a foreign keymap.json left behind would make every later run print the warning.

set -eo pipefail

file=~/.config/zed/keymap.json
fail=0
cmp -s "$file" roles/client/macos/mac/zed/keymap.json ||
    {
        echo '::error::keymap.json is not roles/client/macos/mac/zed/keymap.json'
        fail=1
    }
sed '/^[[:space:]]*\/\//d' roles/client/macos/mac/zed/keymap.json | python3 -m json.tool >/dev/null ||
    {
        echo '::error::roles/client/macos/mac/zed/keymap.json does not parse under its comments'
        fail=1
    }
printf '[]\n' >"$file"
out=$(ws client macos apply --only zed 2>&1) || {
    echo '::error::ws client macos apply --only zed failed'
    fail=1
}
[ "$(cat "$file")" = '[]' ] ||
    {
        echo '::error::ws client macos apply --only zed changed a keymap.json that is not ws'\''s'
        fail=1
    }
[ "$(printf '%s\n' "$out" | grep -c "keymap.json is not ws's")" = 1 ] ||
    {
        echo '::error::ws client macos apply --only zed did not warn once of the foreign keymap.json'
        fail=1
    }
rm "$file"
ws client macos apply --only zed 2>&1 || {
    echo '::error::ws client macos apply --only zed failed after the file went'
    fail=1
}
cmp -s "$file" roles/client/macos/mac/zed/keymap.json ||
    {
        echo '::error::ws client macos apply --only zed did not write ws'\''s keymap.json back'
        fail=1
    }
[ -z "$(ws client macos apply --only zed 2>&1)" ] ||
    {
        echo '::error::a second ws client macos apply --only zed printed something'
        fail=1
    }
[ -z "$(ws client macos apply --check --only zed 2>&1)" ] ||
    {
        echo '::error::ws client macos apply --check --only zed printed something'
        fail=1
    }
exit "$fail"
