#!/bin/sh
# The Mac key stage and the agents, on the Mac that 40-again.sh leaves. With git's key taken out of
# ws's agent, --check must say that a run would load the keys, and a run must stop, naming git's
# .pub. Where launchd has gui/UID, it runs ws's agent, which the stage starts again, so the run must
# pass, with both keys listed. With libykcs11's link away, as ws's agent then exits, a run must stop
# with ssh-add's own words. With the YubiKey out, a run must wait for it, and say no more once it
# comes back. Then a mac.conf of its own with a line of another key, of WS_IDENTITY,
# WS_IDENTITY_AGENT or WS_GIT_SIGNING_KEY, empty too, must stop a run and --check before setup.sh's,
# naming WS_SSH_KEY or WS_GIT_KEY. An empty WS_GIT_SIGNING_KEY is as the wizard offered it.
# None may change the Mac's own or ssh's files.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
github=$HOME/repository/ws/.github
socket=$HOME/.ssh/ws-agent.sock
keys=$HOME/.config/ws/keys
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

# refused WANT LINE...: with the Mac's own mac.conf and LINE..., a run and --check must exit 1 with
# WANT among their lines, before setup.sh's, and leave conf as it was.
refused() {
    want=$1
    shift
    { cat ~/.config/ws/mac.conf && printf '%s\n' "$@"; } >"$conf"
    cp "$conf" "$RUNNER_TEMP/agent.conf.before"
    for args in '' --check; do
        keys ${args:+"$args"}
        case $status:$out in
        1:*"$want"*) ;;
        *) error "wizard.sh${args:+ $args} with $* exited $status, not with '$want'" ;;
        esac
        ! printf '%s\n' "$out" | grep -q '^setup\.sh: ' ||
            error "wizard.sh${args:+ $args} with $* ran setup.sh"
    done
    cmp -s "$conf" "$RUNNER_TEMP/agent.conf.before" || error "wizard.sh changed $conf"
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

trap 'rm -f "$RUNNER_TEMP"/agent*' EXIT
conf=$RUNNER_TEMP/agent.conf
alone="as ssh offers ws the YubiKey's key alone, which WS_SSH_KEY names"
refused "WS_IDENTITY is gone, $alone" 'WS_IDENTITY=~/.ssh/id_ed25519.pub'
refused "WS_IDENTITY is gone, $alone" WS_IDENTITY=
refused "WS_IDENTITY_AGENT is gone, $alone" 'WS_IDENTITY_AGENT=~/.ssh/agent.sock' \
    'WS_IDENTITY=~/.ssh/id_ed25519.pub'
refused "WS_GIT_SIGNING_KEY is gone, as git signs with the YubiKey's key alone, which WS_GIT_KEY\
 names, not with ~/.ssh/id_ed25519_signing" WS_GIT_SIGNING_KEY=
files | cmp -s - "$RUNNER_TEMP/agent.files" || error "wizard.sh changed the Mac's own files"
exit "$fail"
