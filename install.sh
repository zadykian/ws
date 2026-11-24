#!/bin/sh
# Installs ws's binary, as root, on an Ubuntu server or a Mac, through curl. A server runs it as
# root, a Mac through sudo:
#
#   set -o pipefail
#   curl -fsSL --proto '=https' --tlsv1.2 \
#       https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh
#
# Without pipefail, a curl that fails pipes nothing into sh, which exits 0 having done nothing.
#
# It picks ws's file for linux-amd64, linux-arm64 or darwin-arm64 from a release, and downloads it
# beside /usr/local/bin/ws. It checks the file against that release's ws.sha256 and runs it as
# ws version. Only then it renames it to /usr/local/bin/ws, uid 0, gid 0 and 0755: the one ws of
# a machine.
# A refusal leaves the old ws and no file behind. The release is the latest, or the one
# WS_VERSION names (X.Y.Z, with or without the v). A pre-release, X.Y.Z-rc.N, installs only where
# WS_VERSION names it. It makes /usr/local and /usr/local/bin, root's and 755, where they are
# missing. It refuses one that anyone but root may write, as launchd would refuse ws below it.
# It does not chown, as an old Homebrew in /usr/local would lose its writes. Its last line names
# the next command, ws server setup on Linux and ws macos setup on a Mac, which sets the machine
# up and checks the system. It runs in any POSIX sh, and needs curl and sha256sum, or shasum on a
# Mac.
#
# WS_RELEASES_URL stands in for https://github.com/zadykian/ws/releases, for the tests.
#
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

# The directory of the one ws, and its name there.
bin_dir=/usr/local/bin

# say prints a message of the installer's.
say() {
    printf 'install.sh: %s\n' "$*" >&2
}

# die prints a message of the installer's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# platform sets asset to the name of ws's file for this machine, and next to the command that sets
# it up. The Mac's side is arm64 alone, as the release holds no other build for it.
platform() {
    os=$(uname -s)
    arch=$(uname -m)
    case $os-$arch in
    Linux-x86_64 | Linux-amd64)
        asset=linux-amd64
        next="ws server setup"
        ;;
    Linux-aarch64 | Linux-arm64)
        asset=linux-arm64
        next="ws server setup"
        ;;
    Linux-*) die "ws sets up Linux on x86_64 and aarch64, not on $arch" ;;
    Darwin-arm64)
        asset=darwin-arm64
        next="ws macos setup"
        ;;
    Darwin-*) die "ws sets up a Mac on arm64 alone, not on $arch" ;;
    *) die "ws sets up Linux and macOS, not $os" ;;
    esac
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

# release sets releases to the address of ws's releases, and tag to the release whose file
# installs. That is WS_VERSION's, or none for the latest until latest finds it. where is how
# messages name it.
release() {
    releases=${WS_RELEASES_URL:-https://github.com/zadykian/ws/releases}
    version=${WS_VERSION:-}
    tag=
    if [ -n "$version" ]; then
        # A typo stops here, not at a download that finds no file.
        is_version "${version#v}" || is_rc "${version#v}" ||
            die "WS_VERSION is not a version, X.Y.Z or X.Y.Z-rc.N: $version"
        tag=v${version#v}
    fi
    where="the ${tag:-latest} release of ws"
}

# fetch runs curl over HTTPS alone, TLS 1.2 or later, with the options $@. A reply with an HTTP
# error fails it, rather than its page taking the place of a file. Where WS_RELEASES_URL is set,
# for a test, plain HTTP is allowed too.
fetch() {
    proto='=https'
    [ -z "${WS_RELEASES_URL:-}" ] || proto='=http,https'
    curl -fsS --proto "$proto" --tlsv1.2 "$@"
}

# latest sets tag, where none is set, to the latest release's. GitHub redirects releases/latest to
# the latest release's page, releases/tag/vX.Y.Z, or to releases where there is none. A pre-release
# is never the latest. The tag goes into the download's address, so anything but vX.Y.Z is refused.
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

# sha256 prints the SHA-256 checksum of the file $1, by sha256sum, or by shasum where a Mac has no
# sha256sum.
sha256() {
    if command -v sha256sum >/dev/null; then
        sha256sum <"$1" | awk '{ print $1 }'
    else
        shasum -a 256 <"$1" | awk '{ print $1 }'
    fi
}

# root_dir succeeds where the directory $1 is a real directory that root alone may write.
root_dir() {
    [ ! -L "$1" ] &&
        [ -z "$(find "$1" -maxdepth 0 \( ! -user 0 -o -perm -020 -o -perm -002 \))" ]
}

# directories makes /usr/local and /usr/local/bin where they are missing, uid 0, gid 0 and 0755. It
# refuses one that is not root's alone, and changes no owner: an old Homebrew in /usr/local would
# lose its writes. launchd refuses to run ws below a directory that another user may change.
directories() {
    for dir in /usr/local "$bin_dir"; do
        if [ -e "$dir" ] || [ -L "$dir" ]; then
            root_dir "$dir" || die "$dir is not root's alone, and ws below it would not run as root:" \
                "chown 0:0 $dir, and chmod go-w $dir"
        else
            install -d -m 755 -o 0 -g 0 "$dir" || die "cannot create $dir"
        fi
    done
}

# install_ws puts the release's ws at /usr/local/bin/ws. The download goes into a directory in the
# one ws goes to, so that moving it there is a rename that replaces ws at once. A refusal leaves
# the old ws and no file behind.
install_ws() {
    tmp=
    trap '[ -z "$tmp" ] || rm -rf "$tmp"' EXIT
    trap 'exit 1' HUP INT TERM
    tmp=$(mktemp -d "$bin_dir/.ws.XXXXXX") || die "cannot write to $bin_dir"
    base=$releases/download/$tag
    fetch -L -o "$tmp/ws.sha256" "$base/ws.sha256" || die "cannot download the ws.sha256 of $where"
    fetch -L -o "$tmp/ws" "$base/ws-$asset" || die "cannot download ws-$asset of $where"
    # sha256sum lists each file by its name, with a * before it for a binary one.
    want=$(awk -v file="ws-$asset" '$2 == file || $2 == "*" file { print $1 }' "$tmp/ws.sha256")
    case $want in
    '' | *[!0-9a-f]*) die "the ws.sha256 of $where lists no one SHA-256 for ws-$asset" ;;
    esac
    [ "$(sha256 "$tmp/ws")" = "$want" ] ||
        die "ws-$asset does not match the SHA-256 that the ws.sha256 of $where lists"
    chown 0:0 "$tmp/ws"
    chmod 755 "$tmp/ws"
    got=$("$tmp/ws" version </dev/null 2>/dev/null) || die "the ws downloaded does not run as ws version"
    [ "$got" = "$tag" ] || die "the ws downloaded is version \"$got\", not $tag"
    mv -f "$tmp/ws" "$bin_dir/ws" || die "cannot move ws to $bin_dir"
    say "installed ws of $tag as $bin_dir/ws"
}

main() {
    platform
    release
    [ "$(id -u)" = 0 ] || die "run it as root: through sudo"
    # sudo may keep the caller's umask, and one that lets a group write would leave a ws that
    # others may change.
    umask 022
    command -v curl >/dev/null || die "curl is needed to download ws"
    command -v sha256sum >/dev/null || command -v shasum >/dev/null ||
        die "sha256sum, or shasum, is needed to check ws's SHA-256"
    latest
    directories
    install_ws
    say "next, run: $next"
}

main
