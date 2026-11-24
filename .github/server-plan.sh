#!/bin/sh
# The server job's three plans that must fail, as root, on the tree in /opt/ws. A plan from /opt/ws,
# which loads no role, must fail and name the server's run directory. Then a bad value in
# /etc/ws/values.yml must fail the plan before any write, naming its key, and the file goes back.
# mise's plan asks ws for its units' state before it renders a file, so ws unit read refuses the
# value there. Its rule is that of ws ubuntu guest values check, which /etc/ws/values.checked runs.
# Then the check against a silent skip must fail the plan where mise's config lacks a unit. This
# drops the frame's line of file/ws-values, and puts the file back after. ws unit verify names the
# unit, or the plugin that mise no longer loads where the line was the last of ws's. mise's plan
# exits 2 where a part or a unit would change, and 1 where it cannot tell.
set -eu

out=$(mise -C /opt/ws bootstrap plan </dev/null 2>&1) && status=0 || status=$?
if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF 'run it as mise -C /opt/ws/ubuntu/guest'; then
    printf '%s\n' "$out"
    echo "::error::The plan from /opt/ws exited $status, not 1 naming the server's run directory."
    exit 1
fi

vars=/etc/ws/values.yml
vars_saved=$(mktemp)
cp -p "$vars" "$vars_saved"
trap 'cp -p "$vars_saved" "$vars"; rm -f "$vars_saved"' EXIT
printf 'WS_BASE_SWAPPINESS: 400\n' >>"$vars"
status=0
out=$(mise -C /opt/ws/ubuntu/guest bootstrap plan --detailed-exitcode </dev/null 2>&1) || status=$?
cp -p "$vars_saved" "$vars"
if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -Eq 'ws (unit read|ubuntu guest values check): .*WS_BASE_SWAPPINESS is 400'; then
    printf '%s\n' "$out"
    echo "::error::The plan with a bad value exited $status, not 1 by ws naming WS_BASE_SWAPPINESS."
    exit 1
fi

toml=/opt/ws/ubuntu/roles/frame/mise.guest.toml
line='"ws:file/ws-values" = "in-sync"'
saved=$(mktemp)
out=$(mktemp)
cp -p "$toml" "$saved"
trap 'cp -p "$saved" "$toml"; cp -p "$vars_saved" "$vars"; rm -f "$saved" "$out" "$vars_saved"' EXIT
grep -qxF "$line" "$saved" || {
    echo "::error::$toml lacks $line, which this check drops."
    exit 1
}
grep -vxF "$line" "$saved" >"$toml"
status=0
mise -C /opt/ws/ubuntu/guest bootstrap plan --detailed-exitcode </dev/null >"$out" 2>&1 || status=$?
cat "$out"
if [ "$status" != 1 ] || ! grep -q '^ws unit verify: ' "$out"; then
    echo "::error::The plan exited $status, not 1 by ws unit verify, with a unit's line dropped."
    exit 1
fi
