# shellcheck shell=sh
# The mac job's checks of ws's agent, mac/ws-agent, which mac-setup-checks.sh sources after the
# YubiKey's cases, with its functions. The YubiKey is the stand-in's, whose tokens libykcs11's link
# loads (.github/yubikey-standin.sh): 30000001 is in, with the keys of keys/ws.pub and
# keys/git.pub. A run of ws-agent's own, at a socket of its own, must load both as it starts, list
# them, and sign with each through the socket, as git does. Stopped, it must take its ssh-agent and
# its socket with it.
#
# shellcheck disable=SC2154 # tmp and error are mac-setup-checks.sh's
keys=$HOME/.config/ws/keys
socket=$tmp/agent.sock

# listed succeeds where the agent at socket lists the keys of keys/ws.pub and keys/git.pub, each by
# its type and base64.
listed() {
    SSH_AUTH_SOCK=$socket /usr/bin/ssh-add -L >"$tmp/listed" 2>&1 || return 1
    for name in ws git; do
        awk -v key="$(awk '{ print $1, $2 }' "$keys/$name.pub")" '$1 " " $2 == key { found = 1 }
            END { exit !found }' "$tmp/listed" || return 1
    done
}

# within SECONDS COMMAND... runs COMMAND each second until it succeeds, for SECONDS at most.
within() {
    left=$1
    shift
    until "$@"; do
        [ "$left" -gt 0 ] || return 1
        left=$((left - 1))
        sleep 1
    done
}

# signs NAME succeeds where ssh-keygen -Y sign, given keys/NAME.pub alone, signs through the agent
# at socket, and the signature verifies.
signs() {
    echo ws >"$tmp/signed"
    rm -f "$tmp/signed.sig"
    printf '%s %s\n' "$1" "$(awk '{ print $1, $2 }' "$keys/$1.pub")" >"$tmp/signers"
    SSH_AUTH_SOCK=$socket /usr/bin/ssh-keygen -q -Y sign -f "$keys/$1.pub" -n git "$tmp/signed" \
        2>"$tmp/sign.err" && /usr/bin/ssh-keygen -q -Y verify -f "$tmp/signers" -I "$1" -n git \
        -s "$tmp/signed.sig" <"$tmp/signed" >>"$tmp/sign.err" 2>&1
}

WS_AGENT_SOCKET=$socket sh mac/ws-agent >"$tmp/agent.log" 2>&1 </dev/null &
agent_pid=$!
if within 20 listed; then
    for name in ws git; do
        signs "$name" || error "ssh-keygen -Y sign with keys/$name.pub through ws-agent made no" \
            "signature that verifies: $(cat "$tmp/sign.err")"
    done
else
    error "ws-agent lists no keys of keys/ws.pub and keys/git.pub in 20 s:" \
        "$(cat "$tmp/listed" "$tmp/agent.log")"
fi
kill "$agent_pid" 2>/dev/null || true
wait "$agent_pid" || error "ws-agent exited with $? on SIGTERM: $(cat "$tmp/agent.log")"
if [ -e "$socket" ] || pgrep -f "ssh-agent -D -a $socket" >/dev/null; then
    error "ws-agent, stopped, left its ssh-agent or its socket"
fi
