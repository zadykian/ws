#!/bin/sh
# The stand-ins of the wizard's job, made once before checks.sh runs the cases. They are what a new
# Mac, its user and ws give the wizard, on GitHub's macOS runner.

set -eu
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

sw_vers
# ws client macos apply stops where a directory above root's copies isn't root's alone, as on a Mac
# whose /usr/local an old Homebrew took. Where the runner's image gave /usr/local or its libexec to
# its user, they become root's, as in mac.yml: their contents stay as they are.
for dir in /usr/local /usr/local/libexec; do
    [ -d "$dir" ] || continue
    if [ "$(stat -f '%u %Lp' "$dir")" != "0 755" ]; then
        echo "::warning::$dir was $(stat -f '%Su:%Sg %Lp' "$dir"); made root's, 755"
        sudo chown root:wheel "$dir"
        sudo chmod 755 "$dir"
    fi
done
# The conf's answers: ws by the router's name and port alone, as a Mac away from home has it. The
# file ~/.ssh/id_ed25519, made here, is ws's key, as the wizard makes none in a file. Nor can a run
# with no terminal ask for one of Secretive's.
mkdir -p ~/.config/ws
[ -d ~/.ssh ] || mkdir -m 700 ~/.ssh
ssh-keygen -q -t ed25519 -N '' -f ~/.ssh/id_ed25519 </dev/null
printf '%s\n' "WS_USER=$(id -un)" WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub' |
    conf_write ~/.config/ws/mac.conf
# PATH as a shell that predates Homebrew has it, without the directories that hold brew, for the
# cases that give it to the wizard.
path=
IFS=:
for each in $PATH; do
    [ -x "$each/brew" ] || path=$path${path:+:}$each
done
unset IFS
if (PATH=$path && command -v brew); then
    echo "::error::brew is still on PATH, $path"
    exit 1
fi
echo "$path" >"$RUNNER_TEMP/path"
# Stand-ins for the apps of Ghostty and Zed, their font and cld spare the runs
# their downloads, as in mac.yml.
sh "$(dirname "$0")/../mac-standins.sh"
# A stand-in gh, logged in to github.com, for the runner's, which has no login. The wizard puts
# /opt/homebrew/bin first on PATH, so the stand-in replaces the runner's link there. Its state is in
# $RUNNER_TEMP/gh. login is there while it has one. noscope is there where its login lacks the
# scopes, which gh api answers with a 404 status. nosign gives that 404 for the signing keys alone.
# keys holds its login's keys, signing its signing keys, and log its calls.
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$RUNNER_TEMP/gh"
: >"$RUNNER_TEMP/gh/keys"
: >"$RUNNER_TEMP/gh/signing"
touch "$RUNNER_TEMP/gh/login"
rm -f /opt/homebrew/bin/gh
cat >/opt/homebrew/bin/gh <<EOF
#!/bin/sh
state='$RUNNER_TEMP/gh'
echo "\$*" >>"\$state/log"
[ -f "\$state/login" ] || [ "\$1" != api ] || exit 4
case \$* in
'auth status -h github.com') [ -f "\$state/login" ] ;;
'api --paginate user/keys --jq .[].key')
    [ ! -f "\$state/noscope" ] || { echo 'gh: Not Found (HTTP 404)' >&2 && exit 1; }
    cat "\$state/keys"
    ;;
'api --paginate user/ssh_signing_keys --jq .[].key')
    [ ! -f "\$state/nosign" ] || { echo 'gh: Not Found (HTTP 404)' >&2 && exit 1; }
    cat "\$state/signing"
    ;;
