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
# Before the tunnel, it sets up Ghostty: its app and its settings' font from Homebrew's casks, where
# missing, and ws.ghostty, those settings. It sets up cld, for cld restore tabs: cld in
# ~/.local/bin, installed where missing and updated. Once cld has restore tabs, and where Ghostty's
# app is there, config.d/ws gets the watcher's lines. It warns of what it can't install, update or
# write, and goes on (setup-ghostty.sh, setup-cld.sh).
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
# shellcheck source=setup-extras.sh
. "$dir/setup-extras.sh"
# shellcheck source=setup-ghostty.sh
. "$dir/setup-ghostty.sh"
# shellcheck source=setup-cld.sh
. "$dir/setup-cld.sh"

# The daemon's plist. Root's copies are in setup-lib.sh's ws_libexec.
plist=/Library/LaunchDaemons/$ws_label.plist
# The first line of the resolvers of WS_RESOLVERS, in place of lib.sh's, so that setup.sh and a
# script that sources lib.sh alone each replace and remove only their own.
ws_resolver_header="# Written by ws's mac/setup.sh from WS_RESOLVERS in ~/.config/ws/mac.conf."

# program builds ws from the clone into ~/.local/bin, as bootstrap.sh does, so that the copy for
# root has this clone's source, and sets ws_bin. Under --check, it says so where ws is missing.
program() {
    ws_bin=$HOME/.local/bin/ws
    if ws_checking; then
        [ -x "$ws_bin" ] || ws_say "would build ws into $ws_bin"
        return 0
    fi
    ws_build "${dir%/*}" "$ws_bin" || ws_die "cannot build ws"
}

# copies puts root's copy of ws in /usr/local/libexec/ws-tunnel. The user owns ~/.local/bin, so a
# root daemon that ran that file would let anything that runs as the user become root at its next
# start. /usr/local/libexec is made where missing; every directory above the copy must be root's and
# writable by root alone. A rebuilt ws reaches the copy at the next run.
copies() {
    [ -n "$(ws_root_attrs /usr/local)" ] || ws_dir_root /usr/local 755
    [ -n "$(ws_root_attrs /usr/local/libexec)" ] || ws_dir_root /usr/local/libexec 755
    for each in / /usr /usr/local /usr/local/libexec; do
        attrs=$(ws_root_attrs "$each")
        [ -n "$attrs" ] || continue
        case $attrs in
        0:*' '[0-7][0145][0145]) ;;
        *) ws_die "$each, above root's copy, is not root's alone: owner, group and mode $attrs" ;;
        esac
    done
    ws_dir_root "$ws_libexec" 755
    ws_changed=0
    ws_copy_root "$ws_bin" "$ws_libexec/ws" 755
    programs_changed=$ws_changed
}

# key makes the Mac's WireGuard key where it is missing, never shown, with root's ws, which keeps
# it in a file only root reads, and sets key_made.
key() {
    ws_dir_root "$ws_etc" 700
    ws_dir_root "$ws_etc/routes.d" 755
    key_made=0
    if [ -n "$(ws_root_attrs "$ws_etc/private.key")" ]; then
        ws_root_fix "$ws_etc/private.key" 600 "$(ws_root_attrs "$ws_etc/private.key")"
        return 0
    fi
    key_made=1
    if ws_checking; then
        ws_say "would make the Mac's WireGuard key, $ws_etc/private.key"
        return 0
    fi
    sudo "$ws_libexec/ws" tunnel key >/dev/null || ws_die "cannot make the Mac's WireGuard key"
    ws_say "made the Mac's WireGuard key, $ws_etc/private.key"
}

