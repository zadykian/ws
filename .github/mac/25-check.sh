#!/bin/bash
# --check on a fresh Mac changes nothing.
# --check must write no file, keep no copy of ~/.ssh/config and install no formula or cask.
# The brew installs it would run go to stderr, with ws mac apply's other lines.

set -eo pipefail

{ brew list --formula && brew list --cask; } >"$RUNNER_TEMP/brew"
ws mac apply --check 2>&1 | tee "$RUNNER_TEMP/check.out" ||
    {
        echo '::error::ws mac apply --check failed'
        exit 1
    }
fail=0
for path in ~/.ssh/config.d ~/.ssh/config.*~ /etc/ws-tunnel \
    /usr/local/libexec/ws-tunnel "/Library/LaunchDaemons/$LABEL.plist"; do
    if sudo test -e "$path"; then
        echo "::error::ws mac apply --check made $path"
        fail=1
    fi
done
cmp -s ~/.ssh/config "$RUNNER_TEMP/ssh-config-before" ||
    {
        echo '::error::ws mac apply --check changed ~/.ssh/config'
        fail=1
    }
{ brew list --formula && brew list --cask; } | cmp -s - "$RUNNER_TEMP/brew" ||
    {
        echo '::error::ws mac apply --check installed a formula or a cask'
        fail=1
    }
grep -q ': would run brew install --cask jetbrains-gateway$' "$RUNNER_TEMP/check.out" ||
    {
        echo "::error::ws mac apply --check showed no brew install of Gateway's cask"
        fail=1
    }
grep -q ': would run brew install --cask zed$' "$RUNNER_TEMP/check.out" ||
    {
        echo "::error::ws mac apply --check showed no brew install of Zed's cask"
        fail=1
    }
grep -q '^+Include config.d/ws$' "$RUNNER_TEMP/check.out" ||
    {
        echo "::error::ws mac apply --check showed no Include for ~/.ssh/config"
        fail=1
    }
exit "$fail"
