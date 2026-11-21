#!/bin/bash
# Fails where a bootstrap or a plan prints a warning, as no run of ws's may. The server
# job runs it on each of its own (.github/workflows/README.md#the-warnings-check):
#
#   warnings-check.sh LOG                checks the output that a run wrote to LOG
#   warnings-check.sh -- COMMAND [ARG]   runs COMMAND, its output shown as it runs, and checks it
#
# The second form keeps COMMAND's status other than 0, as a plan's exit code tells its own
# failure. A warning is a line that says one of the words below, or starts as apt's W: and E: do.
# The check prints each, past mise's colors, and exits 1. A line that it allows, it names below,
# each with the reason it must stay.
set -uo pipefail

words='warn|warns|warning|warnings|hint|hints|notice|notices|deprecated|deprecation|error|errors'
words="$words|cannot|unable|could not|ignoring"
# A word counts alone, not in a name such as liberror-perl or a path: neither the character before
# it nor the one after it can be a name's.
pattern="(^|[^[:alnum:]_./-])($words)([^[:alnum:]_-]|\$)|^[[:space:]]*[WE]: "

# allowed prints the start of each line that a run may print, a line each. They hold where systemd
# has not booted, as in a container, and come from what a container lacks. The
# server job's runner has booted, so none holds there. BOOTED, yes or no, stands in for the
# probe in the test.
allowed() {
    case ${BOOTED:-} in
    yes) return 0 ;;
    no) ;;
    *) [ ! -d /run/systemd/system ] || return 0 ;;
    esac
    # A package's postinst asks the init for the runlevel, to start its daemon, and the container
    # runs no init. A booted machine's systemd answers. Only a running init stands in for it.
    echo 'invoke-rc.d: could not determine current runlevel'
    # Docker bind-mounts /etc/resolv.conf, which systemd-resolved's first install moves and links
    # to its stub. A server has systemd-resolved from its own install, with that link. No step
    # can replace a mount point in a container without CAP_SYS_ADMIN.
    echo "mv: cannot move '/etc/resolv.conf' to '/etc/.resolv.conf.systemd-resolved.bak': "
    echo 'Cannot take a backup of /etc/resolv.conf.'
    echo 'Cannot install symlink from /etc/resolv.conf to ../run/systemd/resolve/stub-resolv.conf'
}

# found LOG prints each warning of LOG that no line of allowed covers.
found() {
    local allow line entry covered
    allow=$(allowed)
    tr '\r' '\n' <"$1" | sed -E $'s/\e\\[[0-9;?]*[A-Za-z]//g' | grep -iE -- "$pattern" |
        while IFS= read -r line; do
            covered=false
            while IFS= read -r entry; do
                if [ -n "$entry" ] && [[ $line == "$entry"* ]]; then
                    covered=true
                fi
            done <<<"$allow"
            "$covered" || printf '%s\n' "$line"
        done
}

status=0
if [ "${1:-}" = -- ]; then
    shift
    log=$(mktemp)
    trap 'rm -f "$log"' EXIT
    "$@" 2>&1 | tee "$log"
    status=${PIPESTATUS[0]}
else
    log=${1:?usage: warnings-check.sh LOG, or warnings-check.sh -- COMMAND [ARG]...}
fi
warnings=$(found "$log")
if [ -n "$warnings" ]; then
    echo "::error::The run printed $(wc -l <<<"$warnings") warning lines, which no run may print:"
    printf '%s\n' "$warnings" | sed 's/^/> /'
    [ "$status" != 0 ] || status=1
fi
exit "$status"
