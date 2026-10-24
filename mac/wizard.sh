#!/bin/sh
# Sets up the Mac that works on ws, the server this repository sets up, in stages. Run it as
# yourself, from a clone of the repository:
#
#   ~/repository/ws/mac/wizard.sh [--check] [--update]
#
# Each stage reads what it sets up, acts where that isn't done, and reads it again. So a run takes
# up where the last one stopped, and a second prints nothing. --check prints what each stage would
# do, and changes nothing. --update has each stage that skips work by what an earlier run kept, such
# as a stamp, do that work again. A run holds a lock, a directory in $TMPDIR, and keeps the Mac from
# idle sleep until it exits. The stages, in order:
#
# - setup.sh: mac/setup.sh, which needs Homebrew and mac.conf (docs/mac.md).
#
# shellcheck source-path=SCRIPTDIR

set -eu

dir=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=lib.sh
. "$dir/lib.sh"

# args takes the wizard's arguments: --check sets WS_CHECK, as setup-lib.sh's ws_args does, and
# --update sets update.
# shellcheck disable=SC2034 # a stage that skips work by what it kept reads update
args() {
    update=0
    for arg; do
        case $arg in
        --check) WS_CHECK=1 ;;
        --update) update=1 ;;
        -h | --help)
            printf 'usage: %s [--check] [--update]\n' "$ws_prog"
            exit 0
            ;;
        *) ws_die "unknown argument '$arg'; usage: $ws_prog [--check] [--update]" ;;
        esac
    done
}

# lock takes a run's lock, a directory in TMPDIR, or stops where another run holds it. The exit
# removes it, after a signal too.
lock() {
    lock=${TMPDIR:-/tmp}
    lock=${lock%/}/ws-wizard.lock
    if ! mkdir -m 700 "$lock" 2>/dev/null; then
        [ -d "$lock" ] || ws_die "cannot make $lock"
        ws_die "another run holds $lock: wait for it to end, or remove it where none runs"
    fi
    trap 'rmdir "$lock"' EXIT
    trap 'exit 1' HUP INT TERM
}

# stage NAME FUNC runs the stage NAME through the functions whose names start with FUNC, as
# `FUNC_done || { FUNC_do; FUNC_done || ws_die "NAME: what failed; what to do"; }`:
# - FUNC_done reads alone, and succeeds where the stage is done, which then prints nothing;
# - FUNC_after prints the stage this one waits on, where that is not done, which --check prints as
#   `NAME: after STAGE` in place of what this one would do; FUNC_after's status counts for nothing;
# - FUNC_do acts, or under --check prints what it would do instead, as lib.sh's writers do;
# - FUNC_fail prints what failed and what to do, where FUNC_do fails or FUNC_done still does.
# FUNC_after is optional: in a run, the stages before have done what this one waits on, or stopped
# it. So is FUNC_done, without which FUNC_do runs at every run, and is silent where nothing differs.
# FUNC_do runs on the left of ||, where set -e is off, so it checks what it runs itself.
stage() {
    if command -v "$2_done" >/dev/null; then
        "$2_done" && return 0
    fi
    if ws_checking && command -v "$2_after" >/dev/null; then
        stage_after=$("$2_after" || true)
        if [ -n "$stage_after" ]; then
            ws_say "$1: after $stage_after"
            return 0
        fi
    fi
    "$2_do" || ws_die "$1: $("$2_fail")"
    if ! ws_checking && command -v "$2_done" >/dev/null; then
        "$2_done" || ws_die "$1: $("$2_fail")"
    fi
}

# setup_after prints Tools where PATH has no brew, at which setup.sh --check would stop.
setup_after() {
    command -v brew >/dev/null || echo Tools
}

# setup_do runs mac/setup.sh, or setup.sh --check, after sudo's password rather than amid its
# lines. setup.sh changes only what differs, and is silent then, so the stage has no setup_done.
setup_do() {
    sudo -v || ws_die "setup.sh: sudo -v failed; setup.sh needs sudo, as an admin of the Mac"
    if ws_checking; then
        sh "$dir/setup.sh" --check
    else
        sh "$dir/setup.sh"
    fi
}

# setup_fail says what to do where setup.sh stops.
setup_fail() {
    echo "mac/setup.sh stopped; fix what it printed, then run the wizard again"
}

main() {
    args "$@"
    [ "$(uname -s)" = Darwin ] || ws_die "it sets up a Mac, not $(uname -s)"
    [ "$(id -u)" != 0 ] || ws_die "run it as yourself, not root: it asks sudo where it needs root"
    # A shell started before Homebrew's install, or a profile without brew's line, lacks
    # /opt/homebrew/bin, so it goes first here, for the stages and setup.sh.
    PATH=/opt/homebrew/bin:$PATH
    export PATH
    lock
    # caffeinate keeps the Mac from idle sleep while a stage waits on a download or an answer, and
    # ends with this shell.
    caffeinate -i -w $$ </dev/null >/dev/null 2>&1 &
    stage setup.sh setup
}

main "$@"