'ssh-key add '*' --type signing '*) awk 'NF { print \$1, \$2; exit }' "\$3" >>"\$state/signing" ;;
'ssh-key add '*) awk 'NF { print \$1, \$2; exit }' "\$3" >>"\$state/keys" ;;
*) echo "gh: the stand-in takes no \$*" >&2 && exit 2 ;;
esac
EOF
chmod 755 /opt/homebrew/bin/gh
# standin NAME PORT makes a server's stand-in: an sshd of its own at 127.0.0.1:PORT, where the
# server's WS_REMOTE and its port lead. Its host key is made here, and its files are in
# $RUNNER_TEMP/NAME. The runner's user logs in by its own authorized_keys. The cases fill it from
# the stand-in gh's keys, as Ubuntu's installer imports the account's keys for its user. The
# image's file goes, so that no key is there before a case adds it. Each session runs ws.sh. A
# second host key, ECDSA, as ws has more than one, must stay out of known_hosts at each login.
standin() {
    dir=$RUNNER_TEMP/$1
    mkdir -p "$dir" ~/.ssh
    rm -f ~/.ssh/authorized_keys
    ssh-keygen -q -t ed25519 -N '' -C "$1" -f "$dir/host_key" </dev/null
    ssh-keygen -q -t ecdsa -N '' -C "$1" -f "$dir/host_ecdsa" </dev/null
    ssh-keygen -l -f "$dir/host_key.pub" | awk '{ print $2 }' >"$dir/fingerprint"
    printf '%s\n' "ListenAddress 127.0.0.1:$2" "HostKey $dir/host_key" "PidFile $dir/sshd.pid" \
        'PasswordAuthentication no' 'KbdInteractiveAuthentication no' \
        'PermitRootLogin prohibit-password' 'UsePAM no' 'StrictModes no' 'ExposeAuthInfo yes' \
        "SetEnv WS_STANDIN=$dir" "ForceCommand $dir/ws.sh" "HostKey $dir/host_ecdsa" >"$dir/sshd_config"
    # sshd 9.8 and later refuse an address for a while after connections that don't log in, which the
    # cases make in a row. The stand-in takes them, where its sshd knows the setting.
    ! sudo /usr/sbin/sshd -G -f "$dir/sshd_config" -o PerSourcePenalties=no >/dev/null 2>&1 ||
        echo 'PerSourcePenalties no' >>"$dir/sshd_config"
    # ws.sh is ws's shell. It logs each command in log, with the user that runs it, which the runner's
    # sudo makes root. It logs the connection it came in by in conns, then runs the command.
    # A login's authorized_keys may give the key a command, as cloud-init's does for root. There ws.sh
    # stops as that command does, since sshd's ForceCommand runs in its place. It puts ws's programs
    # first on PATH, and roots the paths of the wizard's scripts on ws, each a line NAME=PATH of their
    # own, in root here. So it roots a command's path from /root or the wizard's state alone, with its
    # arguments, or after the script of sh -c. It roots ws's binary too, which ws client macos secrets
    # runs, bare or past sudo's options.
    cat >"$dir/ws.sh" <<'EOF'
#!/bin/sh
run=${SSH_ORIGINAL_COMMAND#sudo -n }
run=${run#sudo -k -S -p \'\' }
who=$(id -un)
[ "$run" = "$SSH_ORIGINAL_COMMAND" ] || {
    who=root
    echo "$SSH_ORIGINAL_COMMAND" >>"$WS_STANDIN/sudo.log"
}
echo "$who: $run" >>"$WS_STANDIN/log"
echo "$SSH_CONNECTION" >>"$WS_STANDIN/conns"
key=$(awk '$1 == "publickey" { print $3 }' "${SSH_USER_AUTH:-/dev/null}")
line=$(awk -v key="$key" 'key != "" && index($0, key) { print; exit }' \
    ~/.ssh/authorized_keys 2>/dev/null)
case $line in
*'command="'*)
    echo 'Please login as the user of the installer rather than the user "root".'
    exit 142
    ;;
esac
PATH=$WS_STANDIN/bin:$PATH
export PATH
case $SSH_ORIGINAL_COMMAND in
'sh -s -- '* | 'sudo '*'sh -s -- '*)
    sed "s|^\([a-z]*=\)/|\1$WS_STANDIN/root/|" | /bin/sh -c "$SSH_ORIGINAL_COMMAND"
    ;;
/root/*) exec "$WS_STANDIN/root$SSH_ORIGINAL_COMMAND" ;;
/var/lib/ws-wizard/*) exec /bin/sh -c "$WS_STANDIN/root$SSH_ORIGINAL_COMMAND" ;;
'/usr/local/bin/ws '* | 'sudo '*' /usr/local/bin/ws '*)
    command=$(printf '%s\n' "$SSH_ORIGINAL_COMMAND" | sed -e "s|^/usr/local/bin/ws |$WS_STANDIN/root&|" \
        -e "s| /usr/local/bin/ws | $WS_STANDIN/root/usr/local/bin/ws |")
    exec /bin/sh -c "$command"
    ;;
"sh -c '"*)
    command=$(printf '%s\n' "$SSH_ORIGINAL_COMMAND" |
        sed "s| /root/| $WS_STANDIN/root/root/|; s| /var/lib/ws-wizard/| $WS_STANDIN/root/var/lib/ws-wizard/|")
    exec /bin/sh -c "$command"
    ;;
*) exec /bin/sh -c "$SSH_ORIGINAL_COMMAND" ;;
esac
EOF
    chmod 755 "$dir/ws.sh"
    # ws's programs that the remote script runs, as stand-ins, in bin and where root's claude is.
    mkdir -p "$dir/bin" "$dir/units"
    for each in systemd-run systemctl journalctl wg mokutil modinfo ip curl getent runuser sudo; do
        ln -s "$here/ws-standin.sh" "$dir/bin/$each"
    done
    : >"$dir/run.log"
    : >"$dir/server.log"
    : >"$dir/install.log"
    : >"$dir/sudo.log"
    sudo mkdir -p "$dir/root/root/.local/bin" "$dir/root/usr/local/bin"
    sudo ln -s "$here/ws-standin.sh" "$dir/root/usr/local/bin/ws"
    sudo ln -s "$here/ws-standin.sh" "$dir/root/root/.local/bin/claude"
    # ws's claude, root's, logged in, and ws's signing key, as the shell role and the stages leave them.
    # root's git config names that key, and the stand-in gh lists it. So no run of the wizard opens a
    # session on ws for a login.
    ssh-keygen -q -t ed25519 -N '' -C "$1" -f "$dir/signing" </dev/null
    sudo mkdir -p "$dir/root/root/.ssh"
    sudo cp "$dir/signing.pub" "$dir/root/root/.ssh/id_ed25519_signing.pub"
    sudo git config -f "$dir/root/root/.gitconfig" user.signingkey \
        "$dir/root/root/.ssh/id_ed25519_signing.pub"
    cut -d ' ' -f 1-2 "$dir/signing.pub" >>"$RUNNER_TEMP/gh/signing"
    # The runner's user has its own home on ws, with the same files, as the remote script reads the
    # claude and git config of the login's user. root's home serves a cloud VM's login.
    home=$dir/root/home/$(id -un)
    sudo mkdir -p "$home/.local/bin" "$home/.ssh"
    sudo ln -s "$here/ws-standin.sh" "$home/.local/bin/claude"
    sudo cp "$dir/signing.pub" "$home/.ssh/id_ed25519_signing.pub"
    sudo git config -f "$home/.gitconfig" user.signingkey "$home/.ssh/id_ed25519_signing.pub"
    sudo chown -R "$(id -un)" "$home"
    touch "$dir/claude"
    # sudo asks no password until a case removes sudo.open.
    touch "$dir/sudo.open"
    : >"$dir/log"
    : >"$dir/conns"
    sudo /usr/sbin/sshd -f "$dir/sshd_config"
}
standin ws 22022
# A second server, lab, as 62-second.sh sets one up on a Mac that has ws. Its sshd listens at
# 22023, and its wg0 on the port that its host_vars names, 51821, beside ws's 51820.
standin lab 22023
ws=$RUNNER_TEMP/ws
lab=$RUNNER_TEMP/lab
# ssh first on the wizard's PATH, as the stand-in ws's wg0 has no address in the tunnel. The Tunnel
# stage logs in to ws through it with ssh ws echo ok, which goes to the stand-in's sshd where ws
# client macos tunnel has a handshake. It lands in tunnel.log. A direct call, which carries
# AddressFamily=inet, as Login's login, and any other call is the runner's ssh.
cat >/opt/homebrew/bin/ssh <<EOF
#!/bin/sh
case " \$* " in
*' AddressFamily=inet '*) ;;
*' ws echo ok ')
    sudo /usr/local/libexec/ws-tunnel/ws client macos tunnel status --server ws 2>/dev/null |
        grep -q '^latest handshake: [0-9]' ||
        { echo 'ssh: no handshake in the tunnel' >&2 && exit 255; }
    echo "\$*" >>'$ws/tunnel.log'
    exec /usr/bin/ssh -o HostName=127.0.0.1 -o Port=22022 "\$@"
    ;;
*' lab echo ok ')
    sudo /usr/local/libexec/ws-tunnel/ws client macos tunnel status --server lab 2>/dev/null |
        grep -q '^latest handshake: [0-9]' ||
        { echo 'ssh: no handshake in the tunnel' >&2 && exit 255; }
    echo "\$*" >>'$lab/tunnel.log'
    exec /usr/bin/ssh -o HostName=127.0.0.1 -o Port=22023 "\$@"
    ;;
esac
exec /usr/bin/ssh "\$@"
EOF
chmod 755 /opt/homebrew/bin/ssh
# nc next, for the Tunnel stage's check of ws's SSH port through the tunnel, by its arguments. It
# goes to the stand-in's sshd, by the runner's nc, where ws client macos tunnel has a handshake.
rm -f /opt/homebrew/bin/nc
cat >/opt/homebrew/bin/nc <<'EOF'
#!/bin/sh
case $* in
'-z -G 3 -w 3 10.99.0.1 22') server=ws port=22022 ;;
'-z -G 3 -w 3 10.99.1.1 22') server=lab port=22023 ;;
*) exec /usr/bin/nc "$@" ;;
esac
sudo /usr/local/libexec/ws-tunnel/ws client macos tunnel status --server "$server" 2>/dev/null |
    grep -q '^latest handshake: [0-9]' && exec /usr/bin/nc -z -G 3 -w 3 127.0.0.1 "$port"
EOF
chmod 755 /opt/homebrew/bin/nc
# The releases of ws, at 127.0.0.1:22080, which WS_RELEASES_URL names job-wide. The job's build is
# release v0.0.0, the latest, so that --update finds this ws and stays. A case that adds another
# release writes the tag to latest. The server goes on after this step.
releases=$RUNNER_TEMP/releases
mkdir -p "$releases/download/v0.0.0"
cp ~/.local/bin/ws "$releases/download/v0.0.0/ws-darwin-arm64"
cp ~/repository/ws/install.sh "$releases/download/v0.0.0/install.sh"
(cd "$releases/download/v0.0.0" && shasum -a 256 ws-darwin-arm64 >ws.sha256)
echo v0.0.0 >"$releases/latest"
(cd ~/repository/ws && "$(sh .tools/run -path go)" build -o "$RUNNER_TEMP/serve-releases" ./.tools/serve-releases)
nohup "$RUNNER_TEMP/serve-releases" "$releases" 22080 >"$RUNNER_TEMP/releases.log" 2>&1 &
for _ in $(seq 50); do
    curl -fsS -o /dev/null -w '%{redirect_url}' http://127.0.0.1:22080/latest >/dev/null 2>&1 && break
    sleep 0.1
done
curl -fsS -o /dev/null -w '%{redirect_url}' http://127.0.0.1:22080/latest | grep -qx 'http://127.0.0.1:22080/tag/v0.0.0'
