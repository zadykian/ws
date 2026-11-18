#!/bin/bash
# mise bootstrap with the Mac's frame, as the runbook runs it, beside ws client macos apply. The
# first run installs ws and links the plugin, and copies ws to /usr/local/bin. Then the plan must
# exit 0, and a second run must call sudo not once, as a run that changes nothing asks no password.
# A spec whose unit no line names, and that has a change, must fail the plan, which ws unit read
# --whole sees. One in sync passes, as mise reads again without the lines in sync.

set -eo pipefail
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

mac_mise
mac_tree
trap 'kill "$mac_server"' EXIT
mac_mark
mac_bootstrap || {
    echo '::error::the first mise bootstrap failed'
    exit 1
}
echo "the first run called sudo $(mac_calls) times"
fail=0
mac_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan after the first run exited $?, not 0"
    fail=1
}
mac_mark
mac_bootstrap || {
    echo '::error::the second mise bootstrap failed'
    exit 1
}
calls=$(mac_calls)
if [ "$calls" != 0 ]; then
    echo "::error::the second mise bootstrap called sudo $calls times, not once"
    fail=1
fi
cmp -s /usr/local/bin/ws "$(mac_build)" || {
    echo '::error::/usr/local/bin/ws is not the build'
    fail=1
}
owner=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/ws)
[ "$owner" = "root:wheel 755" ] || {
    echo "::error::/usr/local/bin/ws is $owner, not root:wheel 755"
    fail=1
}
# A spec of a unit that no line names, beside the frame's, with a change: its read fails each line.
probe=/opt/ws/roles/client/macos/probe
sudo mkdir -p "$probe"
printf '[state.probe.run]\nuser = "root"\ncommand = ["true"]\ncheck = ["false"]\n' | sudo tee "$probe/units.toml" >/dev/null
status=0
mac_bootstrap plan --detailed-exitcode 2>&1 | tee "$RUNNER_TEMP/plan.out" || status=$?
sudo rm -rf "$probe"
if [ "$status" != 1 ] || ! grep -q "lacks ws:state/probe" "$RUNNER_TEMP/plan.out"; then
    echo "::error::the plan with a unit that no line names exited $status, not 1 with its name"
    fail=1
fi
exit "$fail"
