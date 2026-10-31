#!/bin/sh
# The mac-cld job's checks of the lines that start cld's watcher, and of a quiet second run
# (.github/workflows/mac-cld.yml). Run from the repository's root:
#
#   sh .github/mac-cld-checks.sh lines yes|no
#   sh .github/mac-cld-checks.sh watch LOG
#   sh .github/mac-cld-checks.sh quiet
#
# lines: ssh -G gives ws and ws-ssh the two lines of ws setup ssh, with yes, or neither, with no. watch:
# ssh -G ws's LocalCommand, run by zsh and by sh as ssh runs it, must start the watcher in a
# Ghostty pane alone. The job's stub cld writes a line to LOG at each start. Elsewhere, and without
# cld, the line must start none and exit 0. quiet: mac/setup.sh, and its --check, must print
# nothing. Each failure is an ::error::, and the status 1.

set -u

fail=0
# The LocalCommand that ws setup ssh writes, as ssh -G prints it.
# shellcheck disable=SC2016 # the line's $TERM_PROGRAM is for the shell that ssh runs
want='[ "$TERM_PROGRAM" != ghostty ] || [ ! -x %d/.local/bin/cld ] ||'
want="$want %d/.local/bin/cld restore tabs --watch"

# error MESSAGE... notes a failure.
error() {
    echo "::error::$*"
    fail=1
}

# lines yes|no checks ssh -G's PermitLocalCommand and LocalCommand for ws and ws-ssh. An ssh that
# printed %d expanded would pass too.
lines() {
    expanded=$(printf '%s\n' "$want" | sed "s|%d|$HOME|g")
    for host in ws ws-ssh; do
        config=$(ssh -G "$host") || {
            error "ssh -G $host failed"
            continue
        }
        permit=$(printf '%s\n' "$config" | awk '$1 == "permitlocalcommand" { print $2 }')
        command=$(printf '%s\n' "$config" | sed -n 's/^localcommand //p')
        if [ "$1" = yes ]; then
            [ "$permit" = yes ] || error "ssh -G $host gives permitlocalcommand '$permit', not yes"
            [ "$command" = "$want" ] || [ "$command" = "$expanded" ] ||
                error "ssh -G $host gives localcommand '$command', not '$want'"
        else
            [ "$permit" = no ] || error "ssh -G $host gives permitlocalcommand '$permit', not no"
            [ -z "$command" ] || error "ssh -G $host gives localcommand '$command', not none"
        fi
    done
}

# try SHELL TERM_PROGRAM COMMAND runs COMMAND as ssh runs a LocalCommand, SHELL -c COMMAND, with
# TERM_PROGRAM set, or unset where it is empty, and notes a status other than 0.
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
    command=$(ssh -G ws | sed -n 's/^localcommand //p')
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

# quiet runs mac/setup.sh, then mac/setup.sh --check, each of which must print nothing.
quiet() {
    for args in '' --check; do
        if ! out=$(sh mac/setup.sh ${args:+"$args"} 2>&1); then
            error "setup.sh${args:+ $args} failed:"
        elif [ -n "$out" ]; then
            error "setup.sh${args:+ $args} printed what it should not:"
        else
            continue
        fi
        printf '%s\n' "$out"
    done
}

case ${1-} in
lines) lines "${2-}" ;;
watch) watch "${2:?usage: watch LOG}" ;;
quiet) quiet ;;
*) error "usage: sh .github/mac-cld-checks.sh lines yes|no | watch LOG | quiet" ;;
esac
exit "$fail"
