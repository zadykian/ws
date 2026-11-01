#!/bin/sh
# Installs ws, the one binary of this repository's releases, each of which publishes this script,
# through curl:
#
#   set -o pipefail
#   curl -fsSL --proto '=https' --tlsv1.2 \
#       https://github.com/zadykian/ws/releases/latest/download/install.sh | sh
#
# Without pipefail, a curl that fails pipes nothing into sh, which exits 0 having installed nothing.
#
# It downloads the binary for this machine, ws-OS-ARCH, from the latest release, or from the one
# WS_VERSION names (X.Y.Z, with or without the v), checks it against the release's ws.sha256, runs
# it as ws version and checks that it prints the release's tag, and only then moves it over
# ~/.local/bin/ws, or over /usr/local/bin/ws as root. Run again, it updates ws. It finds which
# release is the latest once, before either download, so that both come from one release. Macs
# with Apple silicon, and Linux on x86_64 and aarch64, are released for. WS_RELEASES_URL stands in
# for https://github.com/zadykian/ws/releases, for the tests. It runs in any POSIX sh, and needs
# curl and sha256sum, or shasum on a Mac.
#
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

# say prints a message of the installer's.
say() {
    printf 'install.sh: %s\n' "$*" >&2
}

# die prints a message of the installer's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# platform sets binary to the name the releases give ws for this machine.
platform() {
    os=$(uname -s)
    arch=$(uname -m)
    case $os-$arch in
    Darwin-arm64) binary=ws-darwin-arm64 ;;
    Darwin-*) die "ws is released for Macs with Apple silicon, not for $arch: an Intel Mac is not supported" ;;
    Linux-x86_64 | Linux-amd64) binary=ws-linux-amd64 ;;
    Linux-aarch64 | Linux-arm64) binary=ws-linux-arm64 ;;
    Linux-*) die "ws is released for Linux on x86_64 and aarch64, not on $arch" ;;
    *) die "ws is released for macOS and Linux, not $os" ;;
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

# release sets releases to the address of ws's releases, and tag to the release to install,
# WS_VERSION's, or none for the latest until latest finds it, and where to name it in messages.
release() {
    releases=${WS_RELEASES_URL:-https://github.com/zadykian/ws/releases}
    version=${WS_VERSION:-}
    tag=
    if [ -n "$version" ]; then
        # A typo stops here, not at a download that is not found.
        is_version "${version#v}" || die "WS_VERSION is not a version, X.Y.Z: $version"
        tag=v${version#v}
    fi
    where="the ${tag:-latest} release of ws"
}

# fetch runs curl over HTTPS alone, TLS 1.2 or later, with the options $@. A reply with an HTTP
# error fails it, rather than its page taking the place of a file. Where WS_RELEASES_URL is set, for
# a test, plain HTTP is allowed too.
fetch() {
    proto='=https'
    [ -z "${WS_RELEASES_URL:-}" ] || proto='=http,https'
    curl -fsS --proto "$proto" --tlsv1.2 "$@"
}

# latest sets tag, where none is set, to the latest release's: GitHub redirects releases/latest to
# the latest release's page, releases/tag/vX.Y.Z, or to releases where there is none. The tag goes
# into the downloads' addresses, so anything but vX.Y.Z is refused.
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

main() {
    platform
    release
    command -v curl >/dev/null || die "curl is needed to download ws's releases"
    latest
    echo "the release to install is $tag, and its binary $binary"
}

main
