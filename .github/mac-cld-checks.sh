#!/bin/sh
# The mac-cld job's checks of the lines that start cld's watcher, and of a quiet second run
# (.github/workflows/mac-cld.yml). Run from the repository's root:
#
#   sh .github/mac-cld-checks.sh lines yes|no
#   sh .github/mac-cld-checks.sh quiet
#
# lines: ssh -G gives ws and ws-ssh setup-cld.sh's two lines, with yes, or neither, with no. quiet:
# mac/setup.sh, and its --check, must print nothing. Each failure is an ::error::, and the status
# 1.

set -u

fail=0
# The LocalCommand that setup-cld.sh writes, as ssh -G prints it.
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
quiet) quiet ;;
*) error "usage: sh .github/mac-cld-checks.sh lines yes|no | quiet" ;;
esac
exit "$fail"
