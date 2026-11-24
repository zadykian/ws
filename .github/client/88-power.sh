#!/bin/bash
# mise bootstrap (28-bootstrap.sh) ran macos/roles/power's units: the display stays on while the
# Mac is on power, and the App Store's automatic updates are off. Both read as the user.

set -eo pipefail

fail=0
pmset -g custom | awk '/^AC Power:/ {ac = 1; next} /^[^ ]/ {ac = 0} ac && $1 == "displaysleep" && $2 == 0 {found = 1} END {exit !found}' ||
    {
        echo '::error::pmset -g custom does not show displaysleep 0 under AC Power'
        fail=1
    }
[ "$(defaults read /Library/Preferences/com.apple.commerce AutoUpdate)" = 0 ] ||
    {
        echo "::error::the App Store's AutoUpdate is not 0"
        fail=1
    }
exit "$fail"
