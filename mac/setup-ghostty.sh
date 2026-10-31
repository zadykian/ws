# shellcheck shell=sh
# The part of mac/setup.sh that cld's part reads of Ghostty's: whether its app is there. The step
# ghostty of ws setup installs Ghostty and its font and writes ws.ghostty. setup.sh sources this and
# calls ghostty_mac before cld_mac.

# 1 where Ghostty's app is there, or where --check would install it, as ghostty_mac sets it.
ghostty_ok=0

# ghostty_there succeeds where Ghostty's app is there: Ghostty.app in /Applications, where its cask
# puts it, or in ~/Applications.
ghostty_there() {
    [ -d /Applications/Ghostty.app ] || [ -d "$HOME/Applications/Ghostty.app" ]
}

# ghostty_mac runs the step ghostty, and sets ghostty_ok, which cld_would_tabs reads: the app is
# there, or --check would install it, as Homebrew has not the cask.
# shellcheck disable=SC2034 # setup-cld.sh's cld_would_tabs reads ghostty_ok
ghostty_mac() {
    step ghostty
    ghostty_ok=0
    if ghostty_there; then
        ghostty_ok=1
    elif ws_checking && ! brew list --cask ghostty >/dev/null 2>&1; then
        ghostty_ok=1
    fi
}
