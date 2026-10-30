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
# idle sleep until it exits. A run in which a stage acted ends with what stays by hand. The stages,
# in order, after the Tools and Clone of mac/bootstrap.sh, which runs the wizard on a new Mac:
#
# - Answers: the values mac.conf lacks, from the environment or asked at the terminal;
# - Mac key: the key that ssh offers ws, made where missing, in Secretive or a file of ssh-keygen's;
# - gh on the Mac: gh's login to GitHub, and the Mac's key among its keys, for Ubuntu's installer;
# - setup.sh: mac/setup.sh, which needs Homebrew and mac.conf (docs/mac.md);
# - Host key: ws's host key, matched against ws's console, in known_hosts under the name ws;
# - Root's key: root's login to ws by the Mac's key, which the wizard checks and never adds;
# - ws's run: ws's playbook, bootstrap.sh, in a unit of its own on ws, followed to its end;
# - gh on ws: root's gh there, logged in to GitHub over ssh -t, and ws's signing key on the account;
# - claude on ws: root's claude there, logged in over ssh -t;
# - Tunnel: the Mac's WireGuard key as wg0's peer on ws, WS_PUBLIC_KEY wg0's, and setup.sh again;
# - Hook on ws: the repository that WS_HOOK names, cloned on ws, and its apply.sh run there as root;
# - Hook on the Mac: that repository's clone in ~/repository, its hooks and its Brewfile's formulae;
# - Word list: ws's, copied to the Mac where it differs;
# - Cable: on the LAN, ws's port of the cable from the Mac's display, in host_vars on ws, and the
#   Mac's service.
#
# shellcheck source-path=SCRIPTDIR

set -eu

dir=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=lib.sh
. "$dir/lib.sh"
# shellcheck source=setup-lib.sh
. "$dir/setup-lib.sh"
# shellcheck source=wizard-ask.sh
. "$dir/wizard-ask.sh"
# shellcheck source=wizard-keys.sh
. "$dir/wizard-keys.sh"
# shellcheck source=wizard-gh.sh
. "$dir/wizard-gh.sh"
# shellcheck source=wizard-host.sh
. "$dir/wizard-host.sh"
# shellcheck source=wizard-ws.sh
. "$dir/wizard-ws.sh"
# shellcheck source=wizard-tunnel.sh
. "$dir/wizard-tunnel.sh"
# shellcheck source=wizard-hook.sh
. "$dir/wizard-hook.sh"
# shellcheck source=wizard-words.sh
. "$dir/wizard-words.sh"
# shellcheck source=wizard-cable.sh
. "$dir/wizard-cable.sh"

# acted becomes 1 once a stage acts in a run, and hand holds what stays by hand, a line each.
acted=0
hand=

# args takes the wizard's arguments: --check sets WS_CHECK, as setup-lib.sh's ws_args does, and
# --update sets update.
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

# lock takes a run's lock, a directory in TMPDIR that holds the run's own files, or stops where
# another run holds it. The exit ends the run's ssh masters (host_ssh) and removes the lock, after a
# signal too.
lock() {
    lock=${TMPDIR:-/tmp}
    lock=${lock%/}/ws-wizard.lock
    if ! mkdir -m 700 "$lock" 2>/dev/null; then
        [ -d "$lock" ] || ws_die "cannot make $lock"
        ws_die "another run holds $lock: wait for it to end, or remove it where none runs"
    fi
    trap 'host_end; rm -rf "$lock"' EXIT
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
# FUNC_do runs on the left of ||, where set -e is off, so it checks what it runs itself. A stage
# with FUNC_done acted where it was not done; one without sets acted itself.
stage() {
    if command -v "$2_done" >/dev/null; then
        "$2_done" && return 0
        ws_checking || acted=1
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

# by_hand TEXT... adds a line to what stays by hand, which a stage finds as it runs.
by_hand() {
    hand="$hand- $*
"
}

# closing prints what stays by hand, after a run in which a stage acted.
closing() {
    [ "$acted" = 1 ] && [ -n "$hand" ] || return 0
    printf '\nWhat stays by hand:\n%s' "$hand"
}

# setup_after prints Tools, bootstrap.sh's stage, where PATH has no brew, at which setup.sh --check
# would stop.
setup_after() {
    command -v brew >/dev/null || echo Tools
}

# setup_do runs mac/setup.sh, or setup.sh --check, after sudo's password rather than amid its
# lines. setup.sh changes only what differs, and is silent then, so the stage has no setup_done: a
# run acted where setup.sh printed more than warnings. Until mac.conf has WS_PUBLIC_KEY, setup.sh
# prints the Mac's key and the commands that make it ws's peer at every run, on stdout. So a run
# drops that, as the Tunnel stage takes that step, and runs setup_do again.
setup_do() {
    sudo -v || ws_die "setup.sh: sudo -v failed; setup.sh needs sudo, as an admin of the Mac"
    if ws_checking; then
        sh "$dir/setup.sh" --check
        return
    fi
    # A mac.conf that ws_conf refuses stops setup.sh, which says why.
    exec 3>/dev/null
    if (ws_conf "$WS_CONF" && [ -n "$WS_PUBLIC_KEY" ]) 2>/dev/null; then
        exec 3>&1
    fi
    {
        setup_status=0
        sh "$dir/setup.sh" 2>&1 >&3 3>&- || setup_status=$?
        echo "$setup_status" >"$lock/setup.status"
    } | tee "$lock/setup.log" >&2
    exec 3>&-
    [ "$(cat "$lock/setup.status")" = 0 ] || return 1
    if grep -qv '^setup\.sh: warning: ' "$lock/setup.log"; then
        acted=1
    fi
    ws_conf "$WS_CONF"
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
    # mac.conf, which WS_CONF names for setup.sh too, by a path from /, as ws_put writes it.
    case ${WS_CONF:=$HOME/.config/ws/mac.conf} in /*) ;; *) WS_CONF=$PWD/$WS_CONF ;; esac
    export PATH WS_CONF
    lock
    # caffeinate keeps the Mac from idle sleep while a stage waits on a download or an answer, and
    # ends with this shell.
    caffeinate -i -w $$ </dev/null >/dev/null 2>&1 &
    stage Answers ask
    stage "Mac key" keys
    stage "gh on the Mac" gh
    stage setup.sh setup
    stage "Host key" host
    stage "Root's key" root
    stage "ws's run" run
    stage "gh on ws" wsgh
    stage "claude on ws" wsclaude
    stage Tunnel tunnel
    stage "Hook on ws" wshook
    stage "Hook on the Mac" hook
    stage "Word list" words
    stage Cable cable
    run_hand
    hook_hand
    cable_hand
    closing
}

main "$@"
