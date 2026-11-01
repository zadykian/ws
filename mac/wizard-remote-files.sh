#!/bin/sh
# The wizard's files script on ws, for another repository's command there, root's word list and the
# cable's port. The Mac's wizard sends it to root there over ssh, as it sends wizard-remote.sh:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote-files.sh
#
# - bundle OWNER/REPO[@TAG] prints the path for a git bundle of that repository's origin/main, or
#   given a tag, vX.Y.Z, of that tag, in state as hook-REPO.bundle, where /root/repository/REPO is
#   missing, or the tag is not its HEAD, and removes an old one. The wizard sends the bundle there
#   by a call of its own, as this script takes stdin.
# - clone OWNER/REPO[@TAG] clones that repository into /root/repository/REPO, where nothing is
#   there, from that bundle, on branch main with origin at https://github.com/OWNER/REPO.git, and
#   stops where the Mac sent none. Given a tag, the clone is detached at it, and so is an existing
#   one, from a bundle, where it changed nothing tracked and the tag is not another commit there.
#   Without a tag, it never pulls a clone, as the repository's apply.sh updates its own. It removes
#   the bundle. It stops where the path holds no clone, another repository's, or a clone with no
#   apply.sh that runs. It removes REPO's stamp, as apply.sh runs next, so that a run that fails
#   or is cut short leaves none.
# - arch prints ws's architecture as a release names it, amd64 or arm64, and stops on another.
# - stage OWNER/REPO@TAG prints the path where the wizard sends REPO's command, in state as
#   hook-REPO.bin, and removes an old one. The wizard sends it by a call of its own, as this script
#   takes stdin.
# - install OWNER/REPO@TAG SHA256 puts the sent command at bin/REPO-TAG in state, root's alone, mode
#   700, where its SHA-256 is SHA256, which the Mac checked, and stops where the Mac sent none. It
#   removes REPO's other commands, and its stamp, as the command runs next, so that a run that
#   fails or is cut short leaves none.
# - have OWNER/REPO@TAG prints the installed command's path, or nothing where there is none.
# - stamp OWNER/REPO@TAG stamps REPO's command, which exited 0, with the command's SHA-256 and the
#   value. The remote script's status lists the repositories that have a stamp, as the value.
# - words prints git's hash of root's word list, ~/.config/ws/forbidden-words, which ws's hooks
#   read, or - where root has none. words HASH prints the list, while its hash is HASH.
# - wait PORT prints what link prints, below, once PORT holds 10.77.0.1, or after 10 s.
# - link lists ws's wired ports but the uplinks, those that a default route names, a line each. A
#   line holds the port, its carrier, 1 or 0, or - where the port is down, and its IPv4 addresses,
#   or -. link PORT names that port in ws's /etc/ws/host_vars.yml as
#   network_direct_link_interface, the port of the Mac's cable, which the network and wireguard
#   roles read.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, sent commands and installed ones, root's word list, sysfs's network
# interfaces and the machine's values for ws's playbook. CI's stand-in for ws roots each, a line
# NAME=PATH, in a directory of its own.
state=/var/lib/ws-wizard
repos=/root/repository
list=/root/.config/ws/forbidden-words
net=/sys/class/net
vars=/etc/ws/host_vars.yml

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote-files.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# repo OWNER/REPO@TAG sets slug to OWNER/REPO, repo to REPO and tag to TAG, where OWNER/REPO names a
# repository on GitHub, as mac.conf's WS_HOOK does. Each side of the / is letters, digits and ._-,
# and neither . or .. alone, and REPO isn't ws, in any letter case, which APFS and GitHub ignore.
# TAG is a release, vX.Y.Z, and required. It stops otherwise. It sets pack, REPO's sent command in
# state, and bin, where it is installed, as well.
repo() {
    slug=${1%%@*}
    tag=
    case $1 in *@*) tag=${1#*@} ;; esac
    case $slug in
    *[!A-Za-z0-9._/-]* | */*/* | /* | */ | ./* | ../* | */. | */.. | */[Ww][Ss]) ;;
    */*)
        repo=${slug#*/}
        pack=$state/hook-$repo.bin
        bin=$state/bin/$repo-$tag
        printf '%s\n' "$tag" | grep -Eqx 'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)' ||
            die "'$tag' is not a release, as vX.Y.Z, and WS_HOOK needs one"
        return 0
        ;;
    esac
    die "'$slug' is not a repository on GitHub, as OWNER/REPO, with a REPO other than ws"
}

