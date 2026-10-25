#!/bin/sh
# The Mac key stage with a key of an agent's, on the Mac that 40-again.sh leaves. The stand-in for
# Secretive's agent is an ssh-agent at its socket that holds a P-256 key, whose .pub is in
# PublicKeys. With a mac.conf of its own that names that .pub in WS_IDENTITY, --check must print
# nothing of the stage while the agent lists the key. With git's name too, it must say it would wait
# for git's key made in Secretive, unless mac.conf has an empty WS_GIT_SIGNING_KEY. Once the agent
# lacks ws's key, a run and --check must stop at the stage, naming the socket, before setup.sh's.
# None may change that mac.conf, nor the Mac's own or ssh's files. The stand-in goes after.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
container=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$container/Data
pub=$data/PublicKeys/0123456789abcdef0123456789abcdef.pub
conf=$RUNNER_TEMP/agent.conf
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# keys ARG...: wizard.sh ARGs, with that mac.conf and no terminal. It prints the output, and sets
# status, and mac to the Mac key stage's lines.
keys() {
    out=$(WS_CONF=$conf sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    mac=$(printf '%s\n' "$out" | grep '^wizard\.sh: Mac key: ')
}

# files prints the Mac's own mac.conf and ssh's files.
files() {
    cat ~/.config/ws/mac.conf ~/.ssh/config ~/.ssh/config.d/ws ~/.ssh/id_ed25519.pub
}

if [ -e "$container" ]; then
    echo "::error::the runner has $container, which the case makes"
    exit 1
fi
trap 'kill "$agent" 2>/dev/null; rm -rf "$container" "$RUNNER_TEMP"/agent*' EXIT
mkdir -p "$data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$RUNNER_TEMP/agent" </dev/null
agent=$(ssh-agent -a "$data/socket.ssh" | sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$RUNNER_TEMP/agent"
cp "$RUNNER_TEMP/agent.pub" "$pub"
{ cat ~/.config/ws/mac.conf && echo "WS_IDENTITY=~/${pub#"$HOME"/}"; } >"$conf"
cp "$conf" "$RUNNER_TEMP/agent.conf.before"
files >"$RUNNER_TEMP/agent.files"

keys --check
[ "$status" = 0 ] && [ -z "$mac" ] || error "wizard.sh --check exited $status, with '$mac'"
printf '%s\n' 'WS_GIT_NAME="Jane Doe"' WS_GIT_EMAIL=jane@example.com >>"$conf"
keys --check
want="wizard.sh: Mac key: would open Secretive, and wait for git's signing key made there"
[ "$status" = 0 ] && [ "$mac" = "$want" ] ||
    error "wizard.sh --check with git's name exited $status, with '$mac', not '$want'"
echo WS_GIT_SIGNING_KEY= >>"$conf"
keys --check
[ "$status" = 0 ] && [ -z "$mac" ] ||
    error "wizard.sh --check with an empty WS_GIT_SIGNING_KEY exited $status, with '$mac'"
cp "$RUNNER_TEMP/agent.conf.before" "$conf"
SSH_AUTH_SOCK=$data/socket.ssh ssh-add -D 2>/dev/null
want="wizard.sh: Mac key: the agent at $data/socket.ssh lists no key of $pub: open Secretive,"
want="$want where it is that agent, or start the agent, then run the wizard again"
for args in '' --check; do
    keys ${args:+"$args"}
    [ "$status" = 1 ] && [ "$mac" = "$want" ] ||
        error "wizard.sh${args:+ $args} exited $status, with '$mac', not '$want'"
    ! printf '%s\n' "$out" | grep -q '^setup\.sh: ' || error "wizard.sh${args:+ $args} ran setup.sh"
done
cmp -s "$conf" "$RUNNER_TEMP/agent.conf.before" || error "wizard.sh changed $conf"
files | cmp -s - "$RUNNER_TEMP/agent.files" || error "wizard.sh changed the Mac's own files"
exit "$fail"
