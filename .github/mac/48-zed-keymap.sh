#!/bin/bash
# ws mac apply writes Zed's keymap.json as the golden file. The job's ws renders it, so the case
# checks the render too. A file with another first line is left, with one warning. The
# file then goes, and the next run must write ws's back: a foreign keymap.json left behind would
# make every later run print the warning.

set -eo pipefail

file=~/.config/zed/keymap.json
fail=0
cmp -s "$file" internal/editor/zed/testdata/keymap.json ||
    {
        echo '::error::keymap.json is not the golden file'
        fail=1
    }
sed '/^[[:space:]]*\/\//d' internal/editor/zed/testdata/keymap.json | python3 -m json.tool >/dev/null ||
    {
        echo '::error::the golden keymap does not parse under its comments'
        fail=1
    }
printf '[]\n' >"$file"
out=$(ws mac apply zed 2>&1) || {
    echo '::error::ws mac apply zed failed'
    fail=1
}
[ "$(cat "$file")" = '[]' ] ||
    {
        echo '::error::ws mac apply zed changed a keymap.json that is not ws'\''s'
        fail=1
    }
[ "$(printf '%s\n' "$out" | grep -c "keymap.json is not ws's")" = 1 ] ||
    {
        echo '::error::ws mac apply zed did not warn once of the foreign keymap.json'
        fail=1
    }
rm "$file"
ws mac apply zed 2>&1 || {
    echo '::error::ws mac apply zed failed after the file went'
    fail=1
}
cmp -s "$file" internal/editor/zed/testdata/keymap.json ||
    {
        echo '::error::ws mac apply zed did not write ws'\''s keymap.json back'
        fail=1
    }
[ -z "$(ws mac apply zed 2>&1)" ] ||
    {
        echo '::error::a second ws mac apply zed printed something'
        fail=1
    }
[ -z "$(ws mac apply --check zed 2>&1)" ] ||
    {
        echo '::error::ws mac apply --check zed printed something'
        fail=1
    }
exit "$fail"
