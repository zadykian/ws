# shellcheck shell=sh
# The mac job's checks of ws client macos apply's git step. mac-setup-checks.sh sources them last,
# so that they run with its functions and the job's values.yml. The runner must lack the files the
# step writes, but ~/.gitconfig, which goes aside. The job's own git files go aside too, as its conf
# with git's keys does. As the script exits, on a signal too, the files go, and the job's files and
# the runner's ~/.gitconfig come back.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
key=$HOME/.ssh/id_ed25519_signing
files=$HOME/.config/ws
signers=/etc/ws/allowed_signers
secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
tab=$(printf '\t')

# git_made COMMAND... runs COMMAND with the files that git's cases make as its last arguments.
git_made() {
    "$@" "$key" "$key.pub" "$files/gitconfig" ~/.gitconfig.*~ \
        "$files/secretive-ssh-keygen" "$secretive"
}

# git_restore removes the files that git's cases made, and puts the runner's ~/.gitconfig back. It
# leaves ~/.gitconfig alone until the runner's has gone aside.
git_restore() {
    [ -z "${agent-}" ] || kill "$agent" || true
    git_made rm -rf
    sudo rm -f "$signers"
    [ ! -f "$tmp/job-git/gitconfig" ] || mv "$tmp/job-git/gitconfig" "$files/gitconfig"
    [ ! -f "$tmp/job-git/allowed_signers" ] || sudo mv "$tmp/job-git/allowed_signers" "$signers"
    for each in "$tmp"/job-git/.gitconfig.*~; do
        [ ! -f "$each" ] || mv "$each" ~/
    done
    [ -f "$tmp/gitconfig" ] || [ -n "${gitconfig_aside-}" ] || return 0
    rm -f ~/.gitconfig
    [ ! -f "$tmp/gitconfig" ] || mv "$tmp/gitconfig" ~/.gitconfig
}

# The job's runs had git's keys, so they made ~/.config/ws's files, which go aside as ~/.gitconfig
# does. So do the copies of the runner's ~/.gitconfig that those runs left beside it. The cases then
# start from a conf with no git key, and the job's conf comes back at the end.
# shellcheck disable=SC2034 # prints, in mac-setup-checks.sh, reads it
exact=1
mkdir "$tmp/job-git"
[ ! -f "$files/gitconfig" ] || mv "$files/gitconfig" "$tmp/job-git/gitconfig"
[ ! -f "$signers" ] || sudo mv "$signers" "$tmp/job-git/allowed_signers"
for each in ~/.gitconfig.*~; do
    [ ! -f "$each" ] || mv "$each" "$tmp/job-git/"
done
found=$(git_made ls -d "$signers" 2>/dev/null) || true
if [ -n "$found" ]; then
    error "the runner has what git's cases would make: $(echo "$found" | tr '\n' ' ')"
    exit 1
fi
# The job's runs before these had git's keys, so they must have included the file in ~/.gitconfig.
grep -qs config/ws/gitconfig ~/.gitconfig ||
    error "ws client macos apply did not include ~/.config/ws/gitconfig in ~/.gitconfig with git's keys"
mv "$tmp/job.conf" "$tmp/job-git.conf"
grep -v '^WS_GIT_' "$tmp/job-git.conf" >"$tmp/job.conf" || true
trap 'git_restore; [ ! -f "$tmp/job-git.conf" ] || conf_write "$conf" <"$tmp/job-git.conf"; rm -rf "$tmp"' EXIT
[ ! -f ~/.gitconfig ] || mv ~/.gitconfig "$tmp/gitconfig"
gitconfig_aside=1
# git's XDG file is the cases' own, so that the runner's changes nothing.
XDG_CONFIG_HOME=$tmp/xdg
export XDG_CONFIG_HOME

# signed makes a commit in a new repository. Git must sign it with the key as ws's name and
# email, in roles/internal/gitcfg. It must verify it with the allowed signers, which name the email.
signed() {
    rm -rf "$tmp/repo"
    git init -q "$tmp/repo"
    git -C "$tmp/repo" commit -q --allow-empty -m ws || error "git cannot make a signed commit"
    got=$(git -C "$tmp/repo" log -1 --format='%an <%ae> %G? %GS') || true
    [ "$got" = 'zadykian <me@zadykian.dev> G me@zadykian.dev' ] ||
        error "git log shows '$got' for the commit"
}

# warned PUB prints ws client macos apply's warning that gh, logged out, cannot register the key
# PUB.
warned() {
    echo "ws client macos apply: warning: gh has no login: gh auth login -h github.com -s" \
        "write:ssh_signing_key logs in. Until a later run adds $1, GitHub shows the commits it" \
        "signs as unverified"
}

