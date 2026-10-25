# shellcheck shell=sh
# The wizard's stage Hook on ws: the apply.sh of another repository, which mac.conf's WS_HOOK names
# as OWNER/REPO, run as root on ws. The wizard's files script, wizard-remote-files.sh, clones the
# repository in /root/repository/REPO there, by root's gh login, where it is missing, and never
# pulls it. apply.sh then runs by plain ssh. A stamp on ws marks a run of it that exited 0, so that
# a second run of the wizard leaves it alone. wizard.sh sources it, and runs the stage after the
# Tunnel. It calls the functions of lib.sh, setup-lib.sh, wizard-host.sh and wizard-ws.sh, and reads
# wizard.sh's dir, update and WS_CONF, and wizard-host.sh's root_in.

# wshook_ran becomes 1 once the stage ran apply.sh in this run.
wshook_ran=0

# remote_files OP [ARG...] runs the files script's OP as root on ws, by host_ssh, as remote does.
# shellcheck disable=SC2154 # dir is wizard.sh's
remote_files() {
    host_ssh -o BatchMode=yes -o ServerAliveInterval=15 ws-ssh sh -s -- "$@" \
        <"$dir/wizard-remote-files.sh"
}

# wshook_after prints the stage that the clone waits on, where that isn't done. That is ws's run,
# which installs gh on ws, then gh on ws, whose login clones the repository.
wshook_after() {
    if ! run_done; then
        echo "ws's run"
    elif ! wsgh_done; then
        echo "gh on ws"
    fi
}

# wshook_done succeeds where mac.conf names no repository, or where ws has the stamp of its
# apply.sh, which names it as OWNER/REPO. Under --update, only once the stage ran apply.sh in this
# run. It reads ws only where root logged in there in this run.
# shellcheck disable=SC2154 # update is wizard.sh's, root_in wizard-host.sh's
wshook_done() {
    ws_conf "$WS_CONF"
    [ -n "$WS_HOOK" ] || return 0
    [ "$update" = 0 ] || [ "$wshook_ran" = 1 ] || return 1
    [ "$root_in" = 1 ] && remote_read &&
        case " $(remote_get hook) " in *" $WS_HOOK "*) ;; *) false ;; esac
}

# wshook_do runs the repository's apply.sh as root on ws, by plain ssh, never ssh -t, at which such
# a script could prompt and echo its stdin. The files script's clone first clones the repository
# where ws lacks it, and removes its stamp, which its stamp op writes again once apply.sh exits 0.
# --check says what it would do.
wshook_do() {
    wshook_apply=/root/repository/${WS_HOOK#*/}/apply.sh
    if ws_checking; then
        ws_say "Hook on ws: would clone $WS_HOOK into ${wshook_apply%/*} on ws, where missing," \
            "then run its apply.sh there"
        return 0
    fi
    ws_say "Hook on ws: $WS_HOOK's apply.sh runs as root on ws"
    # shellcheck disable=SC2034 # wizard-ws.sh's remote_read reads it
    remote_got=0
    remote_files clone "$WS_HOOK" >&2 || return 1
    host_ssh -o BatchMode=yes -o ServerAliveInterval=15 ws-ssh "$wshook_apply" </dev/null >&2 ||
        return 1
    remote_files stamp "$WS_HOOK" >&2 || return 1
    wshook_ran=1
}

# wshook_fail says what to do where the clone or apply.sh failed, or ws still lacks the stamp.
wshook_fail() {
    echo "the clone of $WS_HOOK on ws, or its apply.sh, failed, as above; mend what they printed," \
        "then run the wizard again, which runs apply.sh again"
}
