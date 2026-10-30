# shellcheck shell=sh
# The wizard's stage ws's run: ws's playbook, bootstrap.sh, in a unit of its own on ws. The remote
# script that the wizard sends to root there over ssh, wizard-remote.sh, starts it and follows it. A
# run goes on where ssh drops, and the next run of the wizard follows it. Then the stage claude on
# ws, which logs root's claude in, in a session of ssh -t. Last, what stays by hand of ws, the
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
    host_ssh -o BatchMode=yes ws-ssh sh -s -- "$@" \
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

# run_done succeeds where ws's last full run exited 0, and no run's unit runs there now. On a cloud
# VM, that run's -e must have turned the outbound table off, and ws's host_vars must keep it off.
# Under --update, only once the stage ran the playbook in this run. It reads ws only where root
# logged in there in this run.
# shellcheck disable=SC2154 # update is wizard.sh's
run_done() {
    [ "$update" = 0 ] || [ "$run_ran" = 1 ] || return 1
    ws_conf "$WS_CONF"
    [ "$root_in" = 1 ] && remote_read && [ "$(remote_get active)" = - ] &&
        [ "$(run_status)" = 0 ] &&
        { [ "$WS_TARGET" != cloud ] || { run_off && [ "$(remote_get outbound)" = off ]; }; }
}

# run_status prints the status of ws's last full run, as the remote script's status read it, or -.
run_status() {
    run_word=$(remote_get run | cut -d ' ' -f 2)
    echo "${run_word:--}"
}

# run_off succeeds where ws's last full run turned the outbound table off by its -e.
run_off() {
    [ "$(remote_get run | cut -d ' ' -f 3)" = off ]
}

# run_do runs ws's playbook in a unit of its own on ws through the remote script, or follows the run
# that goes on there, with its log on stderr. Under --update, the remote script pulls ws's clone
# first. On a cloud VM, where the outbound table would drop the provider's metadata service, the
# run's -e turns it off, and its unit names that in host_vars for the runs after. A unit that the
# run followed may have lacked that -e: a run of its own follows then. Where the line is missing
# still, run_outbound writes it, alone where a run with -e passed before. --check says what it would
# do. It reads mac.conf, which nothing else read before under --check --update.
run_do() {
    ws_conf "$WS_CONF"
    set --
    run_how="run bootstrap.sh on ws in a unit of its own, and follow its log"
    if [ "$update" = 1 ]; then
        set -- --update
        run_how="pull ws's clone where it is on main and clean, then $run_how"
    fi
    if [ "$WS_TARGET" = cloud ]; then
        set -- "$@" -e security_outbound=false
        run_how="$run_how, with -e security_outbound=false, then name that in ws's host_vars"
        if [ "$update" = 0 ] && [ "$(remote_get active)" = - ] && [ "$(run_status)" = 0 ] &&
            run_off; then
            run_how="name security_outbound: false in ws's host_vars"
            ws_checking || {
                run_outbound
                return
            }
        fi
    fi
    if ws_checking; then
        remote_read || true
        run_unit=$(remote_get active) || run_unit=-
        run_last=$(run_status)
        case $run_unit:$run_last in
        -:- | -:0) ws_say "ws's run: would $run_how" ;;
        -:*) ws_say "ws's run: would $run_how; its last full run exited $run_last" ;;
        *) ws_say "ws's run: would follow $run_unit, which runs on ws" ;;
        esac
        return 0
    fi
    ws_say "ws's run: bootstrap.sh runs on ws in a unit of its own, which goes on where ssh" \
        "drops; a first run takes 20 to 40 minutes"
    remote_got=0
    remote run "$@" >&2 || return 1
    run_ran=1
    [ "$WS_TARGET" = cloud ] || return 0
    remote_read || return 1
    if ! run_off; then
        ws_say "ws's run: the unit it followed ran without -e security_outbound=false, so" \
            "bootstrap.sh runs again with it"
        remote_got=0
        remote run "$@" >&2 && remote_read || return 1
    fi
    [ "$(remote_get outbound)" = off ] || run_outbound
}

