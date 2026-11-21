#!/bin/bash
# The mac-cld job's checks (.github/workflows/mac-cld.yml), run from the repository's root as
# bash .github/mac-cld-checks.sh VERB [ARG]. Each failure is an ::error::, and the status 1.
#
# - setup puts this commit's tree in /opt/ws, as the runbook's clone does (mac-mise.sh).
# - bootstrap [N] runs ws client setup N times, once by default, with ws's build served.
# - installed checks what the bootstrap set up: Ghostty, Zed and the font, as mise's casks, cld,
#   and the files of Ghostty and Zed.
# - lines yes|no: ssh -G gives ws the two lines that start cld's watcher with yes, and neither
#   with no.
# - watch LOG: ssh -G ws's LocalCommand, run by zsh and by sh as ssh runs it, must start the
#   watcher in a Ghostty pane alone. The job's stub cld writes a line to LOG at each start.
#   Elsewhere, and without cld, the line must start none and exit 0.
# - ssh writes ssh's file of hosts alone, by ws unit write, which installs no cask.
# - quiet: a setup must call sudo not once, and its plan exit 0.

set -u
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

fail=0
# The LocalCommand that ssh's unit writes, as ssh -G prints it.
# shellcheck disable=SC2016 # the line's $TERM_PROGRAM is for the shell that ssh runs
want='[ "$TERM_PROGRAM" != ghostty ] || [ ! -x %d/.local/bin/cld ] ||'
want="$want %d/.local/bin/cld restore tabs --watch"
mark='Written by github.com/zadykian/ws, whose next run replaces any change made here.'

# error MESSAGE... notes a failure.
error() {
    echo "::error::$*"
    fail=1
}

# served COMMAND... runs COMMAND with ws's build served, as a bootstrap fetches the pin, and returns
# its status.
served() {
    mac_serve || return 1
    "$@"
    code=$?
    kill "$mac_server"
    return "$code"
}

# setup puts this commit's tree in /opt/ws.
setup() {
    mac_tree || exit 1
    kill "$mac_server"
}

# bootstrap N runs ws client setup N times, and stops at the first that fails.
bootstrap() {
    for run in $(seq "$1"); do
        served mac_bootstrap || {
            error "ws client setup failed, at run $run of $1"
            return
        }
    done
}

# installed checks the apps, the casks, the font, cld and the files of Ghostty and Zed.
installed() {
    for app in Ghostty Zed; do
        [ -d "/Applications/$app.app" ] || error "mise bootstrap installed no /Applications/$app.app"
    done
    status=$(mac_tree_mise bootstrap packages status --json) ||
        error 'mise bootstrap packages status --json failed'
    for cask in ghostty zed font-jetbrains-mono-nerd-font; do
        # The status names each package and its manager. mise's casks need no Homebrew.
        printf '%s\n' "$status" | jq -e --arg c "$cask" '[.. | objects |
            select(any(.[]; . == $c or . == "brew-cask:" + $c)) | .managed_by] | any(. == "mise")' \
            >/dev/null || error "mise's packages status names no cask $cask that mise manages"
    done
    [ "$fail" = 0 ] || printf '%s\n' "$status"
    # A login shell's PATH holds /etc/paths.d/ws, which names the casks' bin.
    zedv=$(env -i HOME="$HOME" zsh -lc 'zed --version') || error 'zed --version failed in a login shell'
    echo "$zedv"
    printf '%s\n' "$zedv" | grep -Eq '^Zed [0-9.]+ – /Applications/Zed\.app$' ||
        error 'zed --version is not Zed N – /Applications/Zed.app'
    [ -f ~/Library/Fonts/"${FONT:?}" ] || error "mise bootstrap installed no $FONT"
    ~/.local/bin/cld version || error "$HOME/.local/bin/cld version failed"
    wsghostty=roles/client/macos/ghostty/files/ws.ghostty
    cmp -s /etc/ws/ghostty "$wsghostty" || error "/etc/ws/ghostty is not $wsghostty"
    support="$HOME/Library/Application Support/com.mitchellh.ghostty"
    [ "$(grep -cx 'config-file = ?/etc/ws/ghostty' "$support/config.ghostty")" = 1 ] ||
        error 'config.ghostty does not read /etc/ws/ghostty once'
    plain=roles/client/macos/zed/files
    zed=~/.config/zed
    [ "$(head -n 1 "$zed/global_settings.json")" = "// $mark" ] ||
        error 'global_settings.json has no ws header'
    cmp -s "$zed/global_settings.json" "$plain/global_settings.json" ||
        error "global_settings.json differs from the one in $plain"
    grep -qxF '{ "ssh_connections": [{ "host": "ws" }] }' "$zed/settings.json" ||
        error 'settings.json has no seed for the host ws'
    cmp -s "$zed/keymap.json" "$plain/keymap.json" || error "keymap.json is not $plain/keymap.json"
    # `+validate-config` reads the config as the app does, /etc/ws/ghostty and its theme included, and
    # names each line it refuses. It leaves the fonts alone, and a family that names no font falls
    # back to Ghostty's own JetBrains Mono, which looks the same. So +list-fonts, which looks a
    # family up as the config does, must find the last one of /etc/ws/ghostty.
    ghostty=/Applications/Ghostty.app/Contents/MacOS/ghostty
    "$ghostty" +validate-config || error 'Ghostty refuses its config, above'
    family=$(sed -n 's/^font-family = \(..*\)$/\1/p' "$wsghostty" | tail -n 1)
    fonts=$("$ghostty" +list-fonts --family="$family") ||
        error "ghostty +list-fonts --family='$family' failed"
    printf '%s\n' "$fonts"
    [ -n "$fonts" ] || error "Ghostty finds no font of the family '$family'"
    # A release of cld counts as one with restore tabs, as its unit keeps it at the latest.
    lines yes
}

