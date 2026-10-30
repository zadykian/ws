# shellcheck shell=sh
# The mac job's checks of mac/setup.sh's git step, which mac-setup-checks.sh sources last, so that
# they run with its functions and the job's mac.conf. The runner must lack the files the step
# writes, but ~/.gitconfig, which goes aside. As the script exits, on a signal too, the files go,
# and the runner's ~/.gitconfig comes back. Last, it sources mac-git-yubikey-checks.sh's cases.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
files=$HOME/.config/ws
pub=$files/keys/git.pub

# git_made COMMAND... runs COMMAND with the files that git's cases make as its last arguments.
git_made() {
    "$@" "$files/gitconfig" "$files/allowed_signers" ~/.gitconfig.*~ "$files/ws-ssh-keygen" \
        "$files/secretive-ssh-keygen"
}

# git_restore removes the files that git's cases made, and puts the runner's ~/.gitconfig back. It
# leaves ~/.gitconfig alone until the runner's has gone aside.
git_restore() {
    [ -z "${agent-}" ] || kill "$agent" || true
    git_made rm -rf
    [ -f "$tmp/gitconfig" ] || [ -n "${gitconfig_aside-}" ] || return 0
    rm -f ~/.gitconfig
    [ ! -f "$tmp/gitconfig" ] || mv "$tmp/gitconfig" ~/.gitconfig
}

found=$(git_made ls -d 2>/dev/null) || true
if [ -n "$found" ]; then
    error "the runner has what git's cases would make: $(echo "$found" | tr '\n' ' ')"
    exit 1
fi
# ws's agent must hold the YubiKey's key for git, whose .pub the job's runs wrote.
if [ ! -s "$pub" ] || ! SSH_AUTH_SOCK=$HOME/.ssh/ws-agent.sock ssh-add -L 2>/dev/null |
    grep -qF "$(cut -d ' ' -f 1-2 "$pub")"; then
    error "ws's agent lacks the YubiKey's key for git, $pub, so git's cases go"
    exit 1
fi
# The job's runs before these had no git keys, so they must have left ~/.gitconfig alone.
! grep -qs config/ws/gitconfig ~/.gitconfig ||
    error "setup.sh included ~/.config/ws/gitconfig in ~/.gitconfig without git's keys"
trap 'git_restore; [ ! -f "$tmp/job.conf" ] || cp "$tmp/job.conf" "$conf"; rm -rf "$tmp"' EXIT
[ ! -f ~/.gitconfig ] || mv ~/.gitconfig "$tmp/gitconfig"
gitconfig_aside=1
# git's XDG file is the cases' own, so that the runner's changes nothing.
XDG_CONFIG_HOME=$tmp/xdg
export XDG_CONFIG_HOME

# signed makes a commit in a new repository, which git must sign with the key as WS_GIT_NAME and
# WS_GIT_EMAIL, and verify with the allowed signers, which name the email.
signed() {
    rm -rf "$tmp/repo"
    git init -q "$tmp/repo"
    git -C "$tmp/repo" commit -q --allow-empty -m ws || error "git cannot make a signed commit"
    got=$(git -C "$tmp/repo" log -1 --format='%an <%ae> %G? %GS') || true
    [ "$got" = 'ws stand-in #1 <ws@example.com> G ws@example.com' ] ||
        error "git log shows '$got' for the commit"
}

# warned PUB prints setup.sh's warning that gh, logged out, cannot register the key PUB.
warned() {
    echo "setup.sh: warning: gh has no login: gh auth login -h github.com -s" \
        "write:ssh_signing_key logs in. Until a later run adds $1, GitHub shows the commits it" \
        "signs as unverified"
}

