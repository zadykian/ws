#!/bin/bash
# The Mac's checks of its containers, claude and gpg: trials 8 and 9. Run as the Mac's user, as
# `tools.sh 8 [DOWNLOAD]` or `tools.sh 9 [SHA256]`. It starts and pulls nothing. The page,
# macos/checks/README.md, says what to pass.
set -uo pipefail

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=macos/checks/lib.sh
. "$dir/lib.sh"

# downloaded succeeds where the file is there and holds data.
downloaded() { [ -s "$1" ] || {
    echo "$1 is missing or empty"
    return 1
}; }

check8() {
    check "8 the firewall blocks all incoming" blocks_all
    check "8 colima runs" colima status
    check "8 docker reaches colima's engine" docker info --format '{{.ServerVersion}}'
    check "8 docker run pulled hello-world" docker image inspect hello-world
    if [ $# -gt 0 ]; then
        check "8 the torrent's file is there" downloaded "$1"
    else
        waits "8 give the torrent's file to see it: tools.sh 8 PATH"
    fi
    check "8 Transmission is installed" test -d /Applications/Transmission.app
}

blocks_all() {
    "${SOCKETFILTERFW:-/usr/libexec/ApplicationFirewall/socketfilterfw}" --getblockall | grep -q 'blocking all'
}

# claude_path prints where claude resolves, through the links, as zsh finds it: ~/.local/bin first.
claude_path() { readlink -f "$(PATH=$HOME/.local/bin:$PATH command -v claude)"; }

# in_cask succeeds where claude resolves into a cask's directory, not a path the updater made.
in_cask() {
    case $(claude_path) in
    */Caskroom/*) ;;
    *)
        echo "claude resolves to $(claude_path), outside a Caskroom"
        return 1
        ;;
    esac
}

# sum_is SHA256 succeeds where the binary's sum is SHA256.
sum_is() {
    local got
    got=$(shasum -a 256 "$(claude_path)" | awk '{ print $1 }')
    [ "$got" = "$1" ] || {
        echo "the binary's sum is $got, not $1"
        return 1
    }
}

# pinentry_named succeeds where gpg-agent.conf names a pinentry-mac that runs.
pinentry_named() {
    local program
    program=$(awk '$1 == "pinentry-program" { print $2 }' "$HOME/.gnupg/gpg-agent.conf")
    case $program in
    */pinentry-mac) [ -x "$program" ] || {
        echo "$program is not executable"
        return 1
    } ;;
    *)
        echo "gpg-agent.conf names '$program', not pinentry-mac"
        return 1
        ;;
    esac
}

check9() {
    check "9 claude runs" claude --version
    check "9 claude is the cask's binary" in_cask
    info "claude: $(claude_path), $(shasum -a 256 "$(claude_path)" 2>&1 | awk '{ print $1 }')"
    if [ $# -gt 0 ]; then
        check "9 claude's binary is the one from before its update" sum_is "$1"
    else
        waits "9 run claude's update, then give the sum above: tools.sh 9 SHA256"
    fi
    check "9 gpg-agent names pinentry-mac" pinentry_named
    check "9 gpg's agent answers" gpg-connect-agent /bye
}

main() {
    local arg=${1:-}
    [ $# -gt 0 ] && shift
    case $arg in
    8) check8 "$@" ;;
    9) check9 "$@" ;;
    *)
        echo "usage: tools.sh 8 [DOWNLOAD] | tools.sh 9 [SHA256]" >&2
        exit 2
        ;;
    esac
    finish
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
