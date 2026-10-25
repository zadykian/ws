#!/bin/sh
# The stand-ins of the wizard's job, made once before checks.sh runs the cases: what a new Mac and
# its user give the wizard, on GitHub's macOS runner.

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
# which gh api answers with a 404. keys holds its login's keys, and log its calls.
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
'api --paginate user/ssh_signing_keys --jq .[].key') ;;
'ssh-key add '*) awk 'NF { print \$1, \$2; exit }' "\$3" >>"\$state/keys" ;;
*) echo "gh: the stand-in takes no \$*" >&2 && exit 2 ;;
esac
EOF
chmod 755 /opt/homebrew/bin/gh