# tell writes the Mac's public key, once key has made it, to ~/.config/ws/tunnel.pub at each run,
# where the wizard's Tunnel stage reads it without sudo. It prints the key, with the commands that
# make it ws's peer, where the key is new or ws's key isn't set yet.
tell() {
    { [ "$key_made" = 0 ] || ! ws_checking; } || return 0
    # A check before the first run has no copy of ws to ask.
    { ! ws_checking || [ -x "$ws_libexec/ws" ]; } || return 0
    public=$(sudo "$ws_libexec/ws" tunnel key) ||
        ws_die "cannot read the Mac's public key"
    [ -d "$HOME/.config" ] || ws_dir "$HOME/.config" 755
    [ -d "$HOME/.config/ws" ] || ws_dir "$HOME/.config/ws" 700
    printf '%s\n' "$public" | ws_put "$HOME/.config/ws/tunnel.pub" 644
    { [ "$key_made" = 1 ] || [ -z "$WS_PUBLIC_KEY" ]; } && ! ws_checking || return 0
    # /etc/wireguard and bootstrap.sh are root's, so a WS_USER other than root goes through sudo,
    # with a terminal for its password. The key, which is public, goes in the remote command, so
    # that ssh's stdin stays the terminal.
    sudo=
    ssh_t=
    if [ "$WS_USER" != root ]; then
        sudo='sudo '
        ssh_t='-t '
    fi
    cat <<EOF
The Mac's WireGuard public key is $public
Make it ws's peer, mac, through ws-ssh:

    ssh ${ssh_t}ws-ssh "printf '%s\\n' '$public' | ${sudo}tee /etc/wireguard/mac.pub >/dev/null"
    ssh ${ssh_t}ws-ssh ${sudo}/root/repository/ws/bootstrap.sh --tags wireguard
EOF
    if [ -z "$WS_PUBLIC_KEY" ]; then
        cat <<EOF

Then set WS_PUBLIC_KEY in ~/.config/ws/mac.conf to ws's key, which that run prints, and run
mac/setup.sh again, which installs the tunnel's daemon.
EOF
    fi
}

# tunnel_conf prints tunnel.conf, the values the daemon takes from mac.conf, so that a root daemon
# reads no file the user can write.
tunnel_conf() {
    echo "# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf."
    echo "WS_PUBLIC_KEY=$WS_PUBLIC_KEY"
    if [ -n "$WS_LAN" ]; then
        echo "WS_LAN=$WS_LAN"
        echo "WS_LAN_ADDRESS=$WS_LAN_ADDRESS"
    fi
    [ -z "$WS_REMOTE" ] || echo "WS_REMOTE=$WS_REMOTE"
    echo "WS_PORT=${WS_PORT:-51820}"
}

# launchd runs, or has launchd run, the daemon: bootstrap where it isn't loaded, bootout and
# bootstrap where the plist changed, kickstart -k where a program or the key changed, and SIGHUP
# where tunnel.conf alone changed, which the daemon takes without dropping the session. A daemon
# that launchd is about to start again, as after a failure, has no process to signal: kickstart
# starts it now, and it reads tunnel.conf as it starts.
launchd() {
    if ! ws_tunnel_loaded; then
        launchd_do bootstrap system "$plist"
    elif [ "$plist_changed" = 1 ]; then
        launchd_do bootout "system/$ws_label"
        launchd_do bootstrap system "$plist"
    elif [ "$programs_changed" = 1 ] || [ "$key_made" = 1 ]; then
        launchd_do kickstart -k "system/$ws_label"
    elif [ "$conf_changed" = 1 ]; then
        if ws_checking; then
            ws_say "would run launchctl kill SIGHUP system/$ws_label"
        elif sudo launchctl kill SIGHUP "system/$ws_label" >/dev/null 2>&1; then
            ws_say "ran launchctl kill SIGHUP system/$ws_label"
        else
            launchd_do kickstart -k "system/$ws_label"
        fi
    fi
}

# launchd_do ARGS... runs sudo launchctl ARGS. A bootstrap right after a bootout may find the
# service still on its way out, so it tries for 10 s.
launchd_do() {
    if ws_checking; then
        ws_say "would run launchctl $*"
        return 0
    fi
    tries=1
    until err=$(sudo launchctl "$@" 2>&1); do
        if [ "$1" != bootstrap ] || [ "$tries" -ge 10 ]; then
            ws_die "launchctl $* failed: $err; is ws-tunnel on under System Settings ›" \
                "General › Login Items & Extensions, Allow in the Background?"
        fi
        tries=$((tries + 1))
        sleep 1
    done
    ws_say "ran launchctl $*"
}

