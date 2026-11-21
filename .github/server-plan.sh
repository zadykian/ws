#!/bin/sh
# The server job's two plans that must fail, as root, on the tree in /opt/ws. A plan from /opt/ws,
# which loads no role, must fail and name the server's run directory. Then the check against a
# silent skip must fail the plan where mise's config lacks a unit. This drops the frame's line of
# state/ws-binary, and puts the file back after. ws unit verify names the unit, or the plugin that
# mise no longer loads where the line was the last of ws's. mise's plan exits 2 where a part or a unit would change,
# and 1 where it cannot tell.
set -eu

out=$(mise -C /opt/ws bootstrap plan </dev/null 2>&1) && status=0 || status=$?
if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF 'run it as mise -C /opt/ws/roles/server'; then
    printf '%s\n' "$out"
    echo "::error::The plan from /opt/ws exited $status, not 1 naming the server's run directory."
    exit 1
fi

toml=/opt/ws/roles/server/ubuntu/frame/mise.server.toml
line='"ws:state/ws-binary" = "in-sync"'
saved=$(mktemp)
out=$(mktemp)
cp -p "$toml" "$saved"
trap 'cp -p "$saved" "$toml"; rm -f "$saved" "$out"' EXIT
grep -qxF "$line" "$saved" || {
    echo "::error::$toml lacks $line, which this check drops."
    exit 1
}
grep -vxF "$line" "$saved" >"$toml"
status=0
mise -C /opt/ws/roles/server bootstrap plan --detailed-exitcode </dev/null >"$out" 2>&1 || status=$?
cat "$out"
if [ "$status" != 1 ] || ! grep -q '^ws unit verify: ' "$out"; then
    echo "::error::The plan exited $status, not 1 by ws unit verify, with a unit's line dropped."
    exit 1
fi
