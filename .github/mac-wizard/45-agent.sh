#!/bin/sh
# The Mac key stage and the agents, on the Mac that 40-again.sh leaves. With git's key taken out of
# ws's agent, --check must say that a run would load the keys, and a run must stop, naming git's
# .pub. Where launchd has gui/UID, it runs ws's agent, which the stage starts again, so the run must
# pass, with both keys listed. With libykcs11's link away, as ws's agent then exits, a run must stop
# with ssh-add's own words. With the YubiKey out, a run must wait for it, and say no more once it
# comes back. Then the stand-in for Secretive's agent is an ssh-agent at its socket that holds a
# P-256 key, whose .pub is in PublicKeys. With a mac.conf of its own that names that .pub in
# WS_IDENTITY, --check must print nothing of the stage while the agent lists the key, with git's
# name too, as git's key is the YubiKey's. Once the agent lacks it, a run and --check must stop at
# the stage, naming the socket, before setup.sh's. None may change the Mac's own or ssh's files.
# The stand-ins go after.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
github=$HOME/repository/ws/.github
socket=$HOME/.ssh/ws-agent.sock
keys=$HOME/.config/ws/keys
container=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$container/Data
pub=$data/PublicKeys/0123456789abcdef0123456789abcdef.pub
conf=$HOME/.config/ws/mac.conf
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# keys ARG...: wizard.sh ARGs, with conf's mac.conf and no terminal. It prints the output, and sets
# status, and mac to the Mac key stage's lines.
keys() {
    out=$(WS_CONF=$conf sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    mac=$(printf '%s\n' "$out" | grep '^wizard\.sh: Mac key: ')
}

# stopped WANT: the run before must have exited 1 at the Mac key stage, with WANT in its lines, and
# run no setup.sh.
stopped() {
    if [ "$status" != 1 ] || ! printf '%s\n' "$mac" | grep -qF -e "wizard.sh: Mac key: $1" ||
        printf '%s\n' "$out" | grep -q '^setup\.sh: '; then
        error "wizard.sh exited $status, with '$mac', not '$1'"
    fi
}

# files prints the Mac's own mac.conf and ssh's files.
files() {
    cat ~/.config/ws/mac.conf ~/.ssh/config ~/.ssh/config.d/ws "$keys"/*.pub
}

files >"$RUNNER_TEMP/agent.files"
SSH_AUTH_SOCK=$socket ssh-add -d "$keys/git.pub"
keys --check
want="wizard.sh: Mac key: would load the keys ws and git into ws's agent, as setup.sh does, and"
want="$want wait for it to list them"
[ "$status" = 0 ] && [ "$mac" = "$want" ] || error "wizard.sh --check exited $status, with '$mac'"
keys
if launchctl print "gui/$(id -u)" >/dev/null 2>&1; then
    [ "$status" = 0 ] && [ "$(SSH_AUTH_SOCK=$socket ssh-add -l | wc -l)" -eq 2 ] ||
        error "wizard.sh exited $status, or launchd's agent lacks a key"
else
    stopped "ws's agent at $socket lists no key of $keys/git.pub. launchd runs it from"
fi
sh "$github/yubikey-agent.sh" restart

library=$(brew --prefix yubico-piv-tool)/lib/libykcs11.dylib
mv "$library" "$library.away"
tries=0
while SSH_AUTH_SOCK=$socket ssh-add -l >/dev/null 2>&1 || [ $? != 2 ]; do
    [ "$((tries += 1))" -le 30 ] || { error "ws's agent stayed without libykcs11" && break; }
    sleep 1
done
keys
stopped "ws's agent at $socket: ssh-add -L: "
mv "$library.away" "$library"
sh "$github/yubikey-agent.sh" restart

sh "$github/yubikey-standin.sh" unplug 30000001
WS_CONF=$conf sh "$wizard" </dev/null >"$RUNNER_TEMP/agent.out" 2>&1 &
pid=$!
sleep 5
sh "$github/yubikey-standin.sh" plug 30000001
wait "$pid"
status=$?
out=$(cat "$RUNNER_TEMP/agent.out")
printf '%s\n' "$out"
mac=$(printf '%s\n' "$out" | grep '^wizard\.sh: Mac key: ')
want="wizard.sh: Mac key: no YubiKey is in: plug in the one with ws's keys, or a new one; the"
want="$want wizard waits up to 5 minutes for it"
[ "$status" = 0 ] && [ "$mac" = "$want" ] || error "wizard.sh exited $status, with '$mac'"

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
conf=$RUNNER_TEMP/agent.conf
{ cat ~/.config/ws/mac.conf && echo "WS_IDENTITY=~/${pub#"$HOME"/}"; } >"$conf"
cp "$conf" "$RUNNER_TEMP/agent.conf.before"

keys --check
[ "$status" = 0 ] && [ -z "$mac" ] || error "wizard.sh --check exited $status, with '$mac'"
printf '%s\n' 'WS_GIT_NAME="Jane Doe"' WS_GIT_EMAIL=jane@example.com >>"$conf"
keys --check
[ "$status" = 0 ] && [ -z "$mac" ] ||
    error "wizard.sh --check with git's name exited $status, with '$mac'"
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
