#!/bin/sh
# Sets up the Mac that works on ws, the server this repository sets up. Run it as yourself, from a
# clone of the repository, after writing ~/.config/ws/mac.conf:
#
#   ~/repository/ws/mac/setup.sh [--check]
#
# It reads mac.conf without running it, and writes ~/.ssh/config.d/ws, whose host ws-ssh reaches
# ws over the cable, the home LAN or the router's port forward, whichever the Mac is on. It puts
# `Include config.d/*` first in ~/.ssh/config. A run changes only what differs, so a second prints
# nothing. --check prints each change as a diff and makes none. README.md's The Mac has the rest.
#
# shellcheck source-path=SCRIPTDIR

set -eu

dir=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=lib.sh
. "$dir/lib.sh"

# ws's end of the cable.
cable_server=10.77.0.1

# ssh_config prints ~/.ssh/config.d/ws from mac.conf's values. ssh keeps the first value it reads
# for a key, so each Match sets Port 22 before Host ws-ssh sets the forward's port, and the cable
# comes before the LAN. localnetwork matches the Mac's own addresses, with no probe; originalhost
# is the name typed, as HostName changes the one Host and Match host see.
ssh_config() {
    cat <<EOF
# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain.
IgnoreUnknown UseKeychain

# ws over the cable, where the Mac has an address on it.
Match originalhost ws-ssh localnetwork $ws_cable_net
    HostName $cable_server
    Port 22
EOF
    if [ -n "$WS_LAN" ]; then
        cat <<EOF
# ws on the home LAN, where the Mac has an address there.
Match originalhost ws-ssh localnetwork $WS_LAN
    HostName $WS_LAN_ADDRESS
    Port 22
EOF
    fi
    if [ -n "$WS_REMOTE_PORT" ]; then
        cat <<EOF
# Elsewhere, through the router's port forward, which takes IPv4 alone, while the router's name
# may have an AAAA record too.
Host ws-ssh
    HostName $WS_REMOTE
    Port $WS_REMOTE_PORT
    AddressFamily inet
EOF
    fi
    cat <<'EOF'

# One known_hosts entry, ws's, whichever way ssh goes. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host ws-ssh
    User root
    HostKeyAlias ws
    IdentityFile ~/.ssh/id_ed25519
    IdentitiesOnly yes
    UseKeychain yes
    AddKeysToAgent yes
    ServerAliveInterval 30
EOF
}

# ssh_files writes ~/.ssh/config.d/ws, once ssh has read it alone, and puts the Include of
# config.d first in ~/.ssh/config.
ssh_files() {
    ws_tmp
    new=$ws_tmpfile
    ssh_config >"$new"
    if ! err=$(ssh -G -F "$new" ws-ssh 2>&1 >/dev/null); then
        rm -f "$new"
        ws_die "ssh refuses the config it would write, as $err; Match localnetwork needs OpenSSH 9.4"
    fi
    [ -d "$HOME/.ssh" ] || ws_dir "$HOME/.ssh" 700
    ws_dir "$HOME/.ssh/config.d" 700
    ws_put "$HOME/.ssh/config.d/ws" 600 <"$new"
    rm -f "$new"
    ws_include_first "$HOME/.ssh/config" 'Include config.d/*'
    ws_own_hosts "$HOME/.ssh/config" ws ws-ssh
}

main() {
    ws_args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not as root"
    ws_conf "$HOME/.config/ws/mac.conf"
    ssh_files
}

main "$@"
