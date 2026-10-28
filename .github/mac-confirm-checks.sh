#!/bin/sh
# The checks of mac/ws-agent-confirm, the askpass that ws's agent runs at each signature of a key
# loaded with ssh-add -c. Run it from the repository's root, on macOS or on Linux: it starts agents
# of its own, with file keys, which -c constrains as it does the YubiKey's. Under /bin/sh, and dash
# where installed, it runs a copy of the askpass, with a HOME of its own and a stand-in
# terminal-notifier first on PATH, which logs each call and lives 5 s. Each failure prints an
# ::error:: line, and any fails the script at its end.

set -eu
src=$(pwd)/mac/ws-agent-confirm
[ -x "$src" ] || { echo "::error::no executable $src: run it at the repository's root" && exit 1; }
tmp=$(mktemp -d "${TMPDIR:-/tmp}/mac-confirm-checks.XXXXXXXX")
# The agents a failure left running go too.
agents=''
trap 'kill $agents 2>/dev/null || :; rm -rf "$tmp"' EXIT
trap 'exit 1' HUP INT TERM
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# agent ASKPASS [WINDOW] starts an agent at a.sock, a path relative to the round's directory, as
# macOS caps a socket's path at 104 bytes. ASKPASS is its askpass, and HOME the round's directory,
# whose .config/ws/keys the askpass reads. The names there are not mac.conf's defaults, and ws.pub
# holds git's key, so a match by any *.pub would show. WINDOW, 5 s by default, is the askpass's.
agent() {
    rm -f a.sock
    eval "$(env SSH_ASKPASS="$1" SSH_ASKPASS_REQUIRE=force PATH="$PWD/bin:$PATH" HOME="$PWD" \
        WS_SSH_KEY=mac-ws WS_GIT_KEY=mac-git WS_CONFIRM_WINDOW="${2:-5}" ssh-agent -a a.sock)" \
        >/dev/null
    agents="$agents $SSH_AGENT_PID"
    for key in ws git other; do
        ssh-add -c "$key" 2>/dev/null || error "ssh-add -c $key failed"
    done
}

# sign KEY signs data with KEY's public key alone, so through the agent, as git does.
sign() {
    ssh-keygen -Y sign -n git -f "$1.pub" <data >/dev/null 2>&1
}

# calls N succeeds where the stand-in has logged N calls, within 3 s, as it runs in the
# background.
calls() {
    tries=0
    while [ "$(cat log 2>/dev/null | wc -l)" -ne "$1" ]; do
        tries=$((tries + 1))
        [ "$tries" -le 30 ] || return 1
        sleep 0.1
    done
}

# call N TEXT succeeds where the stand-in's call N holds TEXT.
call() {
    sed -n "$1p" log | grep -qF -- "$2"
}

# round NAME SHELL checks a copy of the askpass that SHELL runs, in a directory NAME of its own.
round() {
    mkdir "$tmp/$1"
    cd "$tmp/$1"
    mkdir -p bin .config/ws/keys
    { echo "#!$2" && tail -n +2 "$src"; } >ws-agent-confirm
    chmod 755 ws-agent-confirm
    cat >bin/terminal-notifier <<EOF
#!/bin/sh
printf '%s|' "\$@" >>"$PWD/log"
echo >>"$PWD/log"
[ ! -f "$PWD/refuse" ] || exit 3
sleep 5
EOF
    chmod 755 bin/terminal-notifier
    for key in ws git other; do
        ssh-keygen -q -t ed25519 -N '' -C "$key's comment" -f "$key"
    done
    cp ws.pub .config/ws/keys/mac-ws.pub
    cp git.pub .config/ws/keys/mac-git.pub
    cp git.pub .config/ws/keys/ws.pub
    echo data >data
    if out=$(unset SSH_ASKPASS_PROMPT && ./ws-agent-confirm "Allow use of key ws?"); then
        error "$1: the askpass took a prompt without SSH_ASKPASS_PROMPT=confirm, a PIN's"
    fi
    [ -z "$out" ] || error "$1: the askpass printed $out for a PIN's prompt"

    # The burst's agent folds a key's uses for 60 s, so that a slow runner's 50 signatures fall in
    # one window. 8 idle connections to it put the askpass's pipe above descriptor 9, which a shell
    # that takes such descriptors closes by /dev/fd: dash takes none, so it runs with none.
    agent "$PWD/ws-agent-confirm" 60
    holds=
    if "$2" -c '{ :; } 10</dev/null' 2>/dev/null; then
        for _ in 1 2 3 4 5 6 7 8; do
            nc -U a.sock </dev/null >/dev/null 2>&1 &
            holds="$holds $!"
        done
        sleep 1
    fi
    start=$(date +%s)
    sign git || error "$1: the first signature failed"
    # shellcheck disable=SC2086 # the pids, split bare
    [ -z "$holds" ] || kill $holds 2>/dev/null || :
    # A notifier that held the agent's pipe would stall each notice's signature by 5 s: the round
    # ends here.
    if [ $(($(date +%s) - start)) -gt 2 ]; then
        error "$1: the first signature waited for the notifier, which holds the agent's pipe"
        return
    fi
    passed=1
    for _ in $(seq 49); do
        if sign git; then passed=$((passed + 1)); fi
    done
    [ "$passed" = 50 ] || error "$1: $passed of 50 signatures passed"
    calls 1 || error "$1: 50 signatures in a burst posted $(wc -l <log) notices, not 1"
    call 1 '|git key mac-git|-message|1 signature|-group|ws-agent-git|' ||
        error "$1: the first notice was $(sed -n 1p log)"
    sign ws || error "$1: ws's signature failed"
    sign other || error "$1: the other key's signature failed"
    calls 3 || error "$1: ws's and the other key's posted no notice of their own"
    call 2 '|ssh key mac-ws|-message|1 signature|-group|ws-agent-ssh|' ||
        error "$1: ws's notice was $(sed -n 2p log)"
    call 3 "|key other's comment|" || error "$1: the other key's notice was $(sed -n 3p log)"

    # The next agent's window is the default, 5 s, which git's key passes 6 s on.
    kill "$SSH_AGENT_PID"
    agent "$PWD/ws-agent-confirm"
    sleep 6
    sign git || error "$1: git's signature after the window failed"
    if ! { calls 4 && call 4 '|50 signatures since '; }; then
        error "$1: the notice after the window was $(sed -n 4p log), not of 49 folded and 1"
    fi
    touch refuse
    sign ws || error "$1: with the notifier refused, ws's signature failed"
    calls 5 || error "$1: the refusing notifier was not called"
    # With the stand-in gone, the askpass finds Homebrew's, where the job installed it, or none.
    mv bin/terminal-notifier bin/gone
    sign other || error "$1: with the stand-in gone, the other key's signature failed"
    sleep 1
    calls 5 || error "$1: the notifier was called once moved away"
    kill "$SSH_AGENT_PID"

    agent "$PWD/missing"
    ! sign git || error "$1: with no askpass, the agent still signed"
    kill "$SSH_AGENT_PID"
    cd "$tmp"
}

round sh /bin/sh
if dash=$(command -v dash); then
    round dash "$dash"
fi
exit "$fail"
