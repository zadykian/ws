#!/bin/sh
# The wizard's files script on ws, for another repository's files there. The Mac's wizard sends it
# to root there over ssh, as it sends wizard-remote.sh:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote-files.sh
#
# - clone OWNER/REPO clones that repository on GitHub into /root/repository/REPO, by root's gh
#   login, where nothing is there. It never pulls a clone, as the repository's apply.sh updates its
#   own. It stops where the path holds no clone, another repository's, or a clone with no apply.sh
#   that runs.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The repositories' clones, a line NAME=PATH, which CI's stand-in for ws roots in a directory of its
# own.
repos=/root/repository

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote-files.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# repo OWNER/REPO sets repo to REPO, where OWNER/REPO names a repository on GitHub, as mac.conf's
# WS_HOOK does. Each side of the / is letters, digits and ._-, and neither . or .. alone, and REPO
# isn't ws, whose clone is ws's own. It stops otherwise.
repo() {
    case $1 in
    *[!A-Za-z0-9._/-]* | */*/* | /* | */ | ./* | ../* | */. | */.. | */ws) ;;
    */*)
        repo=${1#*/}
        return 0
        ;;
    esac
    die "'$1' is not a repository on GitHub, as OWNER/REPO, with a REPO other than ws"
}

# clone OWNER/REPO clones the repository in repos, as the header says. A clone there must be
# OWNER/REPO's: its origin, by https or ssh as gh sets it, names it on GitHub, letter case aside.
clone() {
    repo "$1"
    if [ ! -e "$repos/$repo" ] && [ ! -L "$repos/$repo" ]; then
        command -v gh >/dev/null || die "ws lacks gh, which ws's run installs"
        mkdir -p "$repos"
        gh repo clone "$1" "$repos/$repo" -- -q || die "gh repo clone $1 failed, as above"
        say "cloned $1 into $repos/$repo"
    fi
    [ -e "$repos/$repo/.git" ] || die "$repos/$repo holds no clone: move it aside"
    origin=$(git -C "$repos/$repo" remote get-url origin 2>/dev/null) || origin=
    case $(printf '%s\n' "${origin%.git}" | tr '[:upper:]' '[:lower:]') in
    *[/@]github.com[/:]"$(printf '%s\n' "${1%.git}" | tr '[:upper:]' '[:lower:]')") ;;
    *) die "$repos/$repo is not $1's clone, its origin '$origin': move it aside" ;;
    esac
    [ -x "$repos/$repo/apply.sh" ] || die "$repos/$repo has no apply.sh that runs"
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    clone) clone "${2-}" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote-files.sh clone OWNER/REPO" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
