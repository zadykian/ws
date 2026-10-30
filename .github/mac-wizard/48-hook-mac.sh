#!/bin/sh
# The stage Hook on the Mac, on the Mac that 47-hook.sh leaves. WS_HOOK names a repository that the
# stand-in gh on the Mac clones, with .githooks and an empty Brewfile, as the stand-in ws's gh
# clones it there. --check must wait for Hook on ws, and clone nothing. A run must clone it once
# into ~/repository/REPO, and point its core.hooksPath at its .githooks, by an absolute path. brew
# bundle check must find the Brewfile met. As git's global config names no credential command of
# gh's, the closing list must name gh auth setup-git, and the next run print nothing. With
# core.hooksPath gone, and gh's credential command in git's global config, a run must set it again,
# and name no gh auth setup-git. The stand-in ws holds a word list of stand-in words, which the
# first run must copy to the Mac's ~/.config/ws/forbidden-words, mode 600, once, printing none of
# it. The next run must fetch it no more. Once the Mac's list differs, --check must say it would
# copy it, with no diff, and a run copy it again. Where the Mac's list is a link to a file, --check
# and a run must stop, naming that file, and leave it. Without the Mac's clone, --check must say it
# would clone it. mac.conf, the clones, the lists and ws's stamp go back, and a run must print
# nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
clone=$HOME/repository/hook
before=$RUNNER_TEMP/hook-mac.conf
list=$ws/root/root/.config/ws/forbidden-words
mine=$HOME/.config/ws/forbidden-words
setup="- Pushes from the Mac's clones to GitHub: gh auth setup-git, which has git take gh's login"
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard [ARG]: a run of the wizard with no terminal, git's global config the file GIT_CONFIG_GLOBAL
# names. It prints the output, and sets out and status.
wizard() {
    out=$(GIT_CONFIG_GLOBAL=$RUNNER_TEMP/gitconfig sh "$wizard" "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# clones prints how many times the stand-in gh on the Mac cloned the repository, 47-hook.sh's runs
# among them.
clones() {
    grep -c "^repo clone example/hook $clone -- -q\$" "$RUNNER_TEMP/gh/log"
}

# copies prints how many times the files script printed ws's word list, for its hash.
copies() {
    grep -c '^root: sh -s -- words [0-9a-f]' "$ws/log"
}

# leaked succeeds where the run's output holds a word of the list, or a line of a diff.
leaked() {
    printf '%s\n' "$out" | grep -q -e example-word -e 'stand-in phrase' -e '^+'
}

# linked [ARG]: the run, with the Mac's list a link to a file, must stop, naming that file. It must
# fetch, print and change none of the list, and leave the link.
linked() {
    wizard "$@"
    [ "$status" = 1 ] && ! leaked && printf '%s\n' "$out" | grep -qxF "wizard.sh: Hook on the \
Mac: $mine is a link to $RUNNER_TEMP/words, which the wizard leaves, and ws's word list differs: \
remove the link, or copy root's ~/.config/ws/forbidden-words on ws to that file, then run the \
wizard again" &&
        [ -L "$mine" ] && [ "$(tail -n 1 "$mine")" = other-word ] &&
        [ "$(copies)" = $((copied + 1)) ] ||
        error "wizard.sh $* with the Mac's list a link exited $status, or replaced it"
}

cp "$conf" "$before"
{ sed 's|^WS_HOOK=$|WS_HOOK=example/hook|' "$before" && echo WS_HOOK_SECRET=; } >"$conf"
: >"$RUNNER_TEMP/gitconfig"
cloned=$(clones)
copied=$(copies)
sudo mkdir -p "${list%/*}"
printf '%s\n' '# stand-in words' example-word 'stand-in phrase' | sudo tee "$list" >/dev/null
wizard --check
[ "$status" = 0 ] && printf '%s\n' "$out" |
    grep -qx 'wizard\.sh: Hook on the Mac: after Hook on ws' && [ ! -e "$clone" ] ||
    error "wizard.sh --check exited $status, or cloned"
wizard
[ "$status" = 0 ] && ! printf '%s\n' "$out" | grep -q 'running brew bundle' &&
    [ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] &&
    [ "$(clones)" = $((cloned + 1)) ] ||
    error "wizard.sh exited $status, or cloned other than once, or set no core.hooksPath"
printf '%s\n' "$out" | grep -qxF -e "$setup (docs/mac-git.md)." ||
    error 'wizard.sh named no gh auth setup-git by hand'
cmp -s "$list" "$mine" && [ "$(stat -f %Lp "$mine")" = 600 ] && ! leaked &&
    [ "$(copies)" = $((copied + 1)) ] ||
    error "wizard.sh copied ws's word list other than once, mode 600, or printed it"
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(clones)" = $((cloned + 1)) ] &&
    [ "$(copies)" = $((copied + 1)) ] || error "the next run exited $status, cloned, or copied"
echo other-word >>"$mine"
wizard --check
[ "$status" = 0 ] && ! leaked && printf '%s\n' "$out" | grep -qxF "wizard.sh: Hook on the Mac: \
would copy ws's word list to $mine, mode 600" && [ "$(tail -n 1 "$mine")" = other-word ] ||
    error "wizard.sh --check with the Mac's list changed exited $status, or copied it"
mv "$mine" "$RUNNER_TEMP/words"
ln -s "$RUNNER_TEMP/words" "$mine"
linked --check
linked
rm "$mine"
mv "$RUNNER_TEMP/words" "$mine"
git -C "$clone" config --unset core.hooksPath
git config -f "$RUNNER_TEMP/gitconfig" credential.https://github.com.helper \
    '!/opt/homebrew/bin/gh auth git-credential'
wizard
[ "$status" = 0 ] && ! printf '%s\n' "$out" | grep -qF -e "$setup" &&
    [ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] ||
    error "wizard.sh with gh's credential command exited $status, set no core.hooksPath, or" \
        "named gh auth setup-git"
cmp -s "$list" "$mine" && [ "$(copies)" = $((copied + 2)) ] ||
    error "wizard.sh copied no word list again"
rm -rf "$clone"
wizard --check
[ "$status" = 0 ] && printf '%s\n' "$out" |
    grep -qF "wizard.sh: Hook on the Mac: would run gh repo clone example/hook $clone, then " &&
    [ ! -e "$clone" ] || error "wizard.sh --check without the Mac's clone exited $status, or cloned"

cp "$before" "$conf"
rm -f "$mine"
sudo rm -rf "$ws/root/root/repository/hook" "$ws/root/var/lib/ws-wizard/hook-hook" \
    "$ws/root/root/.config"
rm -f "$ws/clone.log" "$ws/apply.log" "$before" "$RUNNER_TEMP/gitconfig"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