# The stand-in gh, in $tmp/bin, is first on PATH. See mac-gh.sh for its files in $tmp.
sh .github/mac-gh.sh "$tmp"
if [ "$(command -v gh)" != "$tmp/bin/gh" ]; then
    error "gh is $(command -v gh), not the stand-in, so git's cases go"
    exit 1
fi

# With a file key with no passphrase as WS_GIT_SIGNING_PUB_KEY,
# --check must show the files' diffs and the add to gh's login, and write none. A run must write
# the files. It must add the key to gh's login, titled with the Mac's name, and say so. Its
# credential command for GitHub must be gh's, and a second run and --check say the key is listed.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$key"
signing="WS_GIT_SIGNING_PUB_KEY: $key.pub"
with "$signing"
title=$(hostname -s)
add="gh ssh-key add $key.pub --type signing --title $title"
listed="ws client macos apply: git's key is among GitHub's signing keys already"
setup --check || error "ws client macos apply --check failed: $out"
case $out in
*"$nl+++ $files/gitconfig$nl"*"${nl}ws client macos apply: would run $add") ;;
*) error "ws client macos apply --check showed no gitconfig or add: $out" ;;
esac
for each in "$files/gitconfig" "$signers" ~/.gitconfig; do
    [ ! -e "$each" ] || error "ws client macos apply --check made $each"
done
prints "ws client macos apply: create $signers${nl}ws client macos apply: wrote $files/gitconfig${nl}\
ws client macos apply: wrote $HOME/.gitconfig${nl}ws client macos apply: ran $add"
prints "$listed"
prints "$listed" --check
for host in github.com gist.github.com; do
    [ "$(git config --global --includes --get-all "credential.https://$host.helper")" = \
        "$nl!$tmp/bin/gh auth git-credential" ] ||
        error "git's credential command for $host is not gh's"
done
signed
own_warning="ws client macos apply: warning: $HOME/.gitconfig's own settings of user.email, user.name,"
own_warning="$own_warning commit.gpgsign, below the include, win over ~/.config/ws/gitconfig's"
# A ~/.gitconfig of the user's own must get the include at its top, once, and keep the old file and
# its mode. Its own settings of identity and signing, which win, must be named in its order.
# --check must show it all, and change nothing.
printf '[user]\n\temail = someone@example.com\n\tname = someone\n[commit]\n\tgpgSign = false\n' \
    >~/.gitconfig
chmod 600 ~/.gitconfig
cp ~/.gitconfig "$tmp/own"
setup --check || error "ws client macos apply --check failed: $out"
case $out in
"--- $HOME/.gitconfig$nl"*"$nl+[include]$nl"*"$nl$own_warning") ;;
*) error "ws client macos apply --check printed '$out' for a ~/.gitconfig of its own" ;;
esac
cmp -s ~/.gitconfig "$tmp/own" && [ "$(echo ~/.gitconfig.*~)" = "$HOME/.gitconfig.*~" ] ||
    error "ws client macos apply --check changed ~/.gitconfig, or kept a copy"
setup || error "ws client macos apply failed: $out"
case $out in
"ws client macos apply: kept the old $HOME/.gitconfig as $HOME/.gitconfig."*"~${nl}ws client macos apply: wrote $HOME/\
.gitconfig$nl$listed$nl$own_warning") ;;
*) error "ws client macos apply printed '$out' for a ~/.gitconfig of its own" ;;
esac
prints "$listed"
[ "$(git config --file ~/.gitconfig --get-all include.path)" = \~/.config/ws/gitconfig ] ||
    error "ws client macos apply left ~/.gitconfig with includes other than one of ~/.config/ws/gitconfig"
tail -n +4 ~/.gitconfig | cmp -s - ~/.gitconfig.*~ || error "ws client macos apply lost ~/.gitconfig's own lines"
[ -n "$(find ~/.gitconfig -perm 600)" ] || error "ws client macos apply changed ~/.gitconfig's mode 600"
# An include by a path from the home must pass too. One below the file's own settings, or one at the
# top and again below them, must stop ws client macos apply, as must a link that lacks the include.
# Each must stop before the step acts: with another name, the step would write gitconfig. The link
# must stay one, and once its file has the include at its top, ws client macos apply must take it.
printf '[include]\n\tpath = .config/ws/gitconfig\n' >~/.gitconfig
prints "$listed"
with "$signing"
printf '[user]\n\tname = someone\n[include]\n\tpath = ~/.config/ws/gitconfig\n' >~/.gitconfig
stops "ws client macos apply: $HOME/.gitconfig includes ~/.config/ws/gitconfig below its own settings"
printf '[include]\n\tpath = %s\n[user]\n\tname = someone\n[include]\n\tpath = %s\n' \
    \~/.config/ws/gitconfig .config/ws/gitconfig >~/.gitconfig
