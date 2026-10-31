#!/bin/sh
# A fresh --check, by sh and then by dash, with standin.sh's PATH, which lacks brew. The wizard must
# still find brew, and show the Include for ~/.ssh/config. It must show the Mac's key it would make
# and add to gh's login, and make none. The Host key stage must wait for setup.sh, Root's key for it, ws's run for root's key,
# and the Signing key, claude on ws, the Tunnel and the Cable for ws's run. It must write none of
# the paths that mac.yml's --check step lists, keep ~/.ssh/config as it was, and install no formula
# or cask. It must leave no lock or file of its own in $TMPDIR.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
dash=$(command -v dash)
fail=0
# mac.conf's answers that standin.sh leaves out, from the environment, each empty for no value.
export WS_TARGET=lan WS_LAN='' WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS='' WS_HOOK=''

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
    printf '%s\n' "$out" | grep -qx '+Include config.d/ws' ||
        error "wizard.sh --check by $shell showed no Include for ~/.ssh/config"
    printf '%s\n' "$out" | grep -q "^wizard\.sh: Mac key: would make $HOME/\.ssh/id_ed25519 " ||
        error "wizard.sh --check by $shell showed no key it would make"
    printf '%s\n' "$out" | grep -q '^wizard\.sh: gh on the Mac: would run gh ssh-key add ' ||
        error "wizard.sh --check by $shell showed no key it would add to gh's login"
    printf '%s\n' "$out" | grep -qx 'wizard\.sh: Host key: after setup\.sh' ||
        error "wizard.sh --check by $shell showed no Host key stage after setup.sh"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Root's key: after Host key" ||
        error "wizard.sh --check by $shell showed no Root's key stage after the Host key"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: ws's run: after Root's key" ||
        error "wizard.sh --check by $shell showed no ws's run after Root's key"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Signing key: after ws's run" ||
        error "wizard.sh --check by $shell showed no Signing key after ws's run"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: claude on ws: after ws's run" ||
        error "wizard.sh --check by $shell showed no claude on ws after ws's run"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Tunnel: after ws's run" ||
        error "wizard.sh --check by $shell showed no Tunnel stage after ws's run"
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: after ws's run" ||
        error "wizard.sh --check by $shell showed no Cable stage after ws's run"
    for made in ~/.ssh/config.d ~/.ssh/config.*~ /etc/ws-tunnel /usr/local/libexec/ws-tunnel \
        ~/.ssh/id_ed25519 /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist; do
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
