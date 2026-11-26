#!/bin/bash
# Takes from the runner's image what setup makes on a clean Mac, so that cases 83 and 88 read its
# work. Homebrew's gnupg goes, its dependencies kept, for mise to pour. The display's timer on power
# becomes 10, for power's unit to set to 0.

set -eo pipefail

fail=0
if brew list --formula gnupg >/dev/null 2>&1; then
    HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --ignore-dependencies gnupg
fi
[ ! -e /opt/homebrew/bin/gpg ] || {
    echo '::error::/opt/homebrew/bin/gpg stands with no gnupg'
    fail=1
}
sudo pmset -c displaysleep 10
pmset -g custom | awk '/^AC Power:/ {ac = 1; next} /^[^ ]/ {ac = 0} ac && $1 == "displaysleep" && $2 == 10 {found = 1} END {exit !found}' ||
    {
        echo '::error::pmset -g custom does not show displaysleep 10 under AC Power'
        fail=1
    }
exit "$fail"
