#!/bin/bash
# Tries warnings-check.sh on logs of lines that CI's runs printed. It must pass the clean ones,
# with mise's colors and with names that hold a word. It must fail the warnings, and name each.
# The container's lines pass only where systemd has not booted. CI's lint job runs it.
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
ws unit write: systemd-resolved.service: skipped, systemd is not running
ws unit write: service/base-sysctl: off on this machine, as its container is "yes"
invoke-rc.d: policy-rc.d denied execution of start.
Plan: 0 create, 0 update, 83 unchanged, 0 remove, 0 run, 0 unknown
EOF

cat >"$tmp/container" <<'EOF'
invoke-rc.d: could not determine current runlevel
mv: cannot move '/etc/resolv.conf' to '/etc/.resolv.conf.systemd-resolved.bak': Device or resource busy
Cannot take a backup of /etc/resolv.conf.
Cannot install symlink from /etc/resolv.conf to ../run/systemd/resolve/stub-resolv.conf
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

# expect NAME STATUS BOOTED LOG... runs the check on the LOGs joined, with BOOTED, and wants
# STATUS. The lines it names must be $want's, each once and in their order.
expect() {
    local name=$1 status=$2 booted=$3 got=0 out
    shift 3
    cat "$@" >"$tmp/log"
    out=$(BOOTED=$booted "$check" "$tmp/log") || got=$?
    if [ "$got" = "$status" ] && [ "$(sed -n 's/^> //p' <<<"$out")" = "$want" ]; then
        echo "ok: $name"
    else
        printf 'FAIL: %s: exit %s, not %s, and it printed:\n%s\n' "$name" "$got" "$status" "$out"
        failed=1
    fi
}

want=
expect "a clean log passes" 0 yes "$tmp/clean"
expect "the container's lines pass where systemd has not booted" 0 no "$tmp/clean" "$tmp/container"
want=$(cat "$tmp/container")
expect "the container's lines fail on a booted machine" 1 yes "$tmp/clean" "$tmp/container"
want=$(sed -E $'s/\e\\[[0-9;]*m//g' "$tmp/warnings")
expect "each warning fails, and is named" 1 no "$tmp/clean" "$tmp/warnings" "$tmp/container"

# run NAME STATUS OUT COMMAND... runs the second form on COMMAND, and wants STATUS and OUT: the
# command's output, its stderr's included, then what the check named.
run() {
    local name=$1 status=$2 want=$3 got=0 out
    shift 3
    out=$(BOOTED=yes "$check" -- "$@") || got=$?
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
exit "$failed"
