#!/bin/bash
# Fails where a bootstrap or a plan prints a warning, as no run of ws's may. The server
# job runs it on each of its own (.github/workflows/README.md#the-warnings-check):
#
#   warnings-check.sh LOG                checks the output that a run wrote to LOG
#   warnings-check.sh -- COMMAND [ARG]   runs COMMAND, its output shown as it runs, and checks it
#
# A warning is a line that says one of the words below, or starts as apt's W: and E: do. The check
# prints each, past mise's colors, and exits 1. It allows no line. The second form exits with
# COMMAND's status where it prints none. With one, a status of 0 or 2, a plan's answer, becomes 1,
# and a failure keeps its own.
set -uo pipefail

words='warn|warns|warning|warnings|hint|hints|notice|notices|deprecated|deprecation|error|errors'
words="$words|cannot|unable|could not|ignoring"
# A word counts alone, not in a name such as liberror-perl or a path: neither the character before
# it nor the one after it can be a name's.
pattern="(^|[^[:alnum:]_./-])($words)([^[:alnum:]_-]|\$)|^[[:space:]]*[WE]: "

# found LOG prints each warning of LOG.
found() {
    tr '\r' '\n' <"$1" | sed -E $'s/\e\\[[0-9;?]*[A-Za-z]//g' | grep -iE -- "$pattern"
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
    case $status in 0 | 2) status=1 ;; esac
fi
exit "$status"
