# shellcheck shell=sh
# The wizard's stage ws's run: ws's playbook, bootstrap.sh, in a unit of its own on ws. The remote
# script that the wizard sends to root there over ssh, wizard-remote.sh, starts it and follows it. A
# run goes on where ssh drops, and the next run of the wizard follows it. Then the stage gh on ws:
# root's gh there, logged in to GitHub in a session of ssh -t, and ws's signing key on the account.
# The stage claude on ws logs root's claude in the same way. Last, what stays by hand of ws, the
# router and the Mac, for the closing list. wizard.sh sources it, and runs the stages after Root's
# key. It calls the functions of lib.sh and wizard-host.sh, and reads wizard.sh's dir, lock, update
# and acted, and the values of mac.conf that ws_conf set.

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

# wsgh_after prints ws's run, which installs gh on ws, where that stage isn't done.
wsgh_after() {
    run_done || echo "ws's run"
}

# wsgh_done succeeds where root's gh on ws has a login to github.com, and the account holds ws's
# signing key, or root's git config there names no key file. It reads ws only where root logged in.
wsgh_done() {
    [ "$root_in" = 1 ] && remote_read && [ "$(remote_get gh)" = ok ]
}

# wsgh_do logs root's gh on ws in to github.com, or gives its login the write:ssh_signing_key scope,
# where it lacks either, in a session of ssh -t at the terminal. gh's device flow shows a code
# there, for the page that the Mac's browser opens. So the login is ws's own, never a copy of the
# Mac's. Then bootstrap.sh --tags devtools, in a unit of its own, adds ws's signing key. --check
# says what it would do.
wsgh_do() {
    case $(remote_get gh) in
    none) ws_die "gh on ws: ws lacks gh, which its run installs: run the wizard with --update" ;;
    login) wsgh_how="gh auth login -h github.com -p https -w -s write:ssh_signing_key" ;;
    scope) wsgh_how="gh auth refresh -h github.com -s write:ssh_signing_key" ;;
    *) wsgh_how= ;;
    esac
    wsgh_run="run bootstrap.sh --tags devtools on ws, which adds ws's signing key to GitHub"
    [ -z "$wsgh_how" ] ||
        wsgh_run="open github.com/login/device on the Mac, run $wsgh_how on ws, then $wsgh_run"
    if ws_checking; then
        ws_say "gh on ws: would $wsgh_run"
        return 0
    fi
    if [ -n "$wsgh_how" ]; then
        [ -t 0 ] || ws_die "gh on ws: root's gh lacks a login with the write:ssh_signing_key" \
            "scope, and no terminal is there for $wsgh_how: run it on ws, then the wizard again"
        open https://github.com/login/device ||
            ws_say "gh on ws: warning: open github.com/login/device in a browser yourself"
        ws_say "gh on ws: $wsgh_how runs as root on ws: enter the code it shows at" \
            "github.com/login/device, which the Mac's browser opened"
        host_ssh -t ws-ssh "$wsgh_how" || return 1
    fi
    ws_say "gh on ws: bootstrap.sh --tags devtools runs on ws in a unit of its own, to add ws's" \
        "signing key to GitHub"
    remote_got=0
    remote run --tags devtools >&2
}

# wsgh_fail says what to do where the account still lacks ws's signing key.
wsgh_fail() {
    echo "root's gh on ws still lacks its login, or the account ws's signing key, as above; mend" \
        "what gh or bootstrap.sh printed, then run the wizard again"
}

# wsclaude_after prints ws's run, which installs claude on ws, where that stage isn't done.
wsclaude_after() {
    run_done || echo "ws's run"
}

# wsclaude_done succeeds where root's claude on ws has a login, as wsgh_done reads gh's state.
wsclaude_done() {
    [ "$root_in" = 1 ] && remote_read && [ "$(remote_get claude)" = ok ]
}

# wsclaude_do runs claude auth login as root on ws, in a session of ssh -t at the terminal, through
# a login shell, whose PATH has ~/.local/bin, where claude is. --check says it would.
wsclaude_do() {
    [ "$(remote_get claude)" != none ] || ws_die "claude on ws: ws lacks root's claude, which its" \
        "run installs: run the wizard with --update"
    if ws_checking; then
        ws_say "claude on ws: would run claude auth login as root on ws, at the terminal"
        return 0
    fi
    [ -t 0 ] || ws_die "claude on ws: root's claude has no login, and no terminal is there for" \
        "claude auth login: run it on ws, then the wizard again"
    ws_say "claude on ws: claude auth login runs as root on ws: take its steps at this terminal," \
        "and open the address it shows in the Mac's browser"
    remote_got=0
    host_ssh -t ws-ssh 'exec bash -lc "claude auth login"'
}

# wsclaude_fail says what to do where root's claude on ws still has no login.
wsclaude_fail() {
    echo "root's claude on ws still has no login, as above; run the wizard again"
}

# run_hand adds what stays by hand of ws, the router and the Mac to the closing list, after a run in
# which a stage acted. ws's part comes from the remote script's status, which the stage read.
# shellcheck disable=SC2154 # acted is wizard.sh's, host_first wizard-host.sh's
run_hand() {
    [ "$acted" = 1 ] || return 0
    remote_read || true
    [ "$(remote_get fans)" != refused ] || by_hand "ws's fans' driver: with Secure Boot on," \
        "DKMS's key, enrolled at ws's console (docs/fans.md, Secure Boot)."
    by_hand "macOS's local network: where it asks whether iTerm2, Gateway or another app may find" \
        "devices there, allow it (docs/mac.md, ssh's hosts)."
    by_hand "Gateway's connection: an SSH connection to ws in JetBrains Gateway, with Parse" \
        "config file ~/.ssh/config on (docs/mac.md)."
    [ -z "$host_first" ] || by_hand "ws's lease: static at ${WS_LAN_ADDRESS:-10.88.0.10}, as" \
        "docs/router.md has it, as this run reached ws at $host_first."
    [ -z "$WS_REMOTE" ] || by_hand "The router's forwards to ws, UDP 51820 and TCP" \
        "${WS_REMOTE_PORT:-22022}, for $WS_REMOTE, as docs/router.md has them."
}
