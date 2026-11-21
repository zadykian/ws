#!/bin/bash
# The Mac converged, after every case before it: one more mise bootstrap must change nothing and
# call sudo not once, as a run that changes nothing asks no password. Its plan must exit 0. So must
# mise dot status --missing: the Mac's config holds no dotfile of mise's, as ws's units hold the
# user's files.

set -eo pipefail
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

fail=0
mac_mark
mac_bootstrap || {
    echo '::error::the last mise bootstrap failed'
    exit 1
}
calls=$(mac_calls)
if [ "$calls" != 0 ]; then
    echo "::error::the last mise bootstrap called sudo $calls times, not once"
    fail=1
fi
mac_bootstrap plan --detailed-exitcode || {
    echo "::error::the last plan exited $?, not 0"
    fail=1
}
mac_tree_mise dot status --missing || {
    echo "::error::mise dot status --missing exited $?, not 0"
    fail=1
}
exit "$fail"