# A stand-in gh goes in $tmp/bin, first on PATH, logged in while $tmp/login is there. It logs each
# call, and lists and adds the account's signing keys in $tmp/keys, for setup.sh's commands alone.
# With $tmp/error, it fails with that text; with $tmp/lost, an add keeps no key.
: >"$tmp/keys"
touch "$tmp/login"
cat >"$tmp/bin/gh" <<'EOF'
#!/bin/sh
state=${0%/bin/*}
echo "$*" >>"$state/gh.log"
if [ ! -f "$state/login" ]; then
    echo 'To get started with GitHub CLI, please run:  gh auth login' >&2
    exit 4
fi
[ ! -f "$state/error" ] || { cat "$state/error" >&2 && exit 1; }
case $* in
'api --paginate user/ssh_signing_keys --jq .[].key') cat "$state/keys" ;;
'ssh-key add '*) [ -f "$state/lost" ] || awk '{ print $1, $2 }' "$3" >>"$state/keys" ;;
*) echo "gh: the stand-in takes no $*" >&2 && exit 2 ;;
esac
EOF
chmod 755 "$tmp/bin/gh"
if [ "$(command -v gh)" != "$tmp/bin/gh" ]; then
    error "gh is $(command -v gh), not the stand-in, so git's cases go"
    exit 1
fi

# With WS_GIT_NAME and WS_GIT_EMAIL, git's key must be the YubiKey's. --check must show the
# wrapper, the files' diffs, with the wrapper as gpg.ssh.program, and the add to gh's login, and
# write none. A run must write them, add the key to gh's login, titled with the Mac's name, and say
# so. Its credential command for GitHub must be gh's, and a second run and --check print nothing.
# git must sign through the wrapper, whatever SSH_AUTH_SOCK names.
with 'WS_GIT_NAME="ws stand-in #1"' WS_GIT_EMAIL=ws@example.com
title=$(hostname -s)
add="gh ssh-key add $pub --type signing --title $title"
setup --check || error "setup.sh --check failed: $out"
case $out in
*"+++ $files/ws-ssh-keygen$nl"*"$nl+    program = \"$files/ws-ssh-keygen\""*"\
${nl}setup.sh: would run $add") ;;
*) error "setup.sh --check showed no wrapper, gpg.ssh.program or add: $out" ;;
esac
for each in "$files/ws-ssh-keygen" "$files/gitconfig" "$files/allowed_signers" ~/.gitconfig; do
    [ ! -e "$each" ] || error "setup.sh --check made $each"
done
prints "setup.sh: wrote $files/ws-ssh-keygen${nl}setup.sh: wrote $files/allowed_signers${nl}\
setup.sh: wrote $files/gitconfig${nl}setup.sh: wrote $HOME/.gitconfig${nl}setup.sh: ran $add"
prints ''
prints '' --check
# The wrapper for a key of Secretive's that older runs wrote must go, and --check show that as a
# diff. A file of that name that setup.sh did not write must stay.
wrapper=$files/secretive-ssh-keygen
printf '#!/bin/sh\n%s\n' "# Written by ws's mac/setup.sh: git's gpg.ssh.program" >"$wrapper"
prints "$(diff -u -L "$wrapper" -L /dev/null "$wrapper" /dev/null)" --check
prints "setup.sh: removed $wrapper"
echo '#!/bin/sh' >"$wrapper"
prints ''
[ -f "$wrapper" ] || error "setup.sh removed a $wrapper of its own"
rm -f "$wrapper"
for host in github.com gist.github.com; do
    [ "$(git config --global --includes --get-all "credential.https://$host.helper")" = \
        "$nl!$tmp/bin/gh auth git-credential" ] ||
        error "git's credential command for $host is not gh's"
done
# In a subshell, as macOS's sh keeps a variable set for a function's call after it.
(SSH_AUTH_SOCK=$tmp/none && export SSH_AUTH_SOCK && signed && exit "$fail") || fail=1
own_warning="setup.sh: warning: $HOME/.gitconfig's own settings of user.email, user.name,"
own_warning="$own_warning commit.gpgsign, below the include, win over ~/.config/ws/gitconfig's"
# A ~/.gitconfig of the user's own must get the include at its top, once, and keep the old file and
# its mode. Its own settings of identity and signing, which win, must be named in its order.
# --check must show it all, and change nothing.
printf '[user]\n\temail = someone@example.com\n\tname = someone\n[commit]\n\tgpgSign = false\n' \
    >~/.gitconfig
chmod 600 ~/.gitconfig
cp ~/.gitconfig "$tmp/own"
setup --check || error "setup.sh --check failed: $out"
case $out in
"--- $HOME/.gitconfig$nl"*"$nl+[include]$nl"*"$nl$own_warning") ;;
*) error "setup.sh --check printed '$out' for a ~/.gitconfig of its own" ;;
esac
cmp -s ~/.gitconfig "$tmp/own" && [ "$(echo ~/.gitconfig.*~)" = "$HOME/.gitconfig.*~" ] ||
    error "setup.sh --check changed ~/.gitconfig, or kept a copy"
setup || error "setup.sh failed: $out"
case $out in
"setup.sh: kept the old $HOME/.gitconfig as $HOME/.gitconfig."*"~${nl}setup.sh: wrote $HOME/\
.gitconfig$nl$own_warning") ;;
*) error "setup.sh printed '$out' for a ~/.gitconfig of its own" ;;
esac
prints ''
[ "$(git config --file ~/.gitconfig --get-all include.path)" = \~/.config/ws/gitconfig ] ||
    error "setup.sh left ~/.gitconfig with includes other than one of ~/.config/ws/gitconfig"
tail -n +4 ~/.gitconfig | cmp -s - ~/.gitconfig.*~ || error "setup.sh lost ~/.gitconfig's own lines"
[ -n "$(find ~/.gitconfig -perm 600)" ] || error "setup.sh changed ~/.gitconfig's mode 600"
# An include by a path from the home must pass too. One below the file's own settings, or one at the
# top and again below them, must stop setup.sh, as must a link that lacks the include. Each must
# stop before the step acts: with another name, the step would write gitconfig. The link must stay
# one, and once its file has the include at its top, setup.sh must take it.
printf '[include]\n\tpath = .config/ws/gitconfig\n' >~/.gitconfig
prints ''
with 'WS_GIT_NAME="ws stand-in #2"' WS_GIT_EMAIL=ws@example.com
printf '[user]\n\tname = someone\n[include]\n\tpath = ~/.config/ws/gitconfig\n' >~/.gitconfig
stops "setup.sh: $HOME/.gitconfig includes ~/.config/ws/gitconfig below its own settings"
printf '[include]\n\tpath = %s\n[user]\n\tname = someone\n[include]\n\tpath = %s\n' \
    \~/.config/ws/gitconfig .config/ws/gitconfig >~/.gitconfig
stops "setup.sh: $HOME/.gitconfig includes ~/.config/ws/gitconfig 2 times: keep the one at the top"
printf '[user]\n\tname = someone\n' >"$tmp/linked"
rm ~/.gitconfig
ln -s "$tmp/linked" ~/.gitconfig
stops "setup.sh: $HOME/.gitconfig is a link to $tmp/linked: put"
with 'WS_GIT_NAME="ws stand-in #1"' WS_GIT_EMAIL=ws@example.com
printf '[include]\n\tpath = ~/.config/ws/gitconfig\n[user]\n\tname = someone\n' >"$tmp/linked"
prints ''
[ -L ~/.gitconfig ] || error "setup.sh replaced the link ~/.gitconfig"
rm ~/.gitconfig
# git reads its XDG file before ~/.gitconfig, so ws's settings win over that file's: setup.sh must
# name those of identity and signing as it makes ~/.gitconfig.
mkdir -p "$tmp/xdg/git"
printf '[user]\n\temail = someone@example.com\n' >"$tmp/xdg/git/config"
prints "setup.sh: wrote $HOME/.gitconfig${nl}setup.sh: warning: $tmp/xdg/git/config's settings of\
 user.email come before the include in ~/.gitconfig, so ~/.config/ws/gitconfig's win over them"
rm "$tmp/xdg/git/config" ~/.gitconfig
# A WS_GIT_SIGNING_KEY that is missing must stop setup.sh, as must a .pub that holds more than a
# public key's line, as a private key does.
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com 'WS_GIT_SIGNING_KEY=~/.ssh/ws.pub'
stops "setup.sh: $HOME/.ssh/ws.pub, mac.conf's WS_GIT_SIGNING_KEY, is missing"
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/mixed.pub" </dev/null
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com "WS_GIT_SIGNING_KEY=$tmp/mixed.pub"
stops "setup.sh: $tmp/mixed.pub holds no public key alone"
# So must one it cannot read, where the user cannot, as on the runner: root reads it.
cp "$pub" "$tmp/unread-git.pub" && chmod 000 "$tmp/unread-git.pub"
if ! cat "$tmp/unread-git.pub" >/dev/null 2>&1; then
    with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com \
        "WS_GIT_SIGNING_KEY=$tmp/unread-git.pub"
    stops "setup.sh: cannot read $tmp/unread-git.pub"
fi

# Logged out, gh must draw a warning at each run, and nothing else where the files are written.
# With gh's login, a 404, the answer to a login without the scope of signing keys, must draw a
# warning that names the scope. Another error must draw its last line, its final period dropped, as
# must an add that leaves the account without the key. Then, where the account lacks the key,
# --check must show the add, a run must add it, and a second run and --check print nothing.
with 'WS_GIT_NAME="ws stand-in #1"' WS_GIT_EMAIL=ws@example.com
rm "$tmp/login"
warning=$(warned "$pub")
prints "setup.sh: wrote $HOME/.gitconfig$nl$warning"
prints "$warning"
prints "$warning" --check
touch "$tmp/login"
until="Until a later run adds $pub, GitHub shows the commits it signs as unverified"
printf '%s\n' 'Post "https://api.github.com/user/ssh_signing_keys": EOF' 'gh: offline.' \
    >"$tmp/error"
prints "setup.sh: warning: gh failed: gh: offline. $until"
echo 'gh: Not Found (HTTP 404)' >"$tmp/error"
prints "setup.sh: warning: gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h\
 github.com -s write:ssh_signing_key adds it. $until"
rm "$tmp/error"
: >"$tmp/keys"
touch "$tmp/lost"
prints "setup.sh: warning: gh failed: gh api user/ssh_signing_keys lacks the key after $add. $until"
rm "$tmp/lost"
prints "setup.sh: would run $add" --check
prints "setup.sh: ran $add"
prints ''
prints '' --check
[ "$(grep '^ssh-key add ' "$tmp/gh.log" | tail -n 1)" = "${add#gh }" ] &&
    [ "$(cat "$tmp/keys")" = "$(awk '{ print $1, $2 }' "$pub")" ] ||
    error "the stand-in logged another add, or holds other keys, than $pub's"

# Where PATH has no gh, setup.sh must leave it so without git's keys. With them, --check must say
# that Homebrew would install it, name the gh at Homebrew's prefix as git's credential command, and
# name the add. A run must install it, once. The stand-in brew's prefix is $tmp/brew, where its
# install puts the stand-in gh, and it passes other commands to the runner's brew.
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
with
prints ''
with 'WS_GIT_NAME="ws stand-in #1"' WS_GIT_EMAIL=ws@example.com
setup --check || error "setup.sh --check failed: $out"
case $out in
"setup.sh: would run brew install gh$nl"*"$nl+    helper = !$tmp/brew/bin/gh auth\
 git-credential$nl"*"${nl}setup.sh: would run $add once gh is installed, where it has a login\
 that lacks the key") ;;
*) error "setup.sh --check showed no install of gh, another credential command, or no add: $out" ;;
esac
[ ! -f "$tmp/brew.log" ] || error "setup.sh installed gh without git's keys, or under --check"
prints "setup.sh: installing gh${nl}setup.sh: wrote $files/gitconfig$nl$warning"
prints "$warning"
[ "$(cat "$tmp/brew.log")" = 'install gh' ] || error "setup.sh ran brew install gh other than once"
PATH=$path
# shellcheck source=.github/mac-git-yubikey-checks.sh
. .github/mac-git-yubikey-checks.sh
