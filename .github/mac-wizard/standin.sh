#!/bin/sh
# The stand-ins of the wizard's job, made once before checks.sh runs the cases: what a new Mac, its
# user and ws give the wizard, on GitHub's macOS runner.

set -eu

sw_vers
# setup.sh stops where a directory above root's copies isn't root's alone, as on a Mac whose
# /usr/local an old Homebrew took. Where the runner's image gave /usr/local or its libexec to its
# user, they become root's, as in mac.yml: their contents stay as they are.
for dir in /usr/local /usr/local/libexec; do
    [ -d "$dir" ] || continue
    if [ "$(stat -f '%u %Lp' "$dir")" != "0 755" ]; then
        echo "::warning::$dir was $(stat -f '%Su:%Sg %Lp' "$dir"); made root's, 755"
        sudo chown root:wheel "$dir"
        sudo chmod 755 "$dir"
    fi
done
# mac.conf's answers: ws by the router's name and port alone, as a Mac away from home has it. ws's
# key is the stand-in YubiKey's, below.
mkdir -p ~/.config/ws
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 >~/.config/ws/mac.conf
# dash, which macOS has as /bin/dash, from Homebrew where the runner lacks it.
command -v dash >/dev/null || brew install dash
# The stand-in YubiKey, whose ykman goes first on PATH, for the cases' PATH below and their step.
sh "$(dirname "$0")/../yubikey-standin.sh"
PATH=$RUNNER_TEMP/yubikey/bin:$PATH
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
# A stand-in JetBrains Gateway.app, an empty directory, spares the runs the cask's download, as in
# mac.yml.
mkdir -p ~/Applications/"JetBrains Gateway.app"
# A stand-in gh, logged in to github.com, for the runner's, which has no login. The wizard puts
# /opt/homebrew/bin first on PATH, so the stand-in replaces the runner's link there. Its state is in
# $RUNNER_TEMP/gh: login is there while it has one, and noscope where its login lacks the scopes,
# which gh api answers with a 404. nosign gives that 404 for the signing keys alone. keys holds its
# login's keys, and log its calls. Its repo clone makes a repository with .githooks and an empty
# Brewfile, and its URL on GitHub as origin.
mkdir -p "$RUNNER_TEMP/gh"
: >"$RUNNER_TEMP/gh/keys"
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
    ;;
'ssh-key add '*) awk 'NF { print \$1, \$2; exit }' "\$3" >>"\$state/keys" ;;
'repo clone '*' -- -q')
    git init -q -b main "\$4" && git -C "\$4" remote add origin "https://github.com/\$3.git" &&
        mkdir "\$4/.githooks" && printf '#!/bin/sh\n' >"\$4/.githooks/pre-commit" &&
        : >"\$4/Brewfile" && git -C "\$4" add . &&
        git -C "\$4" -c user.name=ws -c user.email=ws@example.com commit -q -m "\$3"
    ;;
*) echo "gh: the stand-in takes no \$*" >&2 && exit 2 ;;
esac
EOF
chmod 755 /opt/homebrew/bin/gh
# ws's stand-in: an sshd of its own at 127.0.0.1:22022, where mac.conf's WS_REMOTE and its port
# lead, with a host key made here. The cases answer the Host key stage with its fingerprint. Root
# logs in by root's own authorized_keys, which the cases fill from the stand-in gh's keys, as
# Ubuntu's installer imports the account's keys for its user and the README's step copies them to
# root. Each session runs ws.sh. A second host key, ECDSA, as ws has more than one, must stay out
# of known_hosts at each login.
ws=$RUNNER_TEMP/ws
mkdir -p "$ws"
ssh-keygen -q -t ed25519 -N '' -C ws -f "$ws/host_key" </dev/null
ssh-keygen -q -t ecdsa -N '' -C ws -f "$ws/host_ecdsa" </dev/null
ssh-keygen -l -f "$ws/host_key.pub" | awk '{ print $2 }' >"$ws/fingerprint"
printf '%s\n' 'ListenAddress 127.0.0.1:22022' "HostKey $ws/host_key" "PidFile $ws/sshd.pid" \
    'PasswordAuthentication no' 'KbdInteractiveAuthentication no' \
    'PermitRootLogin prohibit-password' 'UsePAM no' 'StrictModes no' 'ExposeAuthInfo yes' \
    "SetEnv WS_STANDIN=$ws" "ForceCommand $ws/ws.sh" "HostKey $ws/host_ecdsa" >"$ws/sshd_config"
# sshd 9.8 and later refuse an address for a while after connections that don't log in, which the
# cases make in a row. The stand-in takes them, where its sshd knows the setting.
! sudo /usr/sbin/sshd -G -f "$ws/sshd_config" -o PerSourcePenalties=no >/dev/null 2>&1 ||
    echo 'PerSourcePenalties no' >>"$ws/sshd_config"
