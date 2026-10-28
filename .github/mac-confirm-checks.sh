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

# agent ASKPASS starts an agent at a.sock, a path relative to the round's directory, as macOS
# caps a socket's path at 104 bytes. ASKPASS is its askpass, and HOME the round's directory, whose
# .config/ws/keys the askpass reads. The names there are not mac.conf's defaults.
agent() {
    rm -f a.sock
    eval "$(env SSH_ASKPASS="$1" SSH_ASKPASS_REQUIRE=force PATH="$PWD/bin:$PATH" HOME="$PWD" \
        WS_SSH_KEY=mac-ws WS_GIT_KEY=mac-git ssh-agent -a a.sock)" >/dev/null
    agents="$agents $SSH_AGENT_PID"
    for key in ws git other; do
        ssh-add -c "$key" 2>/dev/null || error "ssh-add -c $key failed"
    done
}

# sign KEY signs data with KEY's public key alone, so through the agent, as git does.
sign() {
    ssh-keygen -Y sign -n git -f "$1.pub" <data >/dev/null 2>&1
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
sleep 5
EOF
    chmod 755 bin/terminal-notifier
    for key in ws git other; do
        ssh-keygen -q -t ed25519 -N '' -C "$key's comment" -f "$key"
    done
    cp ws.pub .config/ws/keys/mac-ws.pub
    cp git.pub .config/ws/keys/mac-git.pub
    echo data >data
    if out=$(unset SSH_ASKPASS_PROMPT && ./ws-agent-confirm "Allow use of key ws?"); then
        error "$1: the askpass took a prompt without SSH_ASKPASS_PROMPT=confirm, a PIN's"
    fi
    [ -z "$out" ] || error "$1: the askpass printed $out for a PIN's prompt"

    agent "$PWD/ws-agent-confirm"
    for key in ws git other; do
        sign "$key" || error "$1: $key's signature through the askpass failed"
    done
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
