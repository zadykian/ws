#!/bin/sh
# Installs mise and runs ws's bootstrap on an Ubuntu server, as root, through curl:
#
#   set -o pipefail
#   curl -fsSL --proto '=https' --tlsv1.2 \
#       https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
#
# Without pipefail, a curl that fails pipes nothing into sh, which exits 0 having done nothing.
#
# It installs mise at the version this script pins, over /usr/local/bin/mise. It downloads mise's
# binary for this machine and checks it against the SHA-256 that the script holds. It runs the
# binary as mise --version, and only then moves it into place. Then it writes
# /etc/mise/miserc.toml, which names the envs of mise's files that load on this machine. It links
# the tree's plugin, plugins/ws, where mise looks for a system plugin, as below. Last it
# checks a release's tag out in /opt/ws by git, and runs mise bootstrap in /opt/ws/roles/server.
# The release is the latest, or the one WS_VERSION names (X.Y.Z, with or without the v). A
# pre-release, X.Y.Z-rc.N, runs only where WS_VERSION names it. That tree pins ws's binary and
# checks it. Run again, it checks that release's tag out there, dropping any change and any file
# git does not track, and brings the machine back to the release. It refuses a /opt/ws that is not
# root's alone, and a system other than Ubuntu 26.04.
# A Mac follows roles/client/runbook.md instead. It runs in any POSIX sh, and needs curl,
# sha256sum and git.
#
# WS_IN_PLACE=1 runs the bootstrap on the tree in /opt/ws, with no release lookup and no clone.
# WS_RELEASES_URL stands in for https://github.com/zadykian/ws/releases, WS_MISE_URL for
# https://github.com/jdx/mise/releases/download, WS_REPO for https://github.com/zadykian/ws and
# WS_OS_RELEASE for /etc/os-release, for the tests.
#
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

# mise's release, and the SHA-256 of its binary for each Linux target. Where mise moves, take the
# sums from the release's SHASUMS256.txt.
mise_version=2026.11.9
mise_sum_x64=f13f3e5ec88807e0161669b3aa07a91d84478cdf1e95b5b92b63f85a25d42395
mise_sum_arm64=b8dd05af6447cee933928664982a8cb2fa646e6951cfc8539db23ce2554a7b04

# say prints a message of the installer's.
say() {
    printf 'install.sh: %s\n' "$*" >&2
}

# die prints a message of the installer's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# platform sets mise_asset and mise_sum to the name and the checksum of mise's binary for this
# machine. It refuses a system other than Ubuntu 26.04, which the units are written for, by
# /etc/os-release, or by the file WS_OS_RELEASE names, for the tests.
platform() {
    os=$(uname -s)
    arch=$(uname -m)
    case $os-$arch in
    Linux-x86_64 | Linux-amd64)
        mise_asset=linux-x64
        mise_sum=$mise_sum_x64
        ;;
    Linux-aarch64 | Linux-arm64)
        mise_asset=linux-arm64
        mise_sum=$mise_sum_arm64
        ;;
    Linux-*) die "ws sets up Linux on x86_64 and aarch64, not on $arch" ;;
    Darwin-*) die "install.sh sets up an Ubuntu server: a Mac follows roles/client/runbook.md" ;;
    *) die "ws sets up Ubuntu, not $os" ;;
    esac
    release_file=/etc/os-release
    [ -z "${WS_OS_RELEASE:-}" ] || release_file=$WS_OS_RELEASE
    id=$(sed -n 's/^ID=//p' "$release_file" 2>/dev/null | tr -d "\"'")
    version_id=$(sed -n 's/^VERSION_ID=//p' "$release_file" 2>/dev/null | tr -d "\"'")
    [ "$id $version_id" = "ubuntu 26.04" ] ||
        die "ws sets up Ubuntu 26.04 alone, not '$id' '$version_id'"
}

# is_version succeeds where $1 is a version X.Y.Z, as a tag vX.Y.Z holds it: three numbers, none
# empty or with a leading zero.
is_version() {
    case $1 in
    *[!0-9.]* | .* | *. | *..* | *.*.*.* | 0[0-9]* | *.0[0-9]*) return 1 ;;
    *.*.*) return 0 ;;
    *) return 1 ;;
    esac
}

# is_rc succeeds where $1 is a pre-release X.Y.Z-rc.N, as a tag vX.Y.Z-rc.N holds it. X.Y.Z is a
# version, and N a number from 1 with no leading zero.
is_rc() {
    case $1 in
    *-rc.*) ;;
    *) return 1 ;;
    esac
    case ${1##*-rc.} in
    '' | *[!0-9]* | 0*) return 1 ;;
    esac
    is_version "${1%-rc.*}"
}

