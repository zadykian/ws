#!/bin/bash
# setup's plan asks mise's status for what mise's plan leaves out, such as a preference. One of
# desktop's, written another value, must make ws macos setup --plan exit 2 and name it as differs.
# Back at the role's value, by its type, it must exit 0. It goes back even on a failure.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

# A boolean that macos/roles/desktop/mise.macos.toml sets to false.
domain=NSGlobalDomain
key=AppleEnableSwipeNavigateWithScrolls
trap 'defaults write "$domain" "$key" -bool false' EXIT

fail=0
defaults write "$domain" "$key" -bool true
status=0
client_bootstrap plan --detailed-exitcode 2>&1 | tee "$RUNNER_TEMP/status.out" || status=$?
if [ "$status" != 2 ] || ! grep -F "$key" "$RUNNER_TEMP/status.out" | grep -q differs; then
    echo "::error::the plan with $key true exited $status, not 2 with its row of mise's status as differs"
    fail=1
fi
defaults write "$domain" "$key" -bool false
client_bootstrap plan --detailed-exitcode 2>&1 || {
    echo "::error::the plan with $key back at false exited $?, not 0"
    fail=1
}
exit "$fail"
