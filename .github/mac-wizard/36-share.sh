#!/bin/sh
# The wizard's calls to ws share one connection. A run that finds every stage done, and --check,
# read ws at Root's key and at ws's run, so each makes two sessions at least. They must all come in
# by one connection of the stand-in's sshd, whose ws.sh logs each session's in conns. Once the run
# ends, the stand-in must have no connection open, so that no master of ssh lives on, and the lock
# must be gone with its sockets.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
ws=$RUNNER_TEMP/ws
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# connected prints how many ends of connections to the stand-in's port netstat lists.
connected() {
    netstat -an -p tcp | grep -c '\.22022 .*ESTABLISHED'
}

for args in '' --check; do
    before=$(wc -l <"$ws/conns")
    sh "$wizard" ${args:+"$args"} </dev/null
    status=$?
    new=$(awk -v n="$before" 'NR > n' "$ws/conns")
    sessions=$(printf '%s\n' "$new" | grep -c .)
    links=$(printf '%s\n' "$new" | sort -u | grep -c .)
    [ "$status" = 0 ] || error "wizard.sh${args:+ $args} exited $status"
    if [ "$sessions" -lt 2 ] || [ "$links" != 1 ]; then
        error "wizard.sh${args:+ $args} made $sessions sessions to ws in $links connections"
    fi
    tries=0
    until [ "$(connected)" = 0 ] || [ "$tries" = 10 ]; do
        sleep 1
        tries=$((tries + 1))
    done
    [ "$(connected)" = 0 ] || error "a connection to ws stayed open after wizard.sh${args:+ $args}"
    [ ! -e "$tmp/ws-wizard.lock" ] || error "wizard.sh${args:+ $args} left $tmp/ws-wizard.lock"
done
exit "$fail"
