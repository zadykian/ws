#!/bin/bash
# mise bootstrap (28-bootstrap.sh) poured macos/roles/docker's formulae and seeded
# ~/.docker/config.json, which names the plugins' directory and the Keychain. The runner starts no
# machine: colima start stays by hand, so the cases ask the command line alone.

set -eo pipefail

fail=0
for tool in docker colima docker-credential-osxkeychain; do
    env -i HOME="$HOME" zsh -lc "command -v $tool" | grep -qx "/opt/homebrew/bin/$tool" ||
        {
            echo "::error::a login shell's $tool is not Homebrew's in /opt/homebrew/bin"
            fail=1
        }
done
for plugin in compose buildx; do
    env -i HOME="$HOME" zsh -lc "docker $plugin version" >/dev/null ||
        {
            echo "::error::docker $plugin version failed in a login shell: the plugin is not found"
            fail=1
        }
done
jq -e '.cliPluginsExtraDirs == ["/opt/homebrew/lib/docker/cli-plugins"] and .credsStore == "osxkeychain"' \
    ~/.docker/config.json >/dev/null ||
    {
        echo '::error::~/.docker/config.json does not name the plugins and osxkeychain'
        fail=1
    }
[ "$(stat -f '%Lp' ~/.docker/config.json)" = 600 ] ||
    {
        echo '::error::~/.docker/config.json is not at 0600'
        fail=1
    }
exit "$fail"