stops "ws client macos apply: $HOME/.gitconfig includes ~/.config/ws/gitconfig 2 times: keep the one at the top"
printf '[user]\n\tname = someone\n' >"$tmp/linked"
rm ~/.gitconfig
ln -s "$tmp/linked" ~/.gitconfig
stops "ws client macos apply: $HOME/.gitconfig is a link to $tmp/linked: put"
with "$signing"
printf '[include]\n\tpath = ~/.config/ws/gitconfig\n[user]\n\tname = someone\n' >"$tmp/linked"
prints "$listed"
[ -L ~/.gitconfig ] || error "ws client macos apply replaced the link ~/.gitconfig"
rm ~/.gitconfig
# git reads its XDG file before ~/.gitconfig, so ws's settings win over that file's. So
# ws client macos apply must name those of identity and signing as it makes ~/.gitconfig.
mkdir -p "$tmp/xdg/git"
printf '[user]\n\temail = someone@example.com\n' >"$tmp/xdg/git/config"
prints "ws client macos apply: wrote $HOME/.gitconfig$nl$listed${nl}ws client macos apply: warning: $tmp/xdg/git/config's\
 settings of user.email come before the include in ~/.gitconfig, so ~/.config/ws/gitconfig's win\
 over them"
rm "$tmp/xdg/git/config" ~/.gitconfig
# An empty WS_GIT_SIGNING_PUB_KEY must stop ws client macos apply, with no line to add while
# Secretive's agent is not there. So must a key that is missing, and a .pub that holds more than a
# public key's line, as a private key does.
with 'WS_GIT_SIGNING_PUB_KEY: ""'
stops "ws client macos apply: values.yml names no WS_GIT_SIGNING_PUB_KEY, git's signing key: run ws client macos wizard"
with 'WS_GIT_SIGNING_PUB_KEY: ~/.ssh/ws.pub'
stops "ws client macos apply: $HOME/.ssh/ws.pub, values.yml's WS_GIT_SIGNING_PUB_KEY, is missing"
cp "$key" "$tmp/mixed.pub"
with "WS_GIT_SIGNING_PUB_KEY: $tmp/mixed.pub"
stops "ws client macos apply: $tmp/mixed.pub holds no public key alone"
# So must one it cannot read, where the user cannot, as on the runner: root reads it.
cp "$key.pub" "$tmp/unread-git.pub" && chmod 000 "$tmp/unread-git.pub"
if ! cat "$tmp/unread-git.pub" >/dev/null 2>&1; then
    with \
        "WS_GIT_SIGNING_PUB_KEY: $tmp/unread-git.pub"
    stops "ws client macos apply: cannot read $tmp/unread-git.pub"
fi

# A key of Secretive's, held by an agent at its socket, named as ws's copy of its .pub, with
# PublicKeys closed, as on macOS 27. --check must show the wrapper and gpg.ssh.program, and write
# neither. git must sign with the agent, through the wrapper, whatever SSH_AUTH_SOCK names.
mkdir -p "$secretive/Data/PublicKeys" && chmod 000 "$secretive/Data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/p256"
copy=/etc/ws/secretive/$(pubname "$tmp/p256.pub")
agent=$(ssh-agent -s -a "$secretive/Data/socket.ssh" | sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$agent" ] || ! SSH_AUTH_SOCK=$secretive/Data/socket.ssh ssh-add -q "$tmp/p256"; then
    error "no ssh-agent at Secretive's socket took the key"
    exit 1
fi
rm "$tmp/p256"
with 'WS_GIT_SIGNING_PUB_KEY: ""'
stops "$(unnamed WS_GIT_SIGNING_PUB_KEY "git's signing key" "$tmp/p256.pub")"
with \
    "WS_GIT_SIGNING_PUB_KEY: $copy"
setup --check || error "ws client macos apply --check failed: $out"
case $out in
*"+++ $files/secretive-ssh-keygen$nl"*"$nl+${tab}program = \"$files/secretive-ssh-keygen\""*) ;;
*) error "ws client macos apply --check showed no wrapper, or no gpg.ssh.program: $out" ;;
esac
[ ! -e "$files/secretive-ssh-keygen" ] || error "ws client macos apply --check wrote $files/secretive-ssh-keygen"
[ ! -e "$copy" ] || error "ws client macos apply --check wrote $copy"
prints "ws client macos apply: create directory /etc/ws/secretive, owner 0:0, mode 755${nl}ws client macos apply: create $copy${nl}ws client macos apply: wrote $files/secretive-ssh-keygen${nl}\
ws client macos apply: write $signers${nl}ws client macos apply: wrote $files/gitconfig${nl}\
ws client macos apply: wrote $HOME/.gitconfig${nl}ws client macos apply: ran gh ssh-key add $copy --type signing\
 --title $title"
