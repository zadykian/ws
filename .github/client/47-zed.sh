#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote Zed's global_settings.json, macos/roles/zed/files's whole,
# and the seed settings.json where none was. The next leaves that one byte for byte, with no change
# in its plan. devtools' /etc/paths.d/ws puts the cask's zed on a login shell's PATH.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

dir=~/.config/zed
fail=0
head -n 1 "$dir/global_settings.json" | grep -qxF '// Written by github.com/zadykian/ws, whose next run replaces any change made here.' ||
    {
        echo '::error::global_settings.json has no ws header'
        fail=1
    }
cmp -s "$dir/global_settings.json" macos/roles/zed/files/global_settings.json ||
    {
        echo '::error::global_settings.json is not macos/roles/zed/files/global_settings.json'
        fail=1
    }
tail -n +2 "$dir/global_settings.json" | jq -e . >/dev/null ||
    {
        echo '::error::global_settings.json does not parse'
        fail=1
    }
head -n 1 "$dir/settings.json" | grep -qxF "// Zed's own settings, which its UI writes. ws wrote this once, where none was; ws's" ||
    {
        echo '::error::settings.json has no seed comment'
        fail=1
    }
grep -qxF '{ "ssh_connections": [{ "host": "ws" }] }' "$dir/settings.json" ||
    {
        echo '::error::settings.json has no seed for the host ws'
        fail=1
    }
grep -qxF /opt/homebrew/bin /etc/paths.d/ws ||
    {
        echo '::error::/etc/paths.d/ws does not name /opt/homebrew/bin'
        fail=1
    }
zsh -lc 'command -v zed' | grep -qxF /opt/homebrew/bin/zed ||
    {
        echo "::error::a login shell's zed is not the cask's"
        fail=1
    }
printf '{ "theme": "mine" }\n' >"$dir/settings.json"
client_bootstrap 2>&1 || {
    echo '::error::mise bootstrap failed'
    fail=1
}
[ "$(cat "$dir/settings.json")" = '{ "theme": "mine" }' ] ||
    {
        echo '::error::mise bootstrap changed a settings.json that was there'
        fail=1
    }
client_bootstrap plan --detailed-exitcode 2>&1 || {
    echo "::error::the plan with the user's settings.json exited $?, not 0"
    fail=1
}
exit "$fail"
