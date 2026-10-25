#!/bin/sh
# The stage Hook on the Mac, on the Mac that 47-hook.sh leaves. WS_HOOK names a repository that the
# stand-in gh on the Mac clones, with .githooks and an empty Brewfile, as the stand-in ws's gh
# clones it there. --check must wait for Hook on ws, and clone nothing. A run must clone it once
# into ~/repository/REPO, and point its core.hooksPath at its .githooks, by an absolute path. brew
# bundle check must find the Brewfile met. As git's global config names no credential command of
# gh's, the closing list must name gh auth setup-git, and the next run print nothing. With
# core.hooksPath gone, and gh's credential command in git's global config, a run must set it again,
# and name no gh auth setup-git. mac.conf, the clones and ws's stamp go back, and a run must print
# nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
clone=$HOME/repository/hook
before=$RUNNER_TEMP/hook-mac.conf
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

cp "$conf" "$before"
{ sed 's|^WS_HOOK=$|WS_HOOK=example/hook|' "$before" && echo WS_HOOK_SECRET=; } >"$conf"
: >"$RUNNER_TEMP/gitconfig"
cloned=$(clones)
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
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(clones)" = $((cloned + 1)) ] ||
    error "the next run exited $status, or cloned"
git -C "$clone" config --unset core.hooksPath
git config -f "$RUNNER_TEMP/gitconfig" credential.https://github.com.helper \
    '!/opt/homebrew/bin/gh auth git-credential'
wizard
[ "$status" = 0 ] && ! printf '%s\n' "$out" | grep -qF -e "$setup" &&
    [ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] ||
    error "wizard.sh with gh's credential command exited $status, set no core.hooksPath, or" \
        "named gh auth setup-git"

cp "$before" "$conf"
rm -rf "$clone"
sudo rm -rf "$ws/root/root/repository/hook" "$ws/root/var/lib/ws-wizard/hook-hook"
rm -f "$ws/clone.log" "$ws/apply.log" "$before" "$RUNNER_TEMP/gitconfig"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