# release sets releases to the address of ws's releases, repo to ws's repository, and tag to the
# release whose tree runs. That is WS_VERSION's, or none for the latest until latest finds it. where
# is how messages name it.
release() {
    releases=${WS_RELEASES_URL:-https://github.com/zadykian/ws/releases}
    repo=${WS_REPO:-https://github.com/zadykian/ws}
    version=${WS_VERSION:-}
    tag=
    if [ -n "$version" ]; then
        # A typo stops here, not at a clone that finds no tag.
        is_version "${version#v}" || is_rc "${version#v}" ||
            die "WS_VERSION is not a version, X.Y.Z or X.Y.Z-rc.N: $version"
        tag=v${version#v}
    fi
    where="the ${tag:-latest} release of ws"
}

# fetch runs curl over HTTPS alone, TLS 1.2 or later, with the options $@. A reply with an HTTP
# error fails it, rather than its page taking the place of a file. Where WS_RELEASES_URL or
# WS_MISE_URL is set, for a test, plain HTTP is allowed too.
fetch() {
    proto='=https'
    [ -z "${WS_RELEASES_URL:-}${WS_MISE_URL:-}" ] || proto='=http,https'
    curl -fsS --proto "$proto" --tlsv1.2 "$@"
}

# latest sets tag, where none is set, to the latest release's. GitHub redirects releases/latest to
# the latest release's page, releases/tag/vX.Y.Z, or to releases where there is none. A pre-release
# is never the latest. The tag goes into the clone's ref, so anything but vX.Y.Z is refused.
latest() {
    [ -z "$tag" ] || return 0
    page=$(fetch -o /dev/null -w '%{redirect_url}' "$releases/latest") ||
        die "cannot find $where"
    found=${page#"$releases"/tag/v}
    if [ "$found" = "$page" ] || ! is_version "$found"; then
        die "cannot find $where: GitHub redirects to '$page'"
    fi
    tag=v$found
    where="$where, $tag"
}

# sha256 prints the SHA-256 checksum of the file $1.
sha256() {
    sha256sum <"$1" | awk '{ print $1 }'
}

# install_mise puts mise's binary of the pinned version at /usr/local/bin/mise, where that is not
# it already. The download goes into a directory in the one mise goes to, so that moving it there
# is a rename that replaces mise at once. A refusal leaves the old mise and no file behind.
install_mise() {
    dir=/usr/local/bin
    if [ -x "$dir/mise" ] && "$dir/mise" --version 2>/dev/null | grep -q "^$mise_version "; then
        return 0
    fi
    mkdir -p "$dir" || die "cannot create $dir"
    tmp=
    trap '[ -z "$tmp" ] || rm -rf "$tmp"' EXIT
    trap 'exit 1' HUP INT TERM
    tmp=$(mktemp -d "$dir/.mise.XXXXXX") || die "cannot write to $dir"
    base=${WS_MISE_URL:-https://github.com/jdx/mise/releases/download}
    fetch -L -o "$tmp/mise" "$base/v$mise_version/mise-v$mise_version-$mise_asset" ||
        die "cannot download mise $mise_version"
    [ "$(sha256 "$tmp/mise")" = "$mise_sum" ] ||
        die "the mise downloaded does not match the SHA-256 that install.sh holds for $mise_asset"
    chmod 755 "$tmp/mise"
    "$tmp/mise" --version </dev/null 2>/dev/null | grep -q "^$mise_version " ||
        die "the mise downloaded is not version $mise_version"
    mv -f "$tmp/mise" "$dir/mise" || die "cannot move mise to $dir"
    say "installed mise $mise_version as $dir/mise"
}

# miserc writes /etc/mise/miserc.toml by a rename. mise reads it before any config or values file.
miserc() {
    mkdir -p /etc/mise || die "cannot create /etc/mise"
    rc=$(mktemp /etc/mise/.miserc.XXXXXX) || die "cannot write to /etc/mise"
    cat >"$rc" <<TOML
# ws's side, the server, as the set ws: booted where systemd runs. The secrets task adds secrets
# by -E ws,secrets. env sits above the table, which TOML would put it in.
env = ["ws"]
[env_sets]
ws = ["server"{% if "/run/systemd/system" is dir %}, "booted"{% endif %}]
TOML
    chmod 644 "$rc"
    mv -f "$rc" /etc/mise/miserc.toml || die "cannot write /etc/mise/miserc.toml"
}

# plugin links the tree's plugin ws into /usr/share/mise/plugins. mise loads a system plugin there
# as it stands, and installs none. It refuses one that anyone but root may change, or any directory
# on its way, /opt included, and then drops ws's units with a warning alone. So a directory above
# the link, or /opt, that a group or others may write is refused here. mise's two are made root's
# and 755, whatever was there. A [bootstrap.plugins] entry would install the plugin again on
# every run, with a warning. The other such directory, /usr/local/share/mise/plugins, is the data
# directory's own. A plugin there would be both root's and the system's, which mise warns of too.
plugin() {
    for dir in / /usr /usr/share /opt; do
        [ -z "$(find "$dir" -maxdepth 0 \( ! -user 0 -o -perm /022 \))" ] ||
            die "$dir is not root's alone, and mise ignores the plugin ws below it"
    done
    install -d -m 755 -o 0 -g 0 /usr/share/mise /usr/share/mise/plugins ||
        die "cannot create /usr/share/mise/plugins"
    ln -sfn /opt/ws/plugins/ws /usr/share/mise/plugins/ws ||
        die "cannot link /opt/ws/plugins/ws into /usr/share/mise/plugins"
}

# guard refuses a tree in /opt/ws that anyone but root may have changed. mise runs its config, its
# plugin's Lua and its units as root. So a link in its place is refused, and so is a file or
# directory in it that another user owns, or that a group or others may write.
guard() {
    [ ! -L /opt/ws ] || die "/opt/ws is a link: the tree that runs as root must be root's own"
    [ -e /opt/ws ] || return 0
    found=$(find -P /opt/ws \( ! -user 0 -o ! -type l -perm /022 \) -print -quit) ||
        die "cannot read the whole of /opt/ws"
    [ -z "$found" ] || die "/opt/ws is not root's alone, and holds the config that runs as root:" \
        "$(stat -c '%n is of %U, with mode %a' "$found")"
}

# checkout moves the tree in /opt/ws to the tag. A first run makes /opt/ws an empty repository,
# and fetches as a later run does. A change to the tree, and a file git does not track, such as a
# trial's mise.local.toml that would pin another ws, go.
checkout() {
    if [ ! -e /opt/ws ]; then
        git -c init.defaultBranch=ws init -q /opt/ws </dev/null || die "cannot create /opt/ws"
    fi
    [ -e /opt/ws/.git ] || die "/opt/ws holds no checkout: remove it, or set WS_IN_PLACE=1 to run it"
    git -C /opt/ws fetch -q --depth 1 --force "$repo" "refs/tags/$tag:refs/tags/$tag" </dev/null ||
        die "cannot fetch $where into /opt/ws"
    git -C /opt/ws -c advice.detachedHead=false checkout -q --force --detach "refs/tags/$tag" ||
        die "cannot check $where out in /opt/ws"
    git -C /opt/ws clean -q -ffdx || die "cannot clean /opt/ws"
}

# bootstrap runs mise bootstrap in the server's run directory, /opt/ws/roles/server. The tree in
# /opt/ws is the release's, checked out there, or, with WS_IN_PLACE=1, the one that stands there.
# The tree is trusted for this run, as the units' own mise calls inherit it, and for the runs by
# hand after, by mise trust. mise reads no stdin, as the piped script is on it. mise puts the
# tools under the data directory, which is root's own /root/.local/share/mise by default:
# /usr/local/share/mise lets the account run them.
# MISE_LOCKED installs a tool only as the tree's lock holds it, by its URL and checksum, as CI's
# runs and the runbook's do: bootstrap has no --locked.
bootstrap() {
    guard
    MISE_TRUSTED_CONFIG_PATHS=/opt/ws MISE_DATA_DIR=/usr/local/share/mise MISE_LOCKED=1
    export MISE_TRUSTED_CONFIG_PATHS MISE_DATA_DIR MISE_LOCKED
    if [ "${WS_IN_PLACE:-}" = 1 ]; then
        [ -f /opt/ws/mise.toml ] || die "WS_IN_PLACE is set, and /opt/ws holds no mise.toml"
    else
        checkout
    fi
    /usr/local/bin/mise -C /opt/ws/roles/server bootstrap </dev/null
    # While the variable is set, mise trusts the tree and writes no trust. Each config and task
    # is named, as mise trust --all never ends where a role holds an env's file and the root none.
    (
        cd /opt/ws || exit 1
        unset MISE_TRUSTED_CONFIG_PATHS
        for config in mise.toml mise.*.toml roles/server/mise/conf.d/*/mise.*.toml roles/server/mise-tasks/*; do
            [ ! -e "$config" ] || /usr/local/bin/mise trust "$config" </dev/null || exit 1
        done
    )
}

main() {
    platform
    release
    [ "$(id -u)" = 0 ] || die "run it as root: through sudo"
    # sudo may keep the caller's umask, and one that lets a group write would leave a tree that
    # the guard refuses at the next run.
    umask 022
    # mise loads the config of the directory it runs in, and a checkout there is not the tree.
    cd / || die "cannot change to /"
    command -v curl >/dev/null || die "curl is needed to download mise"
    command -v sha256sum >/dev/null || die "sha256sum is needed to check mise's checksum"
    if [ "${WS_IN_PLACE:-}" != 1 ]; then
        command -v git >/dev/null || die "git is needed to clone the tree of $where"
        latest
    fi
    install_mise
    miserc
    plugin
    bootstrap
    if [ "${WS_IN_PLACE:-}" = 1 ]; then
        say "ran the bootstrap on the tree in /opt/ws"
    else
        say "ran the bootstrap of $tag"
    fi
}

main
