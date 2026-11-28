# shellcheck shell=bash
# The client job's checks of mise bootstrap's units of git. They read /etc/ws/gitconfig, its allowed
# signers, the wrapper that gives ssh-keygen seward's agent, the include in ~/.gitconfig and mise's gh
# in /usr/local/bin. 56-keys.sh sources them after ws's key's, so that they run with its functions.
# Git signs with seward's ~/.ssh/ws-sign. The runner's ~/.gitconfig goes
# aside, and comes back as the case exits, on a signal too, with the job's files.
#
# shellcheck disable=SC2154 # tmp, restores and the functions are 56-keys.sh's
gitconfig=/etc/ws/gitconfig
keygen=/etc/ws/ssh-keygen
signers=/etc/ws/allowed_signers
tab=$(printf '\t')

# git_restore removes the files that git's cases made, and puts the runner's ~/.gitconfig back.
git_restore() {
    rm -f ~/.gitconfig.*~
    [ -f "$tmp/gitconfig" ] || return 0
    rm -f ~/.gitconfig
    mv "$tmp/gitconfig" ~/.gitconfig
}
restores="$restores git_restore"
[ ! -f ~/.gitconfig ] || cp ~/.gitconfig "$tmp/gitconfig"
# The job's bootstraps must have included /etc/ws/gitconfig at the top of ~/.gitconfig.
[ "$(head -n 2 ~/.gitconfig)" = "[include]$nl${tab}path = /etc/ws/gitconfig" ] ||
    error "mise bootstrap did not include /etc/ws/gitconfig at the top of ~/.gitconfig"
rm -f ~/.gitconfig ~/.gitconfig.*~

# signed makes a commit in a new repository. Git must sign it with the key as ws's name and
# email, in internal/gitcfg. It must verify it with the allowed signers, which name the email.
signed() {
    rm -rf "$tmp/repo"
    git init -q "$tmp/repo"
    git -C "$tmp/repo" commit -q --allow-empty -m ws || error "git cannot make a signed commit"
    git -C "$tmp/repo" verify-commit HEAD 2>/dev/null || error "git verify-commit refuses the signed commit"
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

# The bootstrap must write ~/.gitconfig with the include alone. git's credential command for GitHub
# must be the gh in /usr/local/bin. git must sign a commit with seward's key and verify it, as the
# allowed signers trust the key for git's email.
[ "$(cat ~/.gitconfig)" = "[include]$nl${tab}path = /etc/ws/gitconfig" ] ||
    error "mise bootstrap wrote ~/.gitconfig as '$(cat ~/.gitconfig)', not the include alone"
for host in github.com gist.github.com; do
    [ "$(git config --global --includes --get-all "credential.https://$host.helper")" = \
        "$nl!/usr/local/bin/gh auth git-credential" ] ||
        error "git's credential command for $host is not /usr/local/bin/gh's"
done
grep -qxF "${tab}signingkey = \"$HOME/.ssh/ws-sign.pub\"" "$gitconfig" || error "$gitconfig names no signing key ~/.ssh/ws-sign.pub"
grep -qxF "${tab}program = \"$keygen\"" "$gitconfig" || error "$gitconfig names no gpg.ssh.program $keygen"
[ "$(stat -f '%Su:%Sg %Lp' "$keygen" 2>/dev/null)" = "root:wheel 755" ] ||
    error "$keygen is missing, or not root:wheel 755"
grep -q "^me@zadykian.dev $(cut -d ' ' -f 1,2 ~/.ssh/ws-sign.pub)\$" "$signers" ||
    error "$signers does not trust ws-sign for git's email"
signed
# The wrapper gives ssh-keygen seward's agent, so git signs whatever SSH_AUTH_SOCK names.
(SSH_AUTH_SOCK=$tmp/none && export SSH_AUTH_SOCK && signed && exit "$fail") || fail=1
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
# With the refusals' files gone, the bootstrap passes again and its plan changes nothing.
boots