prints "$listed"
prints "$listed" --check
# In a subshell, as macOS's sh keeps a variable set for a function's call after it.
(SSH_AUTH_SOCK=$tmp/none && export SSH_AUTH_SOCK && signed && exit "$fail") || fail=1
# ws keeps the file key that the job's values.yml names, though Secretive's one key is there. The
# agent and Secretive's container then go, so that the cases below give ws no key of theirs.
[ "$(ssh -G ws | awk '$1 == "identityfile" { print $2 }')" = \~/.ssh/id_ed25519 ] ||
    error "ssh -G ws gives git's signing key, Secretive's one key, as ws's"
kill "$agent" || true
agent=
chmod -R u+rwx "$secretive" && rm -rf "$secretive" && sudo rm -rf /etc/ws/secretive

# Logged out, gh must draw a warning at each run, and nothing else where the files are written.
# With gh's login, a 404, the answer to a login without the scope of signing keys, must draw a
# warning that names the scope. Another error must draw its last line, its final period dropped, as
# must an add that leaves the account without the key. Then, where the account lacks the key,
# --check must show the add, a run must add it, and a second run and --check say GitHub lists it.
with "$signing"
rm "$tmp/login"
warning=$(warned "$key.pub")
prints "ws client macos apply: write $signers${nl}ws client macos apply: wrote $files/gitconfig$nl$warning"
prints "$warning"
prints "$warning" --check
touch "$tmp/login"
until="Until a later run adds $key.pub, GitHub shows the commits it signs as unverified"
printf '%s\n' 'Post "https://api.github.com/user/ssh_signing_keys": EOF' 'gh: offline.' \
    >"$tmp/error"
prints "ws client macos apply: warning: gh failed: gh: offline. $until"
echo 'gh: Not Found (HTTP 404)' >"$tmp/error"
prints "ws client macos apply: warning: gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h\
 github.com -s write:ssh_signing_key adds it. $until"
rm "$tmp/error"
: >"$tmp/keys"
touch "$tmp/lost"
prints "ws client macos apply: warning: gh failed: gh api user/ssh_signing_keys lacks the key after $add. $until"
rm "$tmp/lost"
prints "ws client macos apply: would run $add" --check
prints "ws client macos apply: ran $add"
prints "$listed"
prints "$listed" --check
[ "$(grep '^ssh-key add ' "$tmp/gh.log" | tail -n 1)" = "${add#gh }" ] &&
    [ "$(cat "$tmp/keys")" = "$(awk '{ print $1, $2 }' "$key.pub")" ] ||
    error "the stand-in logged another add, or holds other keys, than $key.pub's"

# Where PATH has no gh, ws client macos apply --check must say that Homebrew would install it. It must name the gh at Homebrew's prefix
# as git's credential command, and name the add. A run must install it, once. The stand-in brew's prefix is
# $tmp/brew, where its install puts the stand-in gh, and it passes other commands to the runner's
# brew.
brew=$(command -v brew)
mkdir -p "$tmp/brew/bin"
cat >"$tmp/brew/bin/brew" <<EOF
#!/bin/sh
case \$* in
--prefix) echo '$tmp/brew' ;;
'install gh') echo "\$*" >>'$tmp/brew.log' && cp '$tmp/bin/gh' '$tmp/brew/bin' ;;
*) exec '$brew' "\$@" ;;
esac
EOF
chmod 755 "$tmp/brew/bin/brew"
path=$PATH
PATH=$tmp/brew/bin$(echo "$PATH" | tr : '\n' | while read -r each; do
    [ -x "$each/gh" ] || printf ':%s' "$each"
done)
with "$signing"
setup --check || error "ws client macos apply --check failed: $out"
case $out in
"ws client macos apply: would run brew install gh$nl"*"$nl+${tab}helper = !$tmp/brew/bin/gh auth\
 git-credential$nl"*"${nl}ws client macos apply: would run $add once gh is installed, where it has a login\
 that lacks the key") ;;
*) error "ws client macos apply --check showed no install of gh, another credential command, or no add: $out" ;;
esac
[ ! -f "$tmp/brew.log" ] || error "ws client macos apply installed gh under --check"
prints "ws client macos apply: installing gh${nl}ws client macos apply: wrote $files/gitconfig$nl$warning"
prints "$warning"
[ "$(cat "$tmp/brew.log")" = 'install gh' ] || error "ws client macos apply ran brew install gh other than once"
PATH=$path