# arch prints ws's architecture as a release names it.
arch() {
    case $(uname -m) in
    x86_64 | amd64) echo amd64 ;;
    aarch64 | arm64) echo arm64 ;;
    *) die "ws is a $(uname -m), and the command has a build for amd64 and arm64 alone" ;;
    esac
}

# digest FILE prints the SHA-256 of the file.
digest() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d ' ' -f 1
    else
        shasum -a 256 "$1" | cut -d ' ' -f 1
    fi
}

# stage OWNER/REPO@TAG prints the path for REPO's command, as the header says.
stage() {
    repo "$1"
    rm -f "$pack"
    mkdir -p "$state"
    echo "$pack"
}

# install OWNER/REPO@TAG SHA256 installs the command, as the header says. The file is moved whole,
# so that no failure leaves half a command for the next run to take.
install() {
    repo "$1"
    rm -f "$state/hook-$repo" "$state/bin/$repo"-v*
    [ -f "$pack" ] || die "ws lacks $repo's command, and the Mac sent none: run the wizard again"
    case ${2-} in
    *[!0-9a-f]* | '') die "'${2-}' is not a SHA-256" ;;
    esac
    if [ "$(digest "$pack")" != "$2" ]; then
        rm -f "$pack"
        die "the command of $slug@$tag that reached ws is not the one the Mac checked: run the wizard again"
    fi
    if ! { (umask 077 && mkdir -p "$state/bin") && chmod 700 "$state/bin" "$pack" && mv -f "$pack" "$bin"; }; then
        die "cannot install $bin"
    fi
    say "installed $slug@$tag at $bin"
}

# have OWNER/REPO@TAG prints the installed command's path, where there is one.
have() {
    repo "$1"
    [ ! -f "$bin" ] || echo "$bin"
}

# bundle OWNER/REPO[@TAG] prints the path for REPO's bundle, as the header says.
bundle() {
    repo "$1"
    rm -f "$pack"
    if [ -e "$repos/$repo" ] || [ -L "$repos/$repo" ]; then
        [ -n "$tag" ] || return 0
        cur=$(git -C "$repos/$repo" rev-parse -q --verify HEAD 2>/dev/null) || cur=
        [ -z "$cur" ] || [ "$(git -C "$repos/$repo" rev-parse -q --verify "refs/tags/$tag^{commit}" \
            2>/dev/null)" != "$cur" ] || return 0
    fi
    mkdir -p "$state"
    echo "$pack"
}

# unbundle OWNER/REPO[@TAG] makes the clone in repos from REPO's bundle. The clone is built beside
# its place and moved there whole, so that no failure leaves half of one for the next run to take.
# The bundle holds origin/main, so the branch main is made from it, with that as its upstream. Given
# a tag, it holds that tag alone, which the clone is detached at.
unbundle() {
    [ -f "$pack" ] || die "ws lacks $repos/$repo, and the Mac sent no bundle: run the wizard again"
    part=$repos/.$repo.part
    rm -rf "$part"
    mkdir -p "$repos"
    if git init -q -b main "$part" &&
        git -C "$part" remote add origin "https://github.com/$slug.git" &&
        if [ -n "$tag" ]; then
            git -C "$part" fetch -q "$pack" "refs/tags/$tag:refs/tags/$tag" &&
                git -C "$part" checkout -q --detach "$tag"
        else
            git -C "$part" fetch -q "$pack" +refs/remotes/origin/main:refs/remotes/origin/main &&
                git -C "$part" checkout -q -B main --track origin/main
        fi && mv "$part" "$repos/$repo"; then
        return 0
    fi
    rm -rf "$part" "$pack"
    die "git cannot make the clone of $slug from its bundle, as above: run the wizard again"
}

# retag detaches REPO's clone at the tag, from the bundle, where the clone changed nothing tracked.
# The fetch never moves a tag the clone has, so that a tag moved on GitHub stops it.
retag() {
    if [ -n "$(git -C "$repos/$repo" status --porcelain --untracked-files=no)" ]; then
        rm -f "$pack"
        die "$repos/$repo has changes, so $tag can't be checked out: commit or reset them"
    fi
    git -C "$repos/$repo" fetch "$pack" "refs/tags/$tag:refs/tags/$tag" &&
        git -C "$repos/$repo" checkout -q --detach "$tag" && return 0
    rm -f "$pack"
    die "git cannot check out $tag of $slug in $repos/$repo, as above, where a tag moved on GitHub stops it"
}

