#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote Zed's keymap.json as roles/client/macos/zed/files/keymap.json,
# whole. A file without ws's mark at its top is left, and its unit's read warns. mise runs that read
# for its plan and shows none of its warnings, so ws unit read shows it here. The file then goes,
# and the next bootstrap must write ws's back: a foreign keymap.json left behind would make every
# later read warn. The plan after it must then exit 0, as a run that changes nothing.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

file=~/.config/zed/keymap.json
fail=0
cmp -s "$file" roles/client/macos/zed/files/keymap.json ||
    {
        echo '::error::keymap.json is not roles/client/macos/zed/files/keymap.json'
        fail=1
    }
sed '/^[[:space:]]*\/\//d' roles/client/macos/zed/files/keymap.json | python3 -m json.tool >/dev/null ||
    {
        echo '::error::roles/client/macos/zed/files/keymap.json does not parse under its comments'
        fail=1
    }
printf '[]\n' >"$file"
client_bootstrap 2>&1 || {
    echo '::error::mise bootstrap failed'
    fail=1
}
[ "$(cat "$file")" = '[]' ] ||
    {
        echo '::error::mise bootstrap changed a keymap.json that is not ws'\''s'
        fail=1
    }
out=$(cd /opt/ws && ws unit read file/zed-keymap 2>&1) || true
printf '%s\n' "$out" | grep -qF "keymap.json lacks the line" ||
    {
        printf '%s\n' "$out"
        echo '::error::ws unit read did not warn of the foreign keymap.json'
        fail=1
    }
rm "$file"
client_bootstrap 2>&1 || {
    echo '::error::mise bootstrap failed after the file went'
    fail=1
}
cmp -s "$file" roles/client/macos/zed/files/keymap.json ||
    {
        echo '::error::mise bootstrap did not write ws'\''s keymap.json back'
        fail=1
    }
client_bootstrap plan --detailed-exitcode 2>&1 || {
    echo "::error::the plan after ws's keymap.json came back exited $?, not 0"
    fail=1
}
exit "$fail"
