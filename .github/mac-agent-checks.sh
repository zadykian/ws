# shellcheck shell=sh
# The mac job's checks of ws's agent, mac/ws-agent, which mac-setup-checks.sh sources after the
# YubiKey's cases, with its functions. The YubiKey is the stand-in's, whose tokens libykcs11's link
# loads (.github/yubikey-standin.sh): 30000001 is in, with the keys of keys/ws.pub and
# keys/git.pub. A run of ws-agent's own, at a socket of its own, must load both as it starts, list
# them, and sign with each through the socket, as git does. The stand-in's ioreg lists its
# YubiKeys (yubikey-ioreg.sh): as they go out and come in, the agent must take the keys out and load
# them again, at a replug and with a second YubiKey too, and once ssh-add -D empties it. Stopped, it
# must take its ssh-agent and its socket with it. The cases leave 30000001 alone in, as they found
# it.
#
# shellcheck disable=SC2154 # tmp and error are mac-setup-checks.sh's
keys=$HOME/.config/ws/keys
socket=$tmp/agent.sock
standin=.github/yubikey-standin.sh

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

# gone succeeds where the agent at socket answers, and lists no key.
gone() {
    SSH_AUTH_SOCK=$socket /usr/bin/ssh-add -l >/dev/null 2>&1
    [ $? = 1 ]
}

# loaded LINE succeeds where the agent logged a load of the keys past its log's line LINE, and
# lists both.
loaded() {
    tail -n "+$(($1 + 1))" "$tmp/agent.log" | grep -q "loaded the YubiKey's keys" && listed
}

# lines prints the count of the agent's log's lines.
lines() {
    awk 'END { print NR }' "$tmp/agent.log"
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

WS_AGENT_SOCKET=$socket PATH=$RUNNER_TEMP/yubikey/bin:$PATH sh mac/ws-agent >"$tmp/agent.log" \
    2>&1 </dev/null &
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
sh "$standin" unplug 30000001
within 20 gone ||
    error "ws-agent lists keys 20 s after the YubiKey went out: $(cat "$tmp/agent.log")"
line=$(lines)
sh "$standin" plug 30000001
within 20 loaded "$line" || error "ws-agent loaded no keys in 20 s after the YubiKey came in"
# A replug between two of the agent's reads of ioreg, as it stops: a new ID, and no YubiKey out.
line=$(lines)
kill -s STOP "$agent_pid"
sh "$standin" unplug 30000001
sh "$standin" plug 30000001
kill -s CONT "$agent_pid"
if ! within 20 loaded "$line" ||
    tail -n "+$((line + 1))" "$tmp/agent.log" | grep -q 'IDs: none$'; then
    error "ws-agent took a replug for no YubiKey, or loaded no keys: $(cat "$tmp/agent.log")"
fi
# A second YubiKey in, then out, each a load of 30000001's keys again.
for act in plug unplug; do
    line=$(lines)
    sh "$standin" "$act" 30000002
    within 20 loaded "$line" ||
        error "ws-agent loaded no keys in 20 s after the $act of 30000002: $(cat "$tmp/agent.log")"
done
grep -q "IDs: 0x[0-9a-f]* 0x[0-9a-f]*$" "$tmp/agent.log" ||
    error "ws-agent logged no two YubiKeys in: $(cat "$tmp/agent.log")"
# Emptied by ssh-add -D, with the YubiKey still in, it must load the keys again.
line=$(lines)
SSH_AUTH_SOCK=$socket /usr/bin/ssh-add -D >/dev/null 2>&1
within 20 loaded "$line" ||
    error "ws-agent loaded no keys in 20 s after ssh-add -D: $(cat "$tmp/agent.log")"
kill "$agent_pid" 2>/dev/null || true
wait "$agent_pid" || error "ws-agent exited with $? on SIGTERM: $(cat "$tmp/agent.log")"
if [ -e "$socket" ] || pgrep -f "ssh-agent -D -a $socket" >/dev/null; then
    error "ws-agent, stopped, left its ssh-agent or its socket"
fi
