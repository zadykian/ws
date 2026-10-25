# shellcheck shell=sh
# The wizard's stage ws's run: ws's playbook, bootstrap.sh, in a unit of its own on ws. The remote
# script that the wizard sends to root there over ssh, wizard-remote.sh, starts it and follows it. A
# run goes on where ssh drops, and the next run of the wizard follows it. wizard.sh sources it, and
# runs the stage after Root's key. It calls the functions of lib.sh and wizard-host.sh, and reads
# wizard.sh's dir, lock and update.

# remote_st holds what the remote script's status printed, once remote_got is 1. run_ran becomes 1
# once the stage ran the playbook in this run.
remote_got=0
remote_st=
run_ran=0

# remote OP [ARG...] runs the remote script's OP as root on ws, through host_ssh, which reaches ws
# at the address that the Host key stage asked, where it asked one. No ARG holds a blank or a quote.
# shellcheck disable=SC2154 # dir is wizard.sh's
remote() {
    host_ssh -o BatchMode=yes -o ServerAliveInterval=15 ws-ssh sh -s -- "$@" \
        <"$dir/wizard-remote.sh"
}

# remote_read reads the remote script's status into remote_st once a run, and again once remote_got
# is 0, as after the stage ran the playbook. It runs in the wizard's shell, as $(...) would drop
# what it read. Its messages go in remote.err in the lock. It fails where ssh or the script did.
# shellcheck disable=SC2154 # lock is wizard.sh's
remote_read() {
    if [ "$remote_got" = 0 ]; then
        remote_got=1
        remote_st=$(remote status 2>"$lock/remote.err") || remote_st=
    fi
    [ -n "$remote_st" ]
}

# remote_get NAME prints the value of NAME in what remote_read read, and fails where it read none.
remote_get() {
    [ -n "$remote_st" ] || return 1
    printf '%s\n' "$remote_st" | awk -v name="$1" '$1 == name { sub(/^[^ ]* /, ""); print; exit }'
}

# run_after prints Root's key where root didn't log in to ws by the Mac's key in this run, which
# the remote script needs.
# shellcheck disable=SC2154 # root_in is wizard-host.sh's
run_after() {
    [ "$root_in" = 1 ] || echo "Root's key"
}

# run_done succeeds where ws's last full run exited 0, and no run's unit runs there now. Under
# --update, only once the stage ran the playbook in this run. It reads ws only where root logged in
# there in this run.
# shellcheck disable=SC2154 # update is wizard.sh's
run_done() {
    [ "$update" = 0 ] || [ "$run_ran" = 1 ] || return 1
    [ "$root_in" = 1 ] && remote_read && [ "$(remote_get active)" = - ] &&
        [ "$(remote_get run | cut -d ' ' -f 2)" = 0 ]
}

# run_do runs ws's playbook in a unit of its own on ws through the remote script, or follows the run
# that goes on there, with its log on stderr. Under --update, the remote script pulls ws's clone
# first. --check says what it would do.
run_do() {
    set --
    run_how="run bootstrap.sh on ws in a unit of its own, and follow its log"
    if [ "$update" = 1 ]; then
        set -- --update
        run_how="pull ws's clone where it is on main and clean, then $run_how"
    fi
    if ws_checking; then
        remote_read || true
        run_unit=$(remote_get active) || run_unit=-
        run_last=$(remote_get run) || run_last='- -'
        case $run_unit:${run_last#* } in
        -:- | -:0) ws_say "ws's run: would $run_how" ;;
        -:*) ws_say "ws's run: would $run_how; its last full run exited ${run_last#* }" ;;
        *) ws_say "ws's run: would follow $run_unit, which runs on ws" ;;
        esac
        return 0
    fi
    ws_say "ws's run: bootstrap.sh runs on ws in a unit of its own, which goes on where ssh" \
        "drops; a first run takes 20 to 40 minutes"
    remote_got=0
    remote run "$@" >&2 || return 1
    run_ran=1
}

# run_fail says what to do where ws's playbook ran no full run to its end.
run_fail() {
    echo "ws's playbook stopped, as above, or ssh dropped while it goes on; mend what it printed," \
        "then run the wizard again, which follows a run that goes on"
}
