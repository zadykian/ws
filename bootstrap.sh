#!/bin/sh
# Sets up this machine from github.com/zadykian/ws. Run it as root on a fresh Ubuntu Server 26.04:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh
#
# or, for a dry run that shows what the playbook would change:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh |
#       sh -s -- --check --diff
#
# It installs the latest release of ws with install.sh, then runs ws server apply, which installs
# ansible-core and runs the playbook that ws embeds. Its arguments go to ansible-playbook. The
# wizard sends it to ws, and a clone of the repository keeps it for the same use.
#
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

installer=https://github.com/zadykian/ws/releases/latest/download/install.sh

# say prints a message of bootstrap.sh's.
say() {
    printf 'bootstrap.sh: %s\n' "$*" >&2
}

# die prints a message of bootstrap.sh's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# install downloads install.sh of the latest release and runs it, which puts ws in /usr/local/bin
# as root. It needs curl and ca-certificates, which Ubuntu Server has.
install_ws() {
    [ "$(id -u)" = 0 ] || die "run it as root"
    tmp=$(mktemp -d)
    say "installing the latest ws"
    if ! curl -fsSL --retry 3 --proto =https --proto-redir =https -o "$tmp/install.sh" "$installer" ||
        ! sh "$tmp/install.sh"; then
        rm -rf "$tmp"
        die "cannot install ws"
    fi
    rm -rf "$tmp"
}

main() {
    # A Mac's ssh passes the Mac's locale on: LC_CTYPE=UTF-8, a name Linux has no locale for.
    # So all that main runs takes C.UTF-8, which libc-bin gives every Ubuntu.
    export LC_ALL=C.UTF-8
    install_ws
    exec /usr/local/bin/ws server apply -- "$@"
}

# Nothing it runs reads stdin, which under curl | sh is the rest of this script. The braces make a
# line cut short a syntax error, rather than main run without its arguments.
{ main "$@" </dev/null; }
