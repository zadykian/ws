#!/bin/bash
# ws mac apply writes Zed's global_settings.json, which holds the golden file under ws's header, and
# settings.json, the seed, where it is missing. The job's ws renders the golden file, so the case
# checks the render too. A settings.json that is there is left byte for byte.

set -eo pipefail

dir=~/.config/zed
fail=0
head -n 1 "$dir/global_settings.json" | grep -qxF '// Written by ws mac apply: its next run replaces changes made here.' ||
    {
        echo '::error::global_settings.json has no ws header'
        fail=1
    }
tail -n +2 "$dir/global_settings.json" | cmp -s - internal/editor/zed/testdata/global_settings.json ||
    {
        echo '::error::global_settings.json is not the golden file under the header'
        fail=1
    }
tail -n +2 "$dir/global_settings.json" | jq -e . >/dev/null ||
    {
        echo '::error::global_settings.json does not parse'
        fail=1
    }
head -n 1 "$dir/settings.json" | grep -qxF "// Zed's own settings, which its UI writes. ws wrote this once, where it was missing; ws's" ||
    {
        echo '::error::settings.json has no seed comment'
        fail=1
    }
grep -qxF '{ "ssh_connections": [{ "host": "ws" }] }' "$dir/settings.json" ||
    {
        echo '::error::settings.json has no seed for the host ws'
        fail=1
    }
printf '{ "theme": "mine" }\n' >"$dir/settings.json"
ws mac apply zed 2>&1 || {
    echo '::error::ws mac apply zed failed'
    fail=1
}
[ "$(cat "$dir/settings.json")" = '{ "theme": "mine" }' ] ||
    {
        echo '::error::ws mac apply zed changed a settings.json that was there'
        fail=1
    }
exit "$fail"
