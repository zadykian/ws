#!/bin/sh
# Sets up the Mac that works on ws, the server this repository sets up. Run it as yourself, from a
# clone of the repository, after writing ~/.config/ws/mac.conf:
#
#   ~/repository/ws/mac/setup.sh [--check]
#
# It reads mac.conf without running it, and writes ~/.ssh/config.d/ws. Its host ws reaches ws over
# the cable, the home LAN or the router's port forward, the first of these the Mac is on. It puts
# `Include config.d/ws` first in ~/.ssh/config. A run changes only what differs, so a second prints
# nothing. --check prints each change as a diff and makes none. README.md's The Mac has the rest.
#
# shellcheck source-path=SCRIPTDIR

set -eu

dir=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=lib.sh
. "$dir/lib.sh"

# ws's end of the cable.
cable_server=10.77.0.1
# The start of config.d/ws's first line, by which a run knows the file for the script's own.
header="# Written by ws's mac/setup.sh"

# ssh_config prints ~/.ssh/config.d/ws from mac.conf's values and the IgnoreUnknown patterns $1.
# ssh keeps the first value it reads for a key, so each Match sets Port 22 before Host ws sets the
# forward's port, and the cable comes before the LAN. localnetwork matches the Mac's own
# addresses, with no probe. originalhost matches the name typed, which a HostName set above does
# not change, unlike Match host.
ssh_config() {
    cat <<EOF
$header from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain. ssh keeps the first IgnoreUnknown it reads, this one, so it
# carries the patterns of ~/.ssh/config's own too: run setup.sh again after a change to those.
IgnoreUnknown $1

# ws over the cable, where the Mac has an address on it.
Match originalhost ws localnetwork $ws_cable_net
    HostName $cable_server
    Port 22
EOF
    if [ -n "$WS_LAN" ]; then
        cat <<EOF
# ws on the home LAN, where the Mac has an address there.
Match originalhost ws localnetwork $WS_LAN
    HostName $WS_LAN_ADDRESS
    Port 22
EOF
    fi
    if [ -n "$WS_REMOTE_PORT" ]; then
        cat <<EOF
# Elsewhere, through the router's port forward, which takes IPv4 alone, while the router's name
# may have an AAAA record too.
Host ws
    HostName $WS_REMOTE
    Port $WS_REMOTE_PORT
    AddressFamily inet
EOF
    fi
    cat <<EOF

# One known_hosts entry, ws's, whichever way ssh goes. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host ws
    User $WS_USER
    HostKeyAlias ws
    IdentityFile ~/.ssh/id_ed25519
    IdentitiesOnly yes
    UseKeychain yes
    AddKeysToAgent yes
    ServerAliveInterval 30
EOF
}

# ssh_whole stops where ssh refuses the new config.d/ws, $1, read first, then ~/.ssh/config, then
# the system's file, as ssh will read them after the run. ~/.ssh/config's Include of config.d/ws,
# at line $2 where there is one, is left out, as it would read the file the run replaces.
ssh_whole() {
    user=$HOME/.ssh/config
    read_user=$user
    ws_tmp
    whole=$ws_tmpfile
    ws_tmp
    copy=$ws_tmpfile
    if [ -n "$2" ] && [ -f "$user" ]; then
        awk -v skip="$2" '{ print (NR == skip ? "" : $0) }' "$user" >"$copy" ||
            ws_die "cannot read $user"
        read_user=$copy
    fi
    # The system's file goes through an Include too, so ssh resolves its relative Includes under
    # ~/.ssh, and checks its mode, where ssh itself would not.
    {
        printf 'Include "%s"\n' "$1"
        [ ! -f "$user" ] || printf 'Match all\nInclude "%s"\n' "$read_user"
        [ ! -f /etc/ssh/ssh_config ] || printf 'Match all\nInclude /etc/ssh/ssh_config\n'
    } >"$whole" || ws_die "cannot write $whole"
    err=$(ssh -G -F "$whole" ws 2>&1 >/dev/null) || {
        rm -f "$whole" "$copy" "$1"
        err=$(printf '%s\n' "$err" | sed "s|$copy|$user|g")
        case $err in
        *"$HOME/.ssh/config.d/ws"*)
            ws_die "ssh refuses the old config.d/ws, which ~/.ssh/config also reads, as $err:" \
                "move it away, and run setup.sh again"
            ;;
        *'Bad configuration option'*)
            ws_die "ssh refuses ~/.ssh/config read after the new config.d/ws, as $err;" \
                "config.d/ws's IgnoreUnknown, read first, carries only ~/.ssh/config's own" \
                "patterns, so one for a file it includes must be there too"
            ;;
        esac
        ws_die "ssh refuses ~/.ssh/config read after the new config.d/ws, as $err"
    }
    rm -f "$whole" "$copy"
}

# ssh_files writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. Before it
# writes, it stops where config.d/ws is a file it did not write, where the Include could not go
# first, and where ssh refuses the new file, alone or as ssh_whole reads it.
ssh_files() {
    conf=$HOME/.ssh/config.d/ws
    if [ -e "$conf" ] || [ -L "$conf" ]; then
        first=$(head -n 1 "$conf" 2>/dev/null) || first=
        case $first in
        "$header"*) ;;
        *) ws_die "$conf is not mac/setup.sh's: move it away, as the script writes its own there" ;;
        esac
    fi
    at=$(ws_include_ok "$HOME/.ssh/config" config.d/ws) || exit 1
    patterns=$(ws_ignore_unknown "$HOME/.ssh/config" UseKeychain)
    ws_tmp
    new=$ws_tmpfile
    ssh_config "$patterns" >"$new"
    if ! err=$(ssh -G -F "$new" ws 2>&1 >/dev/null); then
        rm -f "$new"
        ws_die "ssh refuses the config it would write, as $err;" \
            "Match localnetwork needs OpenSSH 9.4"
    fi
    ssh_whole "$new" "${at#first }"
    [ -d "$HOME/.ssh" ] || ws_dir "$HOME/.ssh" 700
    ws_dir "$HOME/.ssh/config.d" 700
    if [ -L "$HOME/.ssh/config.d" ] && ! cmp -s "$conf" "$new" 2>/dev/null; then
        ws_say "warning: ~/.ssh/config.d is a link to $(readlink "$HOME/.ssh/config.d"):" \
            "config.d/ws, with mac.conf's addresses, goes there"
    fi
    ws_put "$conf" 600 <"$new"
    rm -f "$new"
    ws_include_first "$HOME/.ssh/config" config.d/ws
    ws_own_hosts "$HOME/.ssh/config" ws
}

main() {
    ws_args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not as root"
    ws_conf "$HOME/.config/ws/mac.conf"
    ssh_files
}

main "$@"
