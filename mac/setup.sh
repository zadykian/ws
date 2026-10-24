#!/bin/sh
# Sets up the Mac that works on ws, the server this repository sets up. Run it as yourself, from a
# clone of the repository, after writing ~/.config/ws/mac.conf:
#
#   ~/repository/ws/mac/setup.sh [--check]
#
# It reads mac.conf without running it, and writes ~/.ssh/config.d/ws. Its host ws goes through the
# WireGuard tunnel to ws, and ws-ssh reaches ws over the cable, the home LAN or the router's port
# forward, the first of these the Mac is on. Until mac.conf has WS_PUBLIC_KEY and launchd has the
# tunnel's daemon, ws goes the ways ws-ssh does. It puts `Include config.d/ws` first in
# ~/.ssh/config. With sudo, it gives the cable's address to the service WS_CABLE_SERVICE names. With
# WS_GIT_NAME, it makes git's signing key and ~/.config/ws/gitconfig, which it includes at the top
# of ~/.gitconfig, and installs Homebrew's gh where PATH has none.
#
# Then, with sudo, the tunnel: Homebrew's wireguard-tools where missing; root's copies of wg,
# wireguard-go, ws-tunnel and its functions in /usr/local/libexec/ws-tunnel; the Mac's key, made
# once, and tunnel.conf in /etc/ws-tunnel; and the LaunchDaemon that runs ws-tunnel. Without
# WS_PUBLIC_KEY, ws's key, it makes the Mac's key, prints it, and installs no daemon yet. Once
# launchd has the daemon, it sends WS_TUNNEL_NETWORKS into the tunnel and writes WS_RESOLVERS'
# files in /etc/resolver.
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
# shellcheck source=setup-extras.sh
. "$dir/setup-extras.sh"
# shellcheck source=setup-ssh.sh
. "$dir/setup-ssh.sh"

# The daemon's files.
libexec=/usr/local/libexec/ws-tunnel
plist=/Library/LaunchDaemons/$ws_label.plist
# The first line of the resolvers of WS_RESOLVERS, in place of lib.sh's, so that setup.sh and a
# script that sources lib.sh alone each replace and remove only their own.
ws_resolver_header="# Written by ws's mac/setup.sh from WS_RESOLVERS in ~/.config/ws/mac.conf."

# tools installs Homebrew's wireguard-tools where missing, which brings wireguard-go, and sets
# wg_from and go_from to their programs.
tools() {
    brew=$(command -v brew) || ws_die "install Homebrew first: https://brew.sh"
    if ! "$brew" list --formula wireguard-tools >/dev/null 2>&1; then
        if ws_checking; then
            ws_say "would run brew install wireguard-tools"
        else
            ws_say "installing wireguard-tools"
            "$brew" install wireguard-tools >&2 || ws_die "brew install wireguard-tools failed"
        fi
    fi
    wg_from=$("$brew" --prefix wireguard-tools 2>/dev/null)/bin/wg
    go_from=$("$brew" --prefix wireguard-go 2>/dev/null)/bin/wireguard-go
}

# copies puts root's copies of ws-tunnel, its functions, wg and wireguard-go in
# /usr/local/libexec/ws-tunnel. The user owns Homebrew's prefix, so a root daemon that ran
# Homebrew's files would let anything that runs as the user become root at its next start.
# /usr/local/libexec is made where missing; every directory above the copies must be root's and
# writable by root alone. A brew upgrade reaches the copies at the next run. The functions go
# first, so that a daemon that starts in between finds those its copy sources.
copies() {
    [ -n "$(ws_root_attrs /usr/local)" ] || ws_dir_root /usr/local 755
    [ -n "$(ws_root_attrs /usr/local/libexec)" ] || ws_dir_root /usr/local/libexec 755
    for each in / /usr /usr/local /usr/local/libexec; do
        attrs=$(ws_root_attrs "$each")
        [ -n "$attrs" ] || continue
        case $attrs in
        0:*' '[0-7][0145][0145]) ;;
        *) ws_die "$each, above root's copies, is not root's alone: owner, group and mode $attrs" ;;
        esac
    done
    ws_dir_root "$libexec" 755
    ws_changed=0
    for each in tunnel-conf.sh tunnel-paths.sh; do
        ws_put_root "$libexec/$each" 644 <"$dir/$each"
    done
    ws_put_root "$libexec/ws-tunnel" 755 <"$dir/ws-tunnel"
    if [ -e "$wg_from" ]; then
        ws_copy_root "$wg_from" "$libexec/wg" 755
        ws_copy_root "$go_from" "$libexec/wireguard-go" 755
    else
        ws_say "would copy wg and wireguard-go once wireguard-tools is installed"
    fi
    programs_changed=$ws_changed
}

# key makes the Mac's WireGuard key where it is missing, never shown, under umask 077, with root's
# wg, and sets key_made.
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
    sudo sh -c 'umask 077 && "$1" genkey >"$2.new" && mv -f "$2.new" "$2"' \
        sh "$libexec/wg" "$ws_etc/private.key" || ws_die "cannot make $ws_etc/private.key"
    ws_say "made the Mac's WireGuard key, $ws_etc/private.key"
}

# tell prints the Mac's public key, with the commands that make it ws's peer, where the key is new
# or ws's key isn't set yet.
tell() {
    [ "$key_made" = 1 ] || [ -z "$WS_PUBLIC_KEY" ] || return 0
    ws_checking && return 0
    public=$(sudo sh -c '"$1" pubkey <"$2"' sh "$libexec/wg" "$ws_etc/private.key") ||
        ws_die "cannot read the Mac's public key"
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
# shellcheck disable=SC2046,SC2086 # ws_conf let in no character that globs, so the lists split bare
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
    tools
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

main() {
    ws_args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not root: it asks sudo where it needs root"
    ws_conf "$HOME/.config/ws/mac.conf"
    # ssh_files sends ws through the tunnel once launchd has its daemon, so that a run that stops
    # in tunnel leaves ssh ws on the ways of ws-ssh. A run that loads the daemon writes config.d/ws
    # again, and --check shows the file of such a run. ws_own_hosts warns once, between the two.
    via_tunnel=0
    if [ -n "$WS_PUBLIC_KEY" ] && { ws_checking || ws_tunnel_loaded; }; then
        via_tunnel=1
    fi
    ssh_files
    ws_own_hosts "$HOME/.ssh/config" ws ws-ssh
    extras
    tunnel
    if [ "$via_tunnel" = 0 ] && [ -n "$WS_PUBLIC_KEY" ] && ws_tunnel_loaded; then
        via_tunnel=1
        ssh_files
    fi
}

main "$@"
