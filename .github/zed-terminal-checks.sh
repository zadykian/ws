#!/bin/sh
# Tests the zed role's terminal script and tmux.conf against the tmux on the PATH. Panes of an
# outer tmux stand for Zed's terminal tabs, and both servers keep their sockets in a directory of
# their own. Run it from the repository's root, with the directory that holds the two files:
#
#   sh .github/zed-terminal-checks.sh [DIR]    # DIR is internal/server/zed/files by default
#
# Each case prints ok or FAIL. Any FAIL makes the exit status 1.

# shellcheck disable=SC2329 # the cases call these functions through until_ok and result
set -eu

dir=$(cd "${1:-internal/server/zed/files}" && pwd -P)
work=$(mktemp -d)
cleanup() {
    tmux -L outer kill-server 2>/dev/null || true
    tmux -L zed kill-server 2>/dev/null || true
    rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
mkdir "$work/tmux" "$work/proj" "$work/proj/ws" "$work/proj/ws-foo" "$work/other" "$work/other/ws"
unset TMUX
export TMUX_TMPDIR="$work/tmux"
failed=0

# zt runs tmux on the script's socket, and ot on the outer one, whose panes are the tabs.
zt() { tmux -L zed "$@"; }
ot() { tmux -L outer "$@"; }

# until_ok runs a command every 0.1 s, for 10 s at most, until it succeeds.
until_ok() {
    n=0
    until "$@"; do
        n=$((n + 1))
        [ "$n" -lt 100 ] || return 1
        sleep 0.1
    done
}

# result NAME COMMAND prints ok where the command succeeds, else FAIL.
result() {
    name=$1
    shift
    if "$@"; then
        echo "ok   $name"
    else
        echo "FAIL $name"
        failed=1
    fi
}

# project_of DIR is the name the script gives DIR's sessions, up to their PID.
project_of() {
    printf '%s-%s' "$(basename "$1" | tr -c 'A-Za-z0-9_\n-' _)" \
        "$(printf %s "$1" | cksum | cut -d ' ' -f 1)"
}

# detached DIR makes a session of DIR's that no client shows, as a tab that has ended leaves.
detached() {
    zt -f "$dir/tmux.conf" new-session -d -s "$(project_of "$1")-$(sh -c 'echo $$')"
}

# tab NAME DIR [-e NAME=VALUE]... starts the script in DIR as Zed does, in a pane of its own, once go
# is touched. The -e options are the tab's own environment.
tab() {
    name=$1
    start=$2
    shift 2
    ot new-session -d -s "$name" -x 100 -y 30 -c "$start" "$@" \
        "while [ ! -e '$work/go' ]; do :; done; exec env -u TMUX TERM=xterm-256color '$dir/terminal'"
}
go() { touch "$work/go"; }

no_server() { ! zt list-sessions >/dev/null 2>&1; }
# reset ends both servers and holds the tabs back again.
reset() {
    ot kill-server 2>/dev/null || true
    zt kill-server 2>/dev/null || true
    rm -f "$work/go"
    until_ok no_server
}

clients() { zt list-clients 2>/dev/null | wc -l; }
have_clients() { [ "$(clients)" -eq "$1" ]; }
sessions() { zt list-sessions 2>/dev/null | wc -l; }
# shown N is true where the sessions have N clients in all, and none has two.
shown() {
    [ "$(clients)" -eq "$1" ] && ! zt list-sessions -F '#{session_attached}' | grep -qv '^[01]$'
}
# named SESSION is true where the session exists.
named() { zt has-session -t "=$1" 2>/dev/null; }
pane_pid() { ot list-panes -t "$1" -F '#{pane_pid}'; }

echo "== T0 tmux.conf loads and its options take effect"
reset
zt -f "$dir/tmux.conf" new-session -d -s probe
got=$(
    zt show -sv extended-keys
    zt show -sv extended-keys-format
    zt show -gv status
    zt show -gv status-interval
    zt show -gv prefix
    zt show -gv mouse
    zt show -gv allow-passthrough
    zt show -sv focus-events
)
result "extended-keys always, csi-u, status on every 10 s, prefix None, mouse on, passthrough on, \
focus-events on" \
    test "$got" = "$(printf '%s\n' always csi-u on 10 None on on on)"

echo "== T1 no session: a tab starts one, and exit ends it and the server"
reset
tab t1 "$work/proj/ws"
go
result "one session, one client" until_ok shown 1
result "named PROJECT-PID, PID the tab's" named "$(project_of "$work/proj/ws")-$(pane_pid t1)"
result "tmux.conf was read through the script's own path" test "$(zt show -gv prefix)" = None
ot send-keys -t t1 exit Enter
result "the server ends with the session" until_ok no_server

echo "== T2 a detached session of this directory's is taken, and named for the tab"
reset
detached "$work/proj/ws"
tab t1 "$work/proj/ws"
go
result "one client" until_ok shown 1
result "the session is named for the tab's PID" named "$(project_of "$work/proj/ws")-$(pane_pid t1)"
result "it is the only session" test "$(sessions)" -eq 1

echo "== T3 the prefix trap: neither ws-foo-123 nor a project ws-foo is taken by project ws"
reset
zt -f "$dir/tmux.conf" new-session -d -s ws-foo-123
detached "$work/proj/ws-foo"
tab t1 "$work/proj/ws"
go
result "a new session beside the two" until_ok shown 1
result "three sessions" test "$(sessions)" -eq 3
echo "== T4 two directories named ws share nothing"
reset
detached "$work/other/ws"
tab t1 "$work/proj/ws"
go
result "a new session beside the other's" until_ok shown 1
result "two sessions" test "$(sessions)" -eq 2

echo "== T5 a second tab does not take the first's session"
reset
tab t1 "$work/proj/ws"
go
result "the first tab shows a session" until_ok shown 1
tab t2 "$work/proj/ws"
result "two clients, each on a session" until_ok shown 2
result "two sessions" test "$(sessions)" -eq 2

echo "== T6 a tab closed with its client killed leaves a session that the next tab takes"
reset
tab t1 "$work/proj/ws"
go
result "the first tab shows a session" until_ok shown 1
ot kill-session -t t1
result "the session stays, shown to no one" until_ok shown 0
tab t2 "$work/proj/ws"
result "the next tab takes it" until_ok shown 1
result "it is named for the new tab" named "$(project_of "$work/proj/ws")-$(pane_pid t2)"
echo "== T7 tabs that start together each end on a session of their own"
# together K N: K detached sessions and N tabs started at once end as the larger number of
# sessions, with N clients and none shown twice. That holds in each of 10 rounds.
together() {
    want=$1
    [ "$2" -le "$1" ] || want=$2
    bad=0
    for _ in 1 2 3 4 5 6 7 8 9 10; do
        reset
        i=0
        while [ "$i" -lt "$1" ]; do
            detached "$work/proj/ws"
            i=$((i + 1))
        done
        i=0
        while [ "$i" -lt "$2" ]; do
            tab "t$i" "$work/proj/ws"
            i=$((i + 1))
        done
        sleep 0.3
        go
        until_ok have_clients "$2" || true
        sleep 0.5
        { shown "$2" && [ "$(sessions)" -eq "$want" ]; } || bad=$((bad + 1))
    done
    test "$bad" -eq 0
}
result "1 detached, 2 tabs together" together 1 2
result "3 detached, 3 tabs together" together 3 3
result "3 detached, 5 tabs together" together 3 5
result "0 detached, 3 tabs together" together 0 3
result "2 detached, 2 tabs together" together 2 2
result "5 detached, 5 tabs together" together 5 5

echo "== T8 a session whose tab still lives is not taken"
reset
zt -f "$dir/tmux.conf" new-session -d -s "$(project_of "$work/proj/ws")-$$"
tab t1 "$work/proj/ws"
go
result "a new session beside it" until_ok shown 1
result "two sessions" test "$(sessions)" -eq 2

echo "== T9 without tmux on the PATH the script runs a login shell"
reset
cat >"$work/shell" <<EOF
#!/bin/sh
echo "\$0 \$*" >"$work/shell.out"
EOF
chmod +x "$work/shell"
(cd "$work/proj/ws" && SHELL="$work/shell" PATH="$work/none" /bin/sh "$dir/terminal")
result "it ran \$SHELL -l" test "$(cat "$work/shell.out")" = "$work/shell -l"

echo "== T10 a new session takes the tab's environment, though the server started with another"
reset
tab t1 "$work/proj/ws"
go
result "the first tab starts the server" until_ok shown 1
tab t2 "$work/other/ws" -e ZED_TRIAL_ENV=other -e "ZED_TWO=$(printf 'a\nb')"
result "the second tab shows a session of its own" until_ok shown 2
# sees SESSION is true where the shell of the session prints both variables as the tab had them.
sees() { zt capture-pane -p -t "=$1:" | grep -qx 'env=other two=a+b'; }
# shellcheck disable=SC2016 # the pane's shell expands the line, not this one
zt send-keys -t "=$(project_of "$work/other/ws")-$(pane_pid t2):" \
    'echo "env=${ZED_TRIAL_ENV:-unset} two=$(printf %s "$ZED_TWO" | tr "\n" +)"' Enter
result "its shell sees a variable and one of two lines" \
    until_ok sees "$(project_of "$work/other/ws")-$(pane_pid t2)"

exit "$failed"
