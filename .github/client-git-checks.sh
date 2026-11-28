# shellcheck shell=bash
# The client job's checks of mise bootstrap's units of git. They read /etc/ws/gitconfig, its allowed
# signers, Secretive's wrapper, the include in ~/.gitconfig and mise's gh in /usr/local/bin.
# 56-keys.sh sources them after ws's key's, so that they run with its functions. The job's
# values.yml names a file key as git's. The runner's ~/.gitconfig goes aside, and comes back as the case
# exits, on a signal too, with the job's files.
#
# shellcheck disable=SC2154 # tmp, restores and the functions are 56-keys.sh's
gitconfig=/etc/ws/gitconfig
keygen=/etc/ws/secretive-ssh-keygen
signers=/etc/ws/allowed_signers
secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
tab=$(printf '\t')

# pubname PUB prints the name that Secretive gives the .pub, and ws's copy keeps: its MD5 in hex.
pubname() {
    ssh-keygen -E md5 -l -f "$1" | sed -e 's/^[0-9]* MD5://' -e 's/ .*//' -e 's/://g' -e 's/$/.pub/'
}

# git_restore stops the stand-in agent, removes the files that git's cases made, and puts the
# runner's ~/.gitconfig back. The job's values.yml then gives the job's files at the next bootstrap.
git_restore() {
    [ -z "${agent-}" ] || kill "$agent" || true
    rm -f ~/.gitconfig.*~
    chmod -R u+rwx "$secretive" 2>/dev/null || true
    rm -rf "$secretive"
    sudo rm -rf /etc/ws/secretive
    [ -f "$tmp/gitconfig" ] || return 0
    rm -f ~/.gitconfig
    mv "$tmp/gitconfig" ~/.gitconfig
}
restores="$restores git_restore"
[ ! -f ~/.gitconfig ] || cp ~/.gitconfig "$tmp/gitconfig"
# The job's bootstraps had git's key, so they must have included /etc/ws/gitconfig at the top of
# ~/.gitconfig.
[ "$(head -n 2 ~/.gitconfig)" = "[include]$nl${tab}path = /etc/ws/gitconfig" ] ||
    error "mise bootstrap did not include /etc/ws/gitconfig at the top of ~/.gitconfig"
rm -f ~/.gitconfig ~/.gitconfig.*~

# signed makes a commit in a new repository. Git must sign it with the key as ws's name and
# email, in internal/gitcfg. It must verify it with the allowed signers, which name the email.
signed() {
    rm -rf "$tmp/repo"
    git init -q "$tmp/repo"
    git -C "$tmp/repo" commit -q --allow-empty -m ws || error "git cannot make a signed commit"
    got=$(git -C "$tmp/repo" log -1 --format='%an <%ae> %G? %GS') || true
    [ "$got" = 'zadykian <me@zadykian.dev> G me@zadykian.dev' ] ||
        error "git log shows '$got' for the commit"
}

# gh: mise's gh must stand in /usr/local/bin as root's, 0755, and replace a gh that was there.
# With the files in place, a bootstrap must call sudo not once.
gh=$(cd / && mise -C /opt/ws/macos which gh) ||
    error "mise has no gh for /opt/ws"
printf '#!/bin/sh\n' | sudo tee /usr/local/bin/gh >/dev/null
boots
cmp -s /usr/local/bin/gh "$gh" || error "/usr/local/bin/gh is not mise's $gh"
owner=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/gh)
[ "$owner" = "root:wheel 755" ] || error "/usr/local/bin/gh is $owner, not root:wheel 755"
client_mark
boots
calls=$(client_calls)
[ "$calls" = 0 ] || error "a bootstrap that changes nothing called sudo $calls times, not once"

# With a file key with no passphrase as WS_GIT_SIGNING_PUB_KEY, the bootstrap must write
# ~/.gitconfig with the include alone. git's credential command for GitHub must be the gh in
# /usr/local/bin, and git must sign and verify a commit.
[ "$(cat ~/.gitconfig)" = "[include]$nl${tab}path = /etc/ws/gitconfig" ] ||
    error "mise bootstrap wrote ~/.gitconfig as '$(cat ~/.gitconfig)', not the include alone"
for host in github.com gist.github.com; do
    [ "$(git config --global --includes --get-all "credential.https://$host.helper")" = \
        "$nl!/usr/local/bin/gh auth git-credential" ] ||
        error "git's credential command for $host is not /usr/local/bin/gh's"
done
[ ! -e "$keygen" ] || error "mise bootstrap wrote $keygen for a key in a file"
grep -q "^me@zadykian.dev $(cut -d ' ' -f 1,2 ~/.ssh/id_ed25519.pub)" "$signers" ||
    error "$signers does not trust the job's key for git's email"
signed

