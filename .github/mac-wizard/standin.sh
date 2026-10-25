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
# key is the file ~/.ssh/id_ed25519, named, as a run with no terminal to ask for another needs it.
mkdir -p ~/.config/ws
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 'WS_IDENTITY=~/.ssh/id_ed25519.pub' \
    >~/.config/ws/mac.conf
# dash, which macOS has as /bin/dash, from Homebrew where the runner lacks it.
command -v dash >/dev/null || brew install dash
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
# login's keys, and log its calls.
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
*) echo "gh: the stand-in takes no \$*" >&2 && exit 2 ;;
esac
EOF
chmod 755 /opt/homebrew/bin/gh
# ws's stand-in: an sshd of its own at 127.0.0.1:22022, where mac.conf's WS_REMOTE and its port
# lead, with a host key made here. The cases answer the Host key stage with its fingerprint. Root
# logs in by root's own authorized_keys, and the runner's user, as the user of Ubuntu's installer,
# by the stand-in gh's keys, which that installer would import. Each session runs ws.sh. A second
# host key, ECDSA, as ws has more than one, must stay out of known_hosts at each login.
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
printf '%s\n' "Match User $USER" "    AuthorizedKeysFile $RUNNER_TEMP/gh/keys" >>"$ws/sshd_config"
# ws.sh, ws's shell: it logs each command in log, which root and the user both write, then runs it.
# Where root's authorized_keys gives the key a command, as cloud-init's does, ws.sh stops as that
# command does, since sshd's ForceCommand runs in its place.
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
exec /bin/sh -c "$SSH_ORIGINAL_COMMAND"
EOF
chmod 755 "$ws/ws.sh"
: >"$ws/log"
chmod 666 "$ws/log"
sudo /usr/sbin/sshd -f "$ws/sshd_config"
