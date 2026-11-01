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

main() {
    platform
    echo "the binary for this machine is $binary"
}

main