# lines yes|no checks ssh -G's PermitLocalCommand and LocalCommand for ws. An ssh that printed %d
# expanded would pass too.
lines() {
    expanded=$(printf '%s\n' "$want" | sed "s|%d|$HOME|g")
    config=$(ssh -T -G ws) || {
        error "ssh -G ws failed"
        return
    }
    permit=$(printf '%s\n' "$config" | awk '$1 == "permitlocalcommand" { print $2 }')
    command=$(printf '%s\n' "$config" | sed -n 's/^localcommand //p')
    if [ "$1" = yes ]; then
        [ "$permit" = yes ] || error "ssh -G ws gives permitlocalcommand '$permit', not yes"
        [ "$command" = "$want" ] || [ "$command" = "$expanded" ] ||
            error "ssh -G ws gives localcommand '$command', not '$want'"
    else
        [ "$permit" = no ] || error "ssh -G ws gives permitlocalcommand '$permit', not no"
        [ -z "$command" ] || error "ssh -G ws gives localcommand '$command', not none"
    fi
}

# try SHELL TERM_PROGRAM COMMAND runs COMMAND as ssh runs a LocalCommand, SHELL -c COMMAND, with
# TERM_PROGRAM set, or unset where empty, and notes a status other than 0.
try() {
    if [ -n "$2" ]; then
        TERM_PROGRAM=$2 "$1" -c "$3"
    else
        env -u TERM_PROGRAM "$1" -c "$3"
    fi
    status=$?
    [ "$status" = 0 ] || error "$1 -c, under TERM_PROGRAM '$2', exits $status: $3"
}

# watch LOG runs ssh -G ws's LocalCommand with %d as the home, and as a home without cld.
watch() {
    command=$(ssh -T -G ws | sed -n 's/^localcommand //p')
    if [ -z "$command" ]; then
        error "ssh -G ws gives no localcommand"
        return
    fi
    empty=$(mktemp -d) || exit 1
    home=$(printf '%s\n' "$command" | sed "s|%d|$HOME|g")
    none=$(printf '%s\n' "$command" | sed "s|%d|$empty|g")
    for shell in zsh sh; do
        : >"$1"
        for term in '' tmux Apple_Terminal; do
            try "$shell" "$term" "$home"
        done
        try "$shell" ghostty "$none"
        [ ! -s "$1" ] || error "$shell: the line started the watcher outside a Ghostty pane"
        try "$shell" ghostty "$home"
        [ "$(cat "$1")" = ghostty ] ||
            error "$shell: in a Ghostty pane, the stub logged '$(cat "$1")', not one start"
    done
    rmdir "$empty"
}

# ssh writes /etc/ws/ssh_config by ws unit write, as the user, from the tree, as the plugin does.
ssh_hosts() {
    (cd /opt/ws && /usr/local/bin/ws unit write file/ssh-config) ||
        error 'ws unit write file/ssh-config failed'
}

# quiet runs setup, which must call sudo not once, then its plan, which must exit 0.
quiet() {
    mac_mark
    served mac_bootstrap || {
        error 'ws client setup failed'
        return
    }
    calls=$(mac_calls)
    [ "$calls" = 0 ] || error "ws client setup called sudo $calls times, not once"
    served mac_bootstrap plan --detailed-exitcode || error "ws client setup's plan exited $?, not 0"
}

case ${1-} in
setup) setup ;;
bootstrap) bootstrap "${2-1}" ;;
installed) installed ;;
lines) lines "${2-}" ;;
watch) watch "${2:?usage: watch LOG}" ;;
ssh) ssh_hosts ;;
quiet) quiet ;;
*) error "usage: bash .github/mac-cld-checks.sh setup | bootstrap [N] | installed | lines yes|no | watch LOG | ssh | quiet" ;;
esac
exit "$fail"
