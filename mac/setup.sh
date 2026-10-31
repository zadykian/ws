#!/bin/sh
# Sets up the Mac that works on ws, the server this repository sets up. Run it as yourself, from a
# clone of the repository, after writing ~/.config/ws/mac.conf, or the file WS_CONF names:
#
#   ~/repository/ws/mac/setup.sh [--check]
#
# It reads mac.conf without running it, and writes ~/.ssh/config.d/ws. Its host ws goes through the
# WireGuard tunnel to ws, and ws-ssh reaches ws over the cable, the home LAN or the router's port
# forward, the first of these the Mac is on. Until mac.conf has WS_PUBLIC_KEY and launchd has the
# tunnel's daemon, ws goes the ways ws-ssh does. It puts `Include config.d/ws` first in
# ~/.ssh/config. With sudo, it gives the cable's address to the service WS_CABLE_SERVICE names. With
# WS_GIT_NAME, it makes git's signing key and ~/.config/ws/gitconfig, which it includes at the top
# of ~/.gitconfig, and installs Homebrew's gh where PATH has none. gh then adds the key to GitHub.
# It installs JetBrains Gateway from Homebrew's cask where its app is missing.
#
# Then, with sudo, the tunnel: ws, the Go CLI in mac/, built into ~/.local/bin and copied to
# /usr/local/libexec/ws-tunnel as root's; the Mac's key, made once by `ws tunnel key`, and
# tunnel.conf in /etc/ws-tunnel; and the LaunchDaemon that runs `ws tunnel run`. Without
# WS_PUBLIC_KEY, ws's key, it makes the Mac's key, prints it, and installs no daemon yet. Once
# launchd has the daemon, it sends WS_TUNNEL_NETWORKS into the tunnel and writes WS_RESOLVERS'
# files in /etc/resolver.
# It keeps the Mac's public key in ~/.config/ws/tunnel.pub too, which the wizard reads.
#
# Before the tunnel, it sets up Ghostty, its app, font and ws.ghostty, and cld in ~/.local/bin, for
# cld restore tabs: once cld has them, and where Ghostty's app is there, config.d/ws gets the
# watcher's lines. It warns of what it can't install, update or write, and goes on (ws setup
# ghostty, ws setup cld).
#
# A run changes only what differs, so a second prints nothing but warnings once mac.conf has
# WS_PUBLIC_KEY. Until then, each run prints the Mac's public key again, with the commands that make
# it ws's peer. --check prints each change as a diff and makes none. docs/mac.md has the rest.
#
# shellcheck source-path=SCRIPTDIR

set -eu

dir=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=lib.sh
. "$dir/lib.sh"
# shellcheck source=setup-lib.sh
. "$dir/setup-lib.sh"
# shellcheck source=build.sh
. "$dir/build.sh"

# run builds ws from the clone into a scratch directory, which the run removes, and has it run every
# step of ws setup in setup.sh's order: ssh, the cable, git, Gateway, Ghostty, cld, the tunnel, the
# networks, and ssh again. Under --check, each step prints its changes and makes none.
run() {
    step_dir=$(mktemp -d "${TMPDIR:-/tmp}/ws.XXXXXXXX") || ws_die "cannot make a scratch directory"
    trap 'rm -rf "$step_dir"' EXIT
    ws_build "${dir%/*}" "$step_dir/ws" || ws_die "cannot build ws"
    if ws_checking; then
        WS_MAC_DIR=$dir WS_PROG=$ws_prog "$step_dir/ws" setup --check || exit 1
    else
        WS_MAC_DIR=$dir WS_PROG=$ws_prog "$step_dir/ws" setup || exit 1
    fi
}

main() {
    ws_args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not root: it asks sudo where it needs root"
    run
}

main "$@"
