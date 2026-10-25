#!/bin/sh
# The first run, by sh, with standin.sh's PATH, which lacks brew. The wizard must still find brew,
# so that setup.sh installs wireguard-tools, root's copies and the Mac's key, and writes ssh's
# files. Without WS_PUBLIC_KEY, setup.sh loads no daemon, and prints the Mac's key, which the
# wizard drops: it must end with the tunnel's step by hand instead. It must make the ssh key that
# mac.conf names, ~/.ssh/id_ed25519, beside its .pub, with no passphrase, as no terminal asked one.
# It must add the .pub to the stand-in gh's login, titled with the Mac's name.
# It must add the environment's answers at the end of mac.conf, and leave no lock or file of its
# own in $TMPDIR. It must warn that ws, reached through WS_REMOTE, faces the internet, and add ws's
# host key to known_hosts under ws, as WS_HOST_KEY gives its fingerprint. As the runner's user, the
# installer's for WS_INSTALLER_USER, it must add the Mac's key to root's authorized_keys with sudo.
# It must run ws's playbook once, bootstrap.sh without arguments in a unit with root's HOME, and
# stamp its status 0.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0
# mac.conf's answers that standin.sh leaves out, from the environment, each empty for no value.
export WS_LAN='' WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS=''

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

host=$(cat "$RUNNER_TEMP/ws/fingerprint")
WS_HOST_KEY=$host WS_INSTALLER_USER=$USER PATH=$(cat "$RUNNER_TEMP/path") sh "$wizard" \
    >"$RUNNER_TEMP/run.out" 2>"$RUNNER_TEMP/run.err"
status=$?
cat "$RUNNER_TEMP/run.err" "$RUNNER_TEMP/run.out"
[ "$status" = 0 ] || error "wizard.sh exited $status"
brew list --formula wireguard-tools >/dev/null 2>&1 ||
    error 'wizard.sh installed no wireguard-tools'
for each in /usr/local/libexec/ws-tunnel/wg /etc/ws-tunnel/private.key ~/.ssh/config.d/ws; do
    sudo test -f "$each" || error "wizard.sh made no $each"
done
[ "$(ssh-keygen -y -P '' -f ~/.ssh/id_ed25519 | cut -d ' ' -f 1-2)" = \
    "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ] ||
    error 'wizard.sh made no ~/.ssh/id_ed25519 with no passphrase, beside its .pub'
added="ssh-key add $HOME/.ssh/id_ed25519.pub --title $(hostname -s)"
if ! grep -qxF "$added" "$RUNNER_TEMP/gh/log" ||
    ! grep -qxF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" "$RUNNER_TEMP/gh/keys"; then
    error "wizard.sh added no ~/.ssh/id_ed25519.pub to gh's login, titled with the Mac's name"
fi
grep -q "^wizard\.sh: Host key: warning: through the router's forward, " "$RUNNER_TEMP/run.err" ||
    error "wizard.sh gave no warning that ws faces the internet through WS_REMOTE"
grep -qxF "ws $(cut -d ' ' -f 1-2 "$RUNNER_TEMP/ws/host_key.pub")" ~/.ssh/known_hosts ||
    error "wizard.sh added no line for ws's host key to ~/.ssh/known_hosts"
if ! sudo grep -qxF "$(cut -d ' ' -f 1-2 ~/.ssh/id_ed25519.pub)" ~root/.ssh/authorized_keys ||
    ! grep -q "^$USER: sudo sh -c " "$RUNNER_TEMP/ws/log"; then
    error "wizard.sh added no ~/.ssh/id_ed25519.pub to root's authorized_keys, as $USER with sudo"
fi
if ! printf '\n' | cmp -s - "$RUNNER_TEMP/ws/bootstrap.log" ||
    ! grep -q -- '--setenv=HOME=/root ' "$RUNNER_TEMP/ws/run.log" ||
    ! grep -q ' 0$' "$RUNNER_TEMP/ws/root/var/lib/ws-wizard/run"; then
    error "wizard.sh ran no bootstrap.sh on ws, once, in a unit with root's HOME, that exited 0"
fi
! launchctl print system/com.github.zadykian.ws-tunnel >/dev/null 2>&1 ||
    error "wizard.sh loaded the daemon without ws's key"
! grep -q 'WireGuard public key' "$RUNNER_TEMP/run.out" || error "wizard.sh printed setup.sh's key"
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY=~/.ssh/id_ed25519.pub' WS_LAN= \
    WS_GIT_NAME= WS_TUNNEL_NETWORKS= WS_RESOLVERS= | cmp -s - ~/.config/ws/mac.conf ||
    error "wizard.sh wrote another mac.conf than standin.sh's lines and the environment's answers"
if [ "$(sed -n 2p "$RUNNER_TEMP/run.out")" != 'What stays by hand:' ] ||
    ! grep -q "^- The Mac's tunnel: run mac/setup.sh, " "$RUNNER_TEMP/run.out"; then
    error "wizard.sh listed no tunnel's step by hand"
fi
for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
    [ ! -e "$left" ] || error "wizard.sh left $left"
done
exit "$fail"