# clone OWNER/REPO[@TAG] clones the repository in repos, as the header says. A clone there must be
# OWNER/REPO's: its origin, by https or ssh as gh sets it, names it on GitHub, letter case aside.
clone() {
    repo "$1"
    rm -f "$state/hook-$repo"
    fresh=
    if [ ! -e "$repos/$repo" ] && [ ! -L "$repos/$repo" ]; then
        unbundle
        say "cloned $slug into $repos/$repo"
        fresh=1
    fi
    [ -e "$repos/$repo/.git" ] || die "$repos/$repo holds no clone: move it aside"
    origin=$(git -C "$repos/$repo" remote get-url origin 2>/dev/null) || origin=
    case $(printf '%s\n' "${origin%.git}" | tr '[:upper:]' '[:lower:]') in
    *[/@]github.com[/:]"$(printf '%s\n' "${slug%.git}" | tr '[:upper:]' '[:lower:]')") ;;
    *) die "$repos/$repo is not $slug's clone, its origin '$origin': move it aside" ;;
    esac
    [ -z "$fresh" ] && [ -n "$tag" ] && [ -f "$pack" ] && retag
    rm -f "$pack"
    [ -x "$repos/$repo/apply.sh" ] || die "$repos/$repo has no apply.sh that runs"
}

# stamp OWNER/REPO@TAG writes REPO's stamp, hook-REPO in state, by a rename: the command's SHA-256,
# 0, its status, and the whole value, which the remote script's status lists.
stamp() {
    repo "$1"
    mkdir -p "$state"
    [ -f "$bin" ] || die "$bin is missing, so there is nothing to stamp"
    echo "$(digest "$bin") 0 $1" >"$state/.hook-$repo" && mv -f "$state/.hook-$repo" "$state/hook-$repo"
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

# link_wait PORT runs link once PORT holds 10.77.0.1, which networkd gives it a moment after the
# network role's networkctl reload, and later for a port that was down. It reads ip each second, 10
# s at most, so that the wizard waits in one login, which a key may take a touch for.
link_wait() {
    ports | grep -qxF -- "${1-}" || die "'${1-}' is none of the wired ports that link lists"
    n=0
    until ip -o -4 address show dev "$1" 2>/dev/null | grep -q ' 10\.77\.0\.1/' || [ "$n" = 10 ]; do
        sleep 1
        n=$((n + 1))
    done
    link
}

# link [PORT] lists the ports, as the header says, with sysfs's carrier, which a port that is down
# lacks, and ip's IPv4 addresses. Given PORT, one that it lists, it names it in vars, where that
# names another, by a rename that keeps the file's other lines and its mode. A new file is 0600 in
# a 0755 directory, and starts with ---, as yamllint asks of YAML.
link() {
    if [ -z "${1-}" ]; then
        for port in $(ports); do
            carrier=$(cat "$net/$port/carrier" 2>/dev/null) || carrier=
            addresses=$(ip -o -4 address show dev "$port" 2>/dev/null | awk '{ printf " %s", $4 }')
            echo "$port ${carrier:--}${addresses:- -}"
        done
        return 0
    fi
    ports | grep -qxF -- "$1" || die "'$1' is none of the wired ports that link lists"
    [ "$(sed -n 's/^network_direct_link_interface: *"\{0,1\}\([A-Za-z0-9_.-]*\).*/\1/p' "$vars" \
        2>/dev/null)" != "$1" ] || return 0
    [ -d "${vars%/*}" ] || { mkdir -p "${vars%/*}" && chmod 755 "${vars%/*}"; }
    umask 077
    [ ! -f "$vars" ] || cp -p "$vars" "$vars.new"
    {
        [ -s "$vars" ] || echo ---
        cat "$vars" 2>/dev/null | awk -v line="network_direct_link_interface: \"$1\"" '
            /^network_direct_link_interface:/ { if (!done++) print line; next }
            { print }
            END { if (!done) print line }'
    } >"$vars.new"
    mv -f "$vars.new" "$vars"
    say "wrote $vars"
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    bundle) bundle "${2-}" ;;
    clone) clone "${2-}" ;;
    arch) arch ;;
    stage) stage "${2-}" ;;
    install) install "${2-}" "${3-}" ;;
    have) have "${2-}" ;;
    stamp) stamp "${2-}" ;;
    words) words "${2-}" ;;
    wait) link_wait "${2-}" ;;
    link) link "${2-}" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote-files.sh bundle OWNER/REPO@TAG | clone OWNER/REPO@TAG | arch | stage OWNER/REPO@TAG |" \
        "install OWNER/REPO@TAG SHA256 | have OWNER/REPO@TAG | stamp OWNER/REPO@TAG | words [HASH] | wait PORT | link [PORT]" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