# ws.sh, ws's shell: it logs each command in log, which root writes, then runs it.
# Where root's authorized_keys gives the key a command, as cloud-init's does, ws.sh stops as that
# command does, since sshd's ForceCommand runs in its place. It puts ws's programs first on PATH,
# and roots the paths of the wizard's scripts on ws, each a line NAME=PATH of their own, in root
# here. So it roots a command's path from /root, such as another repository's apply.sh, alone or
# after the script of sh -c.
cat >"$ws/ws.sh" <<'EOF'
#!/bin/sh
echo "$(id -un): $SSH_ORIGINAL_COMMAND" >>"$WS_STANDIN/log"
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
'sh -s -- '*) sed "s|^\([a-z]*=\)/|\1$WS_STANDIN/root/|" | /bin/sh -c "$SSH_ORIGINAL_COMMAND" ;;
/root/*) exec "$WS_STANDIN/root$SSH_ORIGINAL_COMMAND" ;;
"sh -c '"*)
    command=$(printf '%s\n' "$SSH_ORIGINAL_COMMAND" | sed "s| /root/| $WS_STANDIN/root/root/|")
    exec /bin/sh -c "$command"
    ;;
*) exec /bin/sh -c "$SSH_ORIGINAL_COMMAND" ;;
esac
EOF
chmod 755 "$ws/ws.sh"
# ws's programs that the remote script runs, as stand-ins, in bin and where root's claude is. ws's
# clone, root's, holds the stand-in bootstrap.sh alone.
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$ws/bin" "$ws/units"
for each in systemd-run systemctl journalctl wg gh mokutil modinfo ip; do
    ln -s "$here/ws-standin.sh" "$ws/bin/$each"
done
: >"$ws/run.log"
: >"$ws/bootstrap.log"
clone=$ws/root/root/repository/ws
sudo mkdir -p "$ws/root/root/.local/bin" "$clone"
sudo ln -s "$here/ws-standin.sh" "$ws/root/root/.local/bin/claude"
sudo ln -s "$here/ws-standin.sh" "$clone/bootstrap.sh"
sudo git -C "$clone" init -q -b main
sudo git -C "$clone" add bootstrap.sh
sudo git -C "$clone" -c user.name=ws -c user.email=ws@example.com commit -q -m ws
# Its origin is itself, so that the pull of --update finds nothing new.
sudo git -C "$clone" remote add origin "$clone"
sudo git -C "$clone" fetch -q origin
sudo git -C "$clone" branch -q -u origin/main
# ws's gh and claude, root's, logged in, and ws's signing key, which root's git config names and
# gh's account lists, as the shell role and the logins' stages leave them. So no run of the wizard
# opens a session on ws for a login.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$ws/signing" </dev/null
sudo mkdir -p "$ws/root/root/.ssh"
sudo cp "$ws/signing.pub" "$ws/root/root/.ssh/id_ed25519_signing.pub"
sudo git config -f "$ws/root/root/.gitconfig" user.signingkey \
    "$ws/root/root/.ssh/id_ed25519_signing.pub"
cut -d ' ' -f 1-2 "$ws/signing.pub" >"$ws/gh.keys"
touch "$ws/gh" "$ws/claude"
# ssh first on the wizard's PATH, as the stand-in ws's wg0 has no address in the tunnel. The Tunnel
# stage's login to ws through it, ssh ws echo ok, goes to the stand-in's sshd where ws-tunnel has a
# handshake, and lands in tunnel.log. Any other call is the runner's ssh.
cat >/opt/homebrew/bin/ssh <<EOF
#!/bin/sh
case " \$* " in
*' ws echo ok ')
    sudo /usr/local/libexec/ws-tunnel/ws-tunnel status 2>/dev/null |
        grep -q '^latest handshake: [0-9]' ||
        { echo 'ssh: no handshake in the tunnel' >&2 && exit 255; }
    echo "\$*" >>'$ws/tunnel.log'
    exec /usr/bin/ssh -o HostName=127.0.0.1 -o Port=22022 "\$@"
    ;;
esac
exec /usr/bin/ssh "\$@"
EOF
chmod 755 /opt/homebrew/bin/ssh
# nc next, for the Tunnel stage's check of ws's SSH port through the tunnel, by its arguments. It
# goes to the stand-in's sshd, by the runner's nc, where ws-tunnel has a handshake.
rm -f /opt/homebrew/bin/nc
cat >/opt/homebrew/bin/nc <<'EOF'
#!/bin/sh
[ "$*" = '-z -G 3 -w 3 10.99.0.1 22' ] || exec /usr/bin/nc "$@"
sudo /usr/local/libexec/ws-tunnel/ws-tunnel status 2>/dev/null |
    grep -q '^latest handshake: [0-9]' && exec /usr/bin/nc -z -G 3 -w 3 127.0.0.1 22022
EOF
chmod 755 /opt/homebrew/bin/nc
: >"$ws/log"
sudo /usr/sbin/sshd -f "$ws/sshd_config"