# networks has the daemon route WS_TUNNEL_NETWORKS, from routes.d/mac-conf, and macOS ask the
# nameservers of WS_RESOLVERS for their domains, through lib.sh. A domain whose file lacks
# setup.sh's header stops the run, with a way out that ws_resolver's message lacks. A resolver
# with the header goes once WS_RESOLVERS no longer names its domain. Under --check, key makes no
# routes.d, where ws_tunnel_routes would stop, so a line says what would go there.
# shellcheck disable=SC2046,SC2086,SC2153 # WS_RESOLVERS is ws_conf's, with no character that globs
networks() {
    if ws_checking && [ -z "$(ws_root_attrs "$ws_etc/routes.d")" ]; then
        [ -z "$WS_TUNNEL_NETWORKS" ] || ws_say would put $WS_TUNNEL_NETWORKS in \
            "$ws_etc/routes.d/mac-conf once routes.d is made"
    else
        ws_tunnel_routes mac-conf $WS_TUNNEL_NETWORKS
    fi
    domains=' '
    for pair in $WS_RESOLVERS; do
        domain=${pair%%=*}
        domains="$domains$domain "
        if [ -n "$(ws_root_attrs "$ws_resolvers/$domain")" ] &&
            [ "$(sudo head -n 1 "$ws_resolvers/$domain")" != "$ws_resolver_header" ]; then
            ws_die "$ws_resolvers/$domain is another script's, or written by hand: have that" \
                "script call ws_resolver $domain with no address, or move the file away"
        fi
        ws_resolver "$domain" $(echo "${pair#*=}" | tr , ' ')
    done
    for file in "$ws_resolvers"/*; do
        case $domains in *" ${file##*/} "*) continue ;; esac
        [ "$(sudo head -n 1 "$file" 2>/dev/null)" != "$ws_resolver_header" ] ||
            ws_resolver "${file##*/}"
    done
}

# tunnel sets up the tunnel's daemon, then the networks and resolvers of mac.conf.
tunnel() {
    program
    copies
    key
    if [ -n "$WS_PUBLIC_KEY" ]; then
        ws_tmp
        conf=$ws_tmpfile
        tunnel_conf >"$conf"
        ws_changed=0
        ws_put_root "$ws_etc/tunnel.conf" 600 <"$conf"
        conf_changed=$ws_changed
        rm -f "$conf"
        ws_changed=0
        ws_put_root "$plist" 644 <"$dir/$ws_label.plist"
        plist_changed=$ws_changed
        launchd
        networks
    fi
    tell
}

# step_ws builds ws from the clone into a scratch directory, which the run removes, for the steps
# that ws setup runs. Its bytes are those of the ws that program builds into ~/.local/bin.
step_ws() {
    step_dir=$(mktemp -d "${TMPDIR:-/tmp}/ws.XXXXXXXX") || ws_die "cannot make a scratch directory"
    trap 'rm -rf "$step_dir"' EXIT
    ws_build "${dir%/*}" "$step_dir/ws" || ws_die "cannot build ws"
}

# step NAME runs the step NAME of ws setup, which prints its changes under --check and makes none.
step() {
    if ws_checking; then
        WS_PROG=$ws_prog "$step_dir/ws" setup --check "$1" || exit 1
    else
        WS_PROG=$ws_prog "$step_dir/ws" setup "$1" || exit 1
    fi
}

main() {
    ws_args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not root: it asks sudo where it needs root"
    ws_conf "${WS_CONF:-$HOME/.config/ws/mac.conf}"
    # The ssh step sends ws through the tunnel once launchd has its daemon, so that a run that stops
    # in tunnel leaves ssh ws on the ways of ws-ssh. A run that loads the daemon, or installs a cld
    # with restore tabs, needs config.d/ws again, so the step runs once more after them.
    # ghostty_mac and cld_mac warn rather than stop, and come before the tunnel, so that brew's and
    # install.sh's output comes before the Mac's key.
    via_tunnel=0
    [ -z "$WS_PUBLIC_KEY" ] || ! { ws_checking || ws_tunnel_loaded; } || via_tunnel=1
    cld_tabs_check
    step_ws
    step ssh
    was=$via_tunnel$cld_tabs
    extras
    ghostty_mac
    cld_mac
    tunnel
    [ "$via_tunnel" != 0 ] || [ -z "$WS_PUBLIC_KEY" ] || ! ws_tunnel_loaded || via_tunnel=1
    [ "$via_tunnel$cld_tabs" = "$was" ] || step ssh
}

main "$@"
