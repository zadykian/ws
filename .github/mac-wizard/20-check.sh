#!/bin/sh
# A fresh --check, by sh and then by dash, with standin.sh's PATH, which lacks brew. The wizard must
# still find brew, and show setup.sh's install of wireguard-tools, where Homebrew lacks it, and the
# Include for ~/.ssh/config. It must write none of the paths that mac.yml's --check step lists, keep
# ~/.ssh/config as it was, and install no formula or cask. It must leave no lock or file of its own
# in $TMPDIR.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
dash=$(command -v dash)
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# config prints ~/.ssh/config, and whether it is there.
config() {
    cat ~/.ssh/config 2>/dev/null
    echo "cat: $?"
}

path=$(cat "$RUNNER_TEMP/path")
{ brew list --formula && brew list --cask; } >"$RUNNER_TEMP/brew"
config >"$RUNNER_TEMP/config"
for shell in sh "$dash"; do
    out=$(PATH=$path "$shell" "$wizard" --check 2>&1)
    status=$?
    printf '%s\n' "$out"
    [ "$status" = 0 ] || error "wizard.sh --check by $shell exited $status"
    ! printf '%s\n' "$out" | grep -q ': after Tools$' || error "wizard.sh --check found no brew"
    grep -qx wireguard-tools "$RUNNER_TEMP/brew" ||
        printf '%s\n' "$out" | grep -q '^setup.sh: would run brew install wireguard-tools$' ||
        error "wizard.sh --check by $shell showed no brew install wireguard-tools"
    printf '%s\n' "$out" | grep -qx '+Include config.d/ws' ||
        error "wizard.sh --check by $shell showed no Include for ~/.ssh/config"
    for made in ~/.ssh/config.d ~/.ssh/config.*~ /etc/ws-tunnel /usr/local/libexec/ws-tunnel \
        /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist; do
        ! sudo test -e "$made" || error "wizard.sh --check by $shell made $made"
    done
    config | cmp -s - "$RUNNER_TEMP/config" ||
        error "wizard.sh --check by $shell changed ~/.ssh/config"
    { brew list --formula && brew list --cask; } | cmp -s - "$RUNNER_TEMP/brew" ||
        error "wizard.sh --check by $shell installed a formula or a cask"
    for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
        [ ! -e "$left" ] || error "wizard.sh --check by $shell left $left"
    done
done
exit "$fail"
