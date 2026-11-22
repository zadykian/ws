#!/bin/bash
# Tries warnings-check.sh on logs of lines that CI's runs printed. It must pass the clean ones,
# with mise's colors and with names that hold a word. It must fail the warnings, and name each.
# CI's lint job runs it.
set -euo pipefail

check=$(cd "$(dirname "$0")" && pwd)/warnings-check.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
failed=0
e=$'\e'

cat >"$tmp/clean" <<EOF
Setting up liberror-perl (0.17030-1) ...
Setting up libgpg-error0:amd64 (1.55-2) ...
Unpacking libgpg-error-l10n (1.55-2) ...
Get:1 http://archive.ubuntu.com/ubuntu resolute/main amd64 liberror-perl all 0.17030-1 [25.6 kB]
${e}[2mmise${e}[0m bootstrap: package plugins
unchanged  package:ws:file/docker-resolved   installed (in-sync)  installed (in-sync)
ws unit write: service/base-sysctl: off on this machine, as its container is "yes"
invoke-rc.d: policy-rc.d denied execution of start.
Plan: 0 create, 0 update, 83 unchanged, 0 remove, 0 run, 0 unknown
EOF

# One line of each kind that the jobs' logs held, mise's in its colors.
cat >"$tmp/warnings" <<EOF
${e}[33mmise${e}[0m ${e}[33mWARN${e}[0m  Plugin ws already installed
mise WARN  plugin ws in /usr/local/share/mise/plugins/ws overrides the system plugin
npm notice New major version of npm available! 11.19.1 -> 12.2.3
dpkg: warning: while removing docker-ce, directory '/usr/libexec/docker' not empty so not removed
update-alternatives: warning: skip creation of /usr/share/man/man1/lzma.1.gz
sysctl: setting key "kernel.sysrq", ignoring: Read-only file system
level=warning msg="WARNING: No cpuset support"
hint: Using 'master' as the name for the initial branch.
Unable to find image 'alpine:3.24.3' locally
W: Some index files failed to download.
E: Unable to locate package nonesuch
DEPRECATION: the key is deprecated
::warning::/usr/local/bin was runner:admin 775
mise ERROR runtime error: state/ws-binary fails
mv: cannot move '/etc/resolv.conf': Permission denied
dpkg-statoverride: warning: --update given but /var/log/chrony does not exist
EOF

# expect NAME STATUS LOG... runs the check on the LOGs joined, and wants STATUS. The lines it names must be $want's, each once and in their order.
expect() {
    local name=$1 status=$2 got=0 out
    shift 2
    cat "$@" >"$tmp/log"
    out=$("$check" "$tmp/log") || got=$?
    if [ "$got" = "$status" ] && [ "$(sed -n 's/^> //p' <<<"$out")" = "$want" ]; then
        echo "ok: $name"
    else
        printf 'FAIL: %s: exit %s, not %s, and it printed:\n%s\n' "$name" "$got" "$status" "$out"
        failed=1
    fi
}

want=
expect "a clean log passes" 0 "$tmp/clean"
want=$(sed -E $'s/\e\\[[0-9;]*m//g' "$tmp/warnings")
expect "each warning fails, and is named" 1 "$tmp/clean" "$tmp/warnings"

# run NAME STATUS OUT COMMAND... runs the second form on COMMAND, and wants STATUS and OUT: the
# command's output, its stderr's included, then what the check named.
run() {
    local name=$1 status=$2 want=$3 got=0 out
    shift 3
    out=$("$check" -- "$@") || got=$?
    if [ "$got" = "$status" ] && [ "$out" = "$want" ]; then
        echo "ok: $name"
    else
        printf 'FAIL: %s: exit %s, not %s, and it printed:\n%s\n' "$name" "$got" "$status" "$out"
        failed=1
    fi
}
run "a command's status and output stay" 3 shown sh -c 'echo shown; exit 3'
named='::error::The run printed 1 warning lines, which no run may print:'
run "a command's warning on its stderr fails it" 1 $'W: a warning\n'"$named"$'\n> W: a warning' \
    sh -c 'echo "W: a warning" >&2'
# A plan's exit 2 is its answer, not a failure, so a warning fails it too. A failure keeps its own.
run "a plan's warning fails its exit 2" 1 $'W: a warning\n'"$named"$'\n> W: a warning' \
    sh -c 'echo "W: a warning"; exit 2'
run "a failure's warning keeps its status" 3 $'W: a warning\n'"$named"$'\n> W: a warning' \
    sh -c 'echo "W: a warning"; exit 3'
exit "$failed"
