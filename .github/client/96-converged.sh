#!/bin/bash
# The Mac converged, after every case before it: one more mise bootstrap must change nothing and
# call sudo not once, as a run that changes nothing asks no password. Its plan must exit 0. So must
# mise dot status --missing: each of the roles' [dotfiles] entries stands in the user's home.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

fail=0
client_mark
client_bootstrap || {
    echo '::error::the last mise bootstrap failed'
    exit 1
}
calls=$(client_calls)
if [ "$calls" != 0 ]; then
    echo "::error::the last mise bootstrap called sudo $calls times, not once"
    fail=1
fi
client_bootstrap plan --detailed-exitcode || {
    echo "::error::the last plan exited $?, not 0"
    fail=1
}
client_tree_mise dot status --missing || {
    echo "::error::mise dot status --missing exited $?, not 0"
    fail=1
}
exit "$fail"