# run_outbound names security_outbound: false in ws's host_vars by the remote script's outbound op,
# and stops with its error where that fails. --update runs the playbook, which makes a clone gone.
# shellcheck disable=SC2154 # lock is wizard.sh's
run_outbound() {
    remote_got=0
    remote outbound 2>"$lock/outbound.err" || ws_die "ws's run: ws's host_vars lacks" \
        "security_outbound: false: $(tail -n 1 "$lock/outbound.err"); mend that, or run the" \
        "wizard with --update, which runs ws's playbook again"
    cat "$lock/outbound.err" >&2
}

# run_fail says what to do where ws's playbook ran no full run to its end.
run_fail() {
    echo "ws's playbook stopped, as above, or ssh dropped while it goes on; mend what it printed," \
        "then run the wizard again, which follows a run that goes on"
}

# wsclaude_after prints ws's run, which installs claude on ws, where that stage isn't done.
wsclaude_after() {
    run_done || echo "ws's run"
}

# wsclaude_done succeeds where root's claude on ws has a login. It reads ws only where root logged
# in there.
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
# which a stage acted. ws's part comes from the remote script's status, which the stage read. ws's
# gh has a login by apply.sh of WS_HOOK's repository, or else by hand.
# mac.conf has WS_LAN_ADDRESS with WS_LAN alone, so the lease names an address only with a LAN.
# Off the LAN, run_away's lines take the router's place.
# shellcheck disable=SC2154 # acted is wizard.sh's, host_first wizard-host.sh's
run_hand() {
    [ "$acted" = 1 ] || return 0
    remote_read || true
    [ -n "$WS_HOOK" ] || by_hand "ws's gh login: gh auth login -h github.com --with-token on" \
        "ws, with a fine-grained token, as mac.conf names no WS_HOOK (README.md, By hand)."
    [ "$(remote_get fans)" != refused ] || by_hand "ws's fans' driver: with Secure Boot on," \
        "DKMS's key, enrolled at ws's console (docs/fans.md, Secure Boot)."
    [ "$WS_TARGET" = cloud ] || [ "$(remote_get outbound)" != off ] || by_hand "ws's outbound" \
        "table: off, as ws's host_vars names security_outbound: false, as for a cloud VM; remove" \
        "that line, then run the wizard with --update (docs/security.md, Outbound)."
    by_hand "macOS's local network: where it asks whether iTerm2, Gateway or another app may find" \
        "devices there, allow it (docs/mac.md, ssh's hosts)."
    by_hand "Gateway's connection: an SSH connection to ws in JetBrains Gateway, with Parse" \
        "config file ~/.ssh/config on (docs/mac.md)."
    [ "$WS_TARGET" = lan ] || {
        run_away
        return 0
    }
    [ -z "$host_first" ] || by_hand "ws's lease: static${WS_LAN_ADDRESS:+ at $WS_LAN_ADDRESS}," \
        "as docs/router.md has it, as this run reached ws at $host_first."
    [ -z "$WS_REMOTE" ] || by_hand "The router's forwards to ws, UDP ${WS_PORT:-51820} and TCP" \
        "${WS_REMOTE_PORT:-22022}, for $WS_REMOTE, as docs/router.md has them."
}

# run_away adds what stays by hand of a cloud VM or a VM on the Mac, which have no router. Where the
# run asked ws's address, mac.conf's way must lead to it. A cloud VM's provider must let the tunnel
# in (docs/mac-targets.md).
# shellcheck disable=SC2154 # host_first is wizard-host.sh's
run_away() {
    run_way=WS_LAN_ADDRESS
    [ "$WS_TARGET" != cloud ] || run_way=WS_REMOTE
    [ -z "$host_first" ] || by_hand "ws's address: $run_way in mac.conf must lead to ws, as this" \
        "run reached it at $host_first (docs/mac-targets.md)."
    [ "$WS_TARGET" != cloud ] || by_hand "The provider's firewall: UDP ${WS_PORT:-51820} open to" \
        "ws, for the tunnel (docs/mac-targets.md)."
}
