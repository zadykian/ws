#!/bin/bash
# ws macos setup with the Mac's frame, as the runbook runs it, on the tree in /opt/ws as it stands.
# Its plan before the first run must exit 2. The first run installs mise and links the plugin, and
# runs the build in /usr/local/bin. It sets up ssh, git and cld from the job's
# values.yml, with the stand-ins of cld and the stand-in ws's tools (client-standins.sh). It makes the
# Mac's WireGuard key too (30-key.sh). Then the plan must exit 0, and a second run must call sudo not once, as a run
# that changes nothing asks no password.
# A spec whose unit no line names, and that has a change, must fail the plan, which ws unit read
# --whole sees. One in sync passes, as mise reads again without the lines in sync. A plan from
# /opt/ws, which loads no role, must fail too, with every unit in sync, by the frame's line at the
# root, and name the run directory. The first run turns the firewall on and has it block all incoming
# (macos/roles/firewall/README.md): both states must read as on. The later cases, whose
# stand-ins listen on loopback, then run under block-all, which shows that the firewall spares loopback.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

sh .github/client-standins.sh
client_tree
status=0
client_bootstrap plan --detailed-exitcode || status=$?
if [ "$status" != 2 ]; then
    echo "::error::the plan before the first run exited $status, not 2: setup has mise and the plugin to do"
    exit 1
fi
client_mark
client_bootstrap || {
    echo '::error::the first ws macos setup failed'
    exit 1
}
echo "setup's first run called sudo $(client_calls) times"
fail=0
client_bootstrap plan --detailed-exitcode || {
    echo "::error::the plan after the first run exited $?, not 0"
    fail=1
}
# The get options exit 0 in every state, so each text is read.
socketfilterfw=/usr/libexec/ApplicationFirewall/socketfilterfw
"$socketfilterfw" --getglobalstate | grep -Eq 'State = [12]' || {
    echo "::error::the firewall is off after the first run: $("$socketfilterfw" --getglobalstate)"
    fail=1
}
"$socketfilterfw" --getblockall | grep -q 'blocking all' || {
    echo "::error::the firewall does not block all incoming after the first run: $("$socketfilterfw" --getblockall)"
    fail=1
}
client_mark
client_bootstrap || {
    echo '::error::the second ws macos setup failed'
    exit 1
}
calls=$(client_calls)
if [ "$calls" != 0 ]; then
    echo "::error::the second ws macos setup called sudo $calls times, not once"
    fail=1
fi
owner=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/ws)
[ "$owner" = "root:wheel 755" ] || {
    echo "::error::/usr/local/bin/ws is $owner, not root:wheel 755"
    fail=1
}
# A plan from /opt/ws with every unit in sync: the read fails, as the run would skip each role.
root=0
(cd / && mise -C /opt/ws bootstrap plan </dev/null) 2>&1 |
    tee "$RUNNER_TEMP/root.out" || root=$?
if [ "$root" != 1 ] || ! grep -qF "run mise in /opt/ws/macos" "$RUNNER_TEMP/root.out"; then
    echo "::error::the plan from /opt/ws exited $root, not 1 naming /opt/ws/macos"
    fail=1
fi
# A spec of a unit that no line names, beside the frame's, with a change: its read fails each line.
probe=/opt/ws/macos/roles/probe
sudo mkdir -p "$probe"
printf '[state.probe.run]\nuser = "root"\ncommand = ["true"]\ncheck = ["false"]\n' | sudo tee "$probe/units.toml.tmpl" >/dev/null
status=0
client_bootstrap plan --detailed-exitcode 2>&1 | tee "$RUNNER_TEMP/plan.out" || status=$?
sudo rm -rf "$probe"
if [ "$status" != 1 ] || ! grep -q "lacks ws:state/probe" "$RUNNER_TEMP/plan.out"; then
    echo "::error::the plan with a unit that no line names exited $status, not 1 with its name"
    fail=1
fi
exit "$fail"
