#!/bin/sh
# The Mac key stage with a key of an agent's, on the Mac that 40-again.sh leaves. The stand-in for
# Secretive's agent is an ssh-agent at its socket that holds a P-256 key, whose .pub is in
# PublicKeys. With a mac.conf of its own that names that .pub in WS_IDENTITY_PUB_KEY, --check must print
# nothing of the stage while the agent lists the key. With git's name too, it must say it would ask
# for git's key, offering that key as ws's login key, an empty WS_GIT_SIGNING_PUB_KEY too. With no
# terminal, a run must stop there, naming that key's line for git. With no WS_IDENTITY_PUB_KEY and
# PublicKeys closed, as macOS 27 denies ws Secretive's files, --check must offer the agent's key for
# ws's login. A run with no terminal must name its line. Once the agent lacks ws's key, a run and
# --check must stop at the stage, naming the socket, before ws client macos apply's. None may change
# that mac.conf, nor the Mac's own or ssh's files. The stand-in goes after.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

wsbin=$HOME/.local/bin/ws
container=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$container/Data
conf=$RUNNER_TEMP/confs/agent/mac.conf
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# keys ARG...: ws client macos wizard ARGs, with that mac.conf and no terminal. It prints the
# output, and sets status, and mac to the Mac key stage's lines.
keys() {
    out=$(WS_CONF=$conf "$wsbin" client macos wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    mac=$(printf '%s\n' "$out" | grep '^ws client macos wizard: Mac key: ')
}

# files prints the Mac's own mac.conf and ssh's files.
files() {
    conf_print ~/.config/ws/mac.conf
    cat ~/.ssh/config ~/.ssh/config.d/ws ~/.ssh/id_ed25519.pub
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
# Secretive names a key's .pub by its MD5 fingerprint, and ws's copy takes that name.
name=$(ssh-keygen -E md5 -l -f "$RUNNER_TEMP/agent.pub" |
    sed -e 's/^[0-9]* MD5://' -e 's/ .*//' -e 's/://g' -e 's/$/.pub/')
copy=$HOME/.config/ws/secretive/$name
pub=$data/PublicKeys/$name
cp "$RUNNER_TEMP/agent.pub" "$pub"
{ conf_print ~/.config/ws/mac.conf && echo "WS_IDENTITY_PUB_KEY=~/${pub#"$HOME"/}"; } | conf_write "$conf"
conf_print "$conf" >"$RUNNER_TEMP/agent.conf.before"
files >"$RUNNER_TEMP/agent.files"

keys --check
[ "$status" = 0 ] && [ -z "$mac" ] || error "ws client macos wizard --check exited $status, with '$mac'"
conf_set "$conf" 'WS_GIT_NAME="Jane Doe"' WS_GIT_EMAIL=jane@example.com
label="ws ($(ssh-keygen -l -f "$RUNNER_TEMP/agent.pub" | cut -d ' ' -f 2))"
want="ws client macos wizard: Mac key: would ask for git's signing key, in Secretive: $label, ws's login key;"
want="$want make a new one (the default)"
for empty in '' WS_GIT_SIGNING_PUB_KEY=; do
    [ -z "$empty" ] || conf_set "$conf" "$empty"
    keys --check
    [ "$status" = 0 ] && [ "$mac" = "$want" ] ||
        error "ws client macos wizard --check with git's name${empty:+ and $empty} exited $status, with '$mac', not '$want'"
done
keys
stop="ws client macos wizard: Mac key: no terminal to ask for git's signing key: add one of these lines to mac.conf,"
stop="$stop WS_GIT_SIGNING_PUB_KEY=~/.config/ws/secretive/$name for $label; or run the wizard at a terminal"
[ "$status" = 1 ] && [ "$(printf '%s\n' "$out" | grep -cxF "$stop")" = 1 ] ||
    error "ws client macos wizard with no terminal exited $status, naming no line for git's key"
rm -f "$copy"
conf_print ~/.config/ws/mac.conf | grep -v '^WS_IDENTITY_PUB_KEY' | conf_write "$conf"
chmod 000 "$data/PublicKeys"
keys --check
want="ws client macos wizard: Mac key: would ask for ws's login key, in Secretive: $label; make a new one (the default)"
[ "$status" = 0 ] && [ "$mac" = "$want" ] ||
    error "ws client macos wizard --check with no WS_IDENTITY_PUB_KEY exited $status, with '$mac', not '$want'"
keys
stop="ws client macos wizard: Mac key: no terminal to ask for ws's login key: add one of these lines to mac.conf,"
stop="$stop WS_IDENTITY_PUB_KEY=~/.config/ws/secretive/$name for $label; or run the wizard at a terminal"
[ "$status" = 1 ] && [ "$(printf '%s\n' "$out" | grep -cxF "$stop")" = 1 ] ||
    error "ws client macos wizard with no terminal exited $status, naming no line for ws's key"
chmod 755 "$data/PublicKeys"
[ ! -e "$copy" ] || error "ws client macos wizard wrote $copy, with no key named"
conf_write "$conf" <"$RUNNER_TEMP/agent.conf.before"
SSH_AUTH_SOCK=$data/socket.ssh ssh-add -D 2>/dev/null
want="ws client macos wizard: Mac key: the agent at $data/socket.ssh lists no key of $pub: open Secretive,"
want="$want where it is that agent, or start the agent, then run the wizard again"
for args in '' --check; do
    keys ${args:+"$args"}
    [ "$status" = 1 ] && [ "$mac" = "$want" ] ||
        error "ws client macos wizard${args:+ $args} exited $status, with '$mac', not '$want'"
    ! printf '%s\n' "$out" | grep -qE '^ws client macos wizard: (ws client macos apply: would apply|wrote [^ ]*/\.ssh/config\.d/ws|ran launchctl |made the Mac.s WireGuard key)' || error "ws client macos wizard${args:+ $args} ran ws client macos apply"
done
conf_print "$conf" | cmp -s - "$RUNNER_TEMP/agent.conf.before" || error "ws client macos wizard changed $conf"
files | cmp -s - "$RUNNER_TEMP/agent.files" || error "ws client macos wizard changed the Mac's own files"
exit "$fail"
