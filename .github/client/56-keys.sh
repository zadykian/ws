#!/bin/bash
# The keys that mise bootstrap's units of ssh and git set up from values.yml. The login key of ws
# goes in /etc/ws/ssh_config, and git's signing key in /etc/ws/gitconfig and its allowed signers.
# Each case edits the job's values.yml, runs the bootstrap and reads the files. A refusal must fail
# the bootstrap and its plan, with the reason, and change none of the files. This file holds the
# functions, and sources the cases of ws's key and git's: client-identity-checks.sh and
# client-git-checks.sh. The job's values.yml comes back as it exits, on a signal too, and the cases
# end with the job's files.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

conf=/etc/ws/values.yml
tmpdir=${TMPDIR:-/tmp}
tmp=$(mktemp -d "${tmpdir%/}/client-keys.XXXXXXXX")
conf_print "$conf" >"$tmp/job.conf"
# restores lists the functions that put back what a sourced file's cases changed, run at the exit.
restores=
# shellcheck disable=SC2329 # the trap below calls it
keys_restore() {
    for each in $restores; do
        "$each" || true
    done
    [ ! -f "$tmp/job.conf" ] || conf_write "$conf" <"$tmp/job.conf"
    rm -rf "$tmp"
}
trap keys_restore EXIT
trap 'exit 1' HUP INT TERM
fail=0
nl=$'\n'

# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

# with 'KEY: value'... gives the job's values.yml those lines in place of the job's of the same keys.
with() {
    conf_print "$tmp/job.conf" | conf_write "$conf"
    [ $# = 0 ] || conf_set "$conf" "$@"
}

# boots runs mise bootstrap, which must pass, and its plan, which must then change nothing. out
# holds what the bootstrap printed.
boots() {
    out=$(client_bootstrap 2>&1) || {
        error "mise bootstrap failed: $out"
        return 0
    }
    client_bootstrap plan --detailed-exitcode >/dev/null 2>&1 || error "the plan after mise bootstrap exited $?, not 0"
}

# files prints the SHA-256 of the files that ssh's and git's units write, a line each.
files() {
    for each in /etc/ws/ssh_config /etc/ws/gitconfig /etc/ws/allowed_signers /etc/ws/secretive-ssh-keygen \
        ~/.ssh/config ~/.gitconfig; do
        sum=none
        [ ! -e "$each" ] || sum=$(shasum -a 256 <"$each")
        printf '%s %s\n' "$each" "$sum"
    done
}

# refuses TEXT: mise bootstrap's plan, then the bootstrap, must fail with TEXT in what they print,
# and leave the files as they were. The bootstrap runs by mise, as ws client setup would wait for a
# key where values.yml names no login key.
refuses() {
    before=$(files)
    for run in plan apply; do
        status=0
        if [ "$run" = plan ]; then
            out=$(client_bootstrap plan 2>&1) || status=$?
        else
            out=$(client_tree_mise bootstrap 2>&1) || status=$?
        fi
        if [ "$status" = 0 ] || ! grep -qF -e "$1" <<<"$out"; then
            error "mise bootstrap's $run exited $status, printing '$out', where a failure and '$1' were due"
        fi
    done
    [ "$(files)" = "$before" ] || error "mise bootstrap changed a file where it refused '$1'"
}

# shellcheck source=.github/client-identity-checks.sh
. .github/client-identity-checks.sh
# shellcheck source=.github/client-git-checks.sh
. .github/client-git-checks.sh
exit "$fail"
