#!/bin/bash
# Tries that update-checks.sh, which runs as root and deletes /usr/local/bin/ws as it ends, deletes
# nothing where a guard refuses it. It runs a copy whose /usr/local/bin is a scratch directory, with
# no CI variable and no spare machine's, so that a guard refuses it. CI's lint job runs it.
set -euo pipefail

script=$(cd "$(dirname "$0")" && pwd)/update-checks.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/bin" "$tmp/fake"
sed "s#/usr/local/bin#$tmp/bin#g" "$script" >"$tmp/update-checks.sh"
# The copy must reach no file of the machine, and a stand-in sudo runs its command unchanged.
if grep -q /usr/local/bin "$tmp/update-checks.sh"; then
    echo "update-checks-test: the copy names /usr/local/bin" >&2
    exit 1
fi
printf '#!/bin/sh\nexec "$@"\n' >"$tmp/fake/sudo"
chmod 755 "$tmp/fake/sudo"
echo stand-in >"$tmp/bin/ws"
echo stand-in >"$tmp/bin/.ws.1"

rc=0
(cd "$tmp" && env -u GITHUB_ACTIONS -u WS_SPARE_MACHINE PATH="$tmp/fake:$PATH" bash update-checks.sh) >"$tmp/out" 2>&1 || rc=$?
if [ "$rc" = 0 ]; then
    echo "update-checks-test: a run with no runner and no spare machine went through" >&2
    exit 1
fi
if [ ! -f "$tmp/bin/ws" ] || [ ! -f "$tmp/bin/.ws.1" ]; then
    echo "update-checks-test: a refused run deleted the machine's ws" >&2
    cat "$tmp/out" >&2
    exit 1
fi
echo "update-checks-test: ok"
