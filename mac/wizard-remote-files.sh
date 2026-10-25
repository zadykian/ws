#!/bin/sh
# The wizard's files script on ws, for another repository's files there, root's word list and ws's
# wired ports. The Mac's wizard sends it to root there over ssh, as it sends wizard-remote.sh:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote-files.sh
#
# - clone OWNER/REPO clones that repository on GitHub into /root/repository/REPO, by root's gh
#   login, where nothing is there. It never pulls a clone, as the repository's apply.sh updates its
#   own. It stops where the path holds no clone, another repository's, or a clone with no apply.sh
#   that runs. It removes REPO's stamp, as apply.sh runs next, so that a run that fails or is cut
#   short leaves none.
# - stamp OWNER/REPO stamps REPO's apply.sh, which exited 0, with the clone's commit and OWNER/REPO.
#   The remote script's status lists the repositories that have a stamp, as OWNER/REPO.
# - words prints git's hash of root's word list, ~/.config/ws/forbidden-words, which ws's hooks
#   read, or - where root has none. words HASH prints the list, while its hash is HASH.
# - link lists ws's wired ports but the uplinks, those that a default route names, a line each. A
#   line holds the port, its carrier, 1 or 0, or - where the port is down, and its IPv4 addresses,
#   or -.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, the repositories' clones, root's word list and sysfs's network interfaces.
# CI's stand-in for ws roots each, a line NAME=PATH, in a directory of its own.
state=/var/lib/ws-wizard
repos=/root/repository
list=/root/.config/ws/forbidden-words
net=/sys/class/net

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
    rm -f "$state/hook-$repo"
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

# stamp OWNER/REPO writes REPO's stamp, hook-REPO in state, by a rename: the clone's commit, which
# apply.sh may have moved, 0, its status, and OWNER/REPO, which the remote script's status lists.
stamp() {
    repo "$1"
    mkdir -p "$state"
    sha=$(git -C "$repos/$repo" rev-parse HEAD 2>/dev/null) || sha=-
    echo "$sha 0 $1" >"$state/.hook-$repo" && mv -f "$state/.hook-$repo" "$state/hook-$repo"
}

# words [HASH] prints git's hash of the word list, or - where there is none. git hashes from /, as
# the wizard does. Given HASH, as it printed it, it prints the list instead, and stops where the
# list's hash is no longer HASH.
words() {
    sum=-
    [ ! -f "$list" ] || sum=$(git -C / hash-object --stdin <"$list")
    if [ -z "${1-}" ]; then
        echo "$sum"
    elif [ "$1" = "$sum" ] && [ "$sum" != - ]; then
        cat "$list"
    else
        die "the word list on ws is no longer $1: run the wizard again"
    fi
}

# ports prints ws's wired ports: a device's, of ARPHRD_ETHER's type 1 and not wireless, but those
# that a default route names, in any table, as an uplink's. A bridge's or bond's member, which
# sysfs's master marks, is left out too, as an uplink may be one. So is a name of other characters
# than letters, digits and ._-, or with a - first, as no argument may hold them.
ports() {
    uplinks=$({
        ip route show default table all
        ip -6 route show default table all
    } 2>/dev/null | awk '{ for (i = 1; i < NF; i++) if ($i == "dev") print $(i + 1) }')
    for path in "$net"/*; do
        port=${path##*/}
        case $port in *[!A-Za-z0-9_.-]* | -*) continue ;; esac
        [ -e "$path/device" ] && [ ! -e "$path/master" ] &&
            [ "$(cat "$path/type" 2>/dev/null)" = 1 ] &&
            [ ! -e "$path/wireless" ] && [ ! -e "$path/phy80211" ] || continue
        printf '%s\n' "$uplinks" | grep -qxF -- "$port" || echo "$port"
    done
}

# link lists the ports, as the header says, with sysfs's carrier, which a port that is down lacks,
# and ip's IPv4 addresses.
link() {
    for port in $(ports); do
        carrier=$(cat "$net/$port/carrier" 2>/dev/null) || carrier=
        addresses=$(ip -o -4 address show dev "$port" 2>/dev/null | awk '{ printf " %s", $4 }')
        echo "$port ${carrier:--}${addresses:- -}"
    done
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    clone) clone "${2-}" ;;
    stamp) stamp "${2-}" ;;
    words) words "${2-}" ;;
    link) link ;;
    *) die "unknown op '${1-}'; usage: wizard-remote-files.sh clone OWNER/REPO |" \
        "stamp OWNER/REPO | words [HASH] | link" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