# A ~/.gitconfig of the user's own must get the include at its top, once, and keep its own lines,
# its mode and the old file.
printf '[user]\n\temail = someone@example.com\n[commit]\n\tgpgSign = false\n' >~/.gitconfig
chmod 600 ~/.gitconfig
cp ~/.gitconfig "$tmp/own"
boots
[ "$(git config --file ~/.gitconfig --get-all include.path)" = /etc/ws/gitconfig ] ||
    error "mise bootstrap left ~/.gitconfig with includes other than one of /etc/ws/gitconfig"
tail -n +3 ~/.gitconfig | grep -v '^$' | cmp -s - "$tmp/own" || error "mise bootstrap lost ~/.gitconfig's own lines"
[ -n "$(find ~/.gitconfig -perm 600)" ] || error "mise bootstrap changed ~/.gitconfig's mode 600"
cmp -s ~/.gitconfig.*~ "$tmp/own" || error "mise bootstrap kept no copy of the old ~/.gitconfig"
# One include below the file's own settings, or one at the top and again below them, must stop the
# bootstrap, as must a link.
printf '[user]\n\tname = someone\n[include]\n\tpath = /etc/ws/gitconfig\n' >~/.gitconfig
refuses "includes /etc/ws/gitconfig below its own settings"
printf '[include]\n\tpath = %s\n[user]\n\tname = someone\n[include]\n\tpath = %s\n' \
    /etc/ws/gitconfig /etc/ws/gitconfig >~/.gitconfig
refuses "includes /etc/ws/gitconfig 2 times: keep the one at the top"
printf '[user]\n\tname = someone\n' >"$tmp/linked"
rm ~/.gitconfig
ln -s "$tmp/linked" ~/.gitconfig
refuses "$HOME/.gitconfig is a link to $tmp/linked"
rm -f ~/.gitconfig ~/.gitconfig.*~
# An empty WS_GIT_SIGNING_PUB_KEY must stop mise bootstrap, with no line to add while Secretive's
# agent is not there. So must a key that is missing, and a .pub that holds more than a public
# key's line, as a private key does.
with 'WS_GIT_SIGNING_PUB_KEY: ""'
refuses "values.yml names no WS_GIT_SIGNING_PUB_KEY, git's signing key: make a P-256 key in Secretive, then name its .pub there"
with 'WS_GIT_SIGNING_PUB_KEY: ~/.ssh/ws.pub'
refuses "$HOME/.ssh/ws.pub, values.yml's WS_GIT_SIGNING_PUB_KEY, is missing"
cp ~/.ssh/id_ed25519 "$tmp/mixed-git.pub"
with "WS_GIT_SIGNING_PUB_KEY: $tmp/mixed-git.pub"
refuses "$tmp/mixed-git.pub holds no public key alone"

# A key of Secretive's, held by an agent at its socket, named as ws's copy of its .pub, with
# PublicKeys closed, as on macOS 27. The bootstrap must write the wrapper, root's and 0755, and
# name it as gpg.ssh.program. git must sign with the agent, through the wrapper, whatever
# SSH_AUTH_SOCK names.
mkdir -p "$secretive/Data/PublicKeys" && chmod 000 "$secretive/Data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/p256-git"
copy=/etc/ws/secretive/$(pubname "$tmp/p256-git.pub")
agent=$(ssh-agent -s -a "$secretive/Data/socket.ssh" | sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$agent" ] || ! SSH_AUTH_SOCK=$secretive/Data/socket.ssh ssh-add -q "$tmp/p256-git"; then
    error "no ssh-agent at Secretive's socket took the key"
    exit 1
fi
rm "$tmp/p256-git"
with "WS_GIT_SIGNING_PUB_KEY: $copy"
boots
cmp -s "$copy" "$tmp/p256-git.pub" || error "mise bootstrap wrote no $copy"
[ "$(stat -f '%Su:%Sg %Lp' "$keygen" 2>/dev/null)" = "root:wheel 755" ] ||
    error "$keygen is missing, or not root:wheel 755"
grep -qxF "${tab}program = \"$keygen\"" "$gitconfig" || error "$gitconfig names no gpg.ssh.program $keygen"
(SSH_AUTH_SOCK=$tmp/none && export SSH_AUTH_SOCK && signed && exit "$fail") || fail=1
# ws keeps seward's login key as ws's, though Secretive's one key is git's.
[ "$(ssh -T -G ws | awk '$1 == "identityfile" { print $2 }' | sed "s|^$HOME/|~/|")" = \~/.ssh/ws-auth.pub ] ||
    error "ssh -G ws gives git's signing key, Secretive's one key, as ws's"
# With the job's file key again, the wrapper must go.
git_restore
agent=
with
boots
[ ! -e "$keygen" ] || error "mise bootstrap left $keygen for a key in a file"
