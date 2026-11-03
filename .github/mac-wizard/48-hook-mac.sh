#!/bin/sh
# The stage Hook on the Mac, on the Mac that 47-hook.sh leaves. WS_HOOK names a
# repository that the stand-in gh on the Mac clones, with .githooks and an empty Brewfile. --check
# must say it would clone it, and clone nothing, and Hook on ws wait for it. A run must clone it once
# into ~/repository/REPO, and point its core.hooksPath at its .githooks, by an absolute path. brew
# bundle check must find the Brewfile met. As git's global config names no credential command of
# gh's, the closing list must name gh auth setup-git, and the next run print nothing. With
# core.hooksPath gone, and gh's credential command in git's global config, a run must set it again,
# and name no gh auth setup-git. Without the Mac's clone, --check must say it
# would clone it. mac.conf, the clone and ws's stamp go back, and a run must print
# nothing.

set -u

wsbin=$HOME/.local/bin/ws
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
    out=$(GIT_CONFIG_GLOBAL=$RUNNER_TEMP/gitconfig "$wsbin" mac wizard "$@" </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# clones prints how many times the stand-in gh on the Mac cloned the repository, 47-hook.sh's runs
# among them.
clones() {
    grep -c "^repo clone example/hook $clone -- -q\$" "$RUNNER_TEMP/gh/log"
}

cp "$conf" "$before"
sed 's|^WS_HOOK=$|WS_HOOK=example/hook@v1.0.0|' "$before" >"$conf"
: >"$RUNNER_TEMP/gitconfig"
cloned=$(clones)
wizard --check
[ "$status" = 0 ] &&
    printf '%s\n' "$out" | grep -qF "ws mac wizard: Hook on the Mac: would run gh repo clone \
example/hook $clone, then " && [ ! -e "$clone" ] ||
    error "ws mac wizard --check exited $status, or cloned"
wizard
[ "$status" = 0 ] && ! printf '%s\n' "$out" | grep -q 'running brew bundle' &&
    [ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] &&
    [ "$(clones)" = $((cloned + 1)) ] ||
    error "ws mac wizard exited $status, or cloned other than once, or set no core.hooksPath"
printf '%s\n' "$out" | grep -qxF -e "$setup (docs/mac-git.md)." ||
    error 'ws mac wizard named no gh auth setup-git by hand'
wizard
[ "$status" = 0 ] && [ -z "$out" ] && [ "$(clones)" = $((cloned + 1)) ] ||
    error "the next run exited $status, or cloned"
git -C "$clone" config --unset core.hooksPath
git config -f "$RUNNER_TEMP/gitconfig" credential.https://github.com.helper \
    '!/opt/homebrew/bin/gh auth git-credential'
wizard
[ "$status" = 0 ] && ! printf '%s\n' "$out" | grep -qF -e "$setup" &&
    [ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] ||
    error "ws mac wizard with gh's credential command exited $status, set no core.hooksPath, or" \
        "named gh auth setup-git"
rm -rf "$clone"
wizard --check
[ "$status" = 0 ] && printf '%s\n' "$out" |
    grep -qF "ws mac wizard: Hook on the Mac: would run gh repo clone example/hook $clone, then " &&
    [ ! -e "$clone" ] || error "ws mac wizard --check without the Mac's clone exited $status, or cloned"

cp "$before" "$conf"
sudo rm -rf "$ws/root/root/repository/hook" "$ws/root/var/lib/ws-wizard/hook-hook" "$ws/root/var/lib/ws-wizard/bin"
rm -f "$ws/apply.log" "$ws/check.log" "$ws/hook.args" "$before" "$RUNNER_TEMP/gitconfig"
wizard
[ "$status" = 0 ] && [ -z "$out" ] || error "ws mac wizard after it exited $status"
exit "$fail"
