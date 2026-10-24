#!/bin/sh
# Sets up a Mac to work on ws, the server this repository sets up. Run it as yourself, on a Mac with
# Homebrew:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/mac/bootstrap.sh | sh
#
# It clones the repository into ~/repository/ws, or leaves a clone there as it is, which --update
# pulls. Then it runs the clone's mac/wizard.sh (docs/mac-wizard.md) with its own arguments, given
# as `sh -s -- --check`, under which neither changes anything. A second run prints nothing.
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

repository=https://github.com/zadykian/ws
dir=$HOME/repository/ws
# The wizard's stages, split by |, as mac/wizard.sh's main runs them, for --check without a clone.
stages='Answers|setup.sh'

# say prints a message of bootstrap.sh's.
say() {
    printf 'bootstrap.sh: %s\n' "$*" >&2
}

# die prints a message of bootstrap.sh's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# would STAGE WHAT says, under --check, what STAGE would do, and fails otherwise, where it acts.
would() {
    [ "$check" = 1 ] && say "$1: would $2"
}

# args takes the arguments, which go to the wizard too: --check sets check, and --update update.
args() {
    check=0
    update=0
    for arg; do
        case $arg in
        --check) check=1 ;;
        --update) update=1 ;;
        *) die "unknown argument '$arg'; usage: bootstrap.sh [--check] [--update]" ;;
        esac
    done
}

# clt_done succeeds where the Command Line Tools, or Xcode, are there: xcode-select names their
# directory, which holds git. Never git itself, whose shim in /usr/bin would open their install.
clt_done() {
    clt=$(xcode-select -p 2>/dev/null) && [ -x "$clt/usr/bin/git" ]
}

# tools stops where the clone and the wizard would lack the Command Line Tools or Homebrew.
tools() {
    if ! clt_done || ! /opt/homebrew/bin/brew --version >/dev/null 2>&1; then
        die "Tools: install Homebrew first, with the Command Line Tools it brings: https://brew.sh"
    fi
}

# clone clones the repository into dir, or leaves a clone there as it is, with core.hooksPath at its
# .githooks by an absolute path (docs/checks.md, The hooks).
clone() {
    if [ ! -e "$dir/.git" ]; then
        [ ! -e "$dir" ] || die "Clone: $dir is there, but is not a clone: move it away"
        would Clone "clone $repository into $dir" && return 0
        say "Clone: cloning $repository into $dir"
        git clone -q "$repository" "$dir" || die "Clone: cannot clone $repository into $dir"
    fi
    if [ "$(git -C "$dir" config core.hooksPath)" != "$dir/.githooks" ] &&
        ! would Clone "set core.hooksPath to $dir/.githooks"; then
        git -C "$dir" config core.hooksPath "$dir/.githooks" || die "Clone: git config failed"
        say "Clone: set core.hooksPath to $dir/.githooks"
    fi
    update
}

# update pulls a clean clone on main by fast-forward, under --update, where ls-remote, which writes
# nothing, finds origin's main elsewhere. On another branch, or not clean, it warns and leaves it.
update() {
    [ "$update" = 1 ] || return 0
    if [ "$(git -C "$dir" symbolic-ref -q --short HEAD)" != main ] ||
        [ -n "$(git -C "$dir" --no-optional-locks status --porcelain --untracked-files=no)" ]; then
        say "warning: Clone: leaves $dir as it is, not on main or not clean: pull it by hand"
    elif head=$(git -C "$dir" rev-parse HEAD) && ! git -C "$dir" ls-remote origin refs/heads/main |
        grep -q "^$head" && ! would Clone "pull $dir by fast-forward"; then
        git -C "$dir" pull -q --ff-only || die "Clone: cannot pull $dir by fast-forward"
        [ "$(git -C "$dir" rev-parse HEAD)" = "$head" ] ||
            say "Clone: pulled $dir to $(git -C "$dir" rev-parse --short HEAD)"
    fi
}

main() {
    # A shell started before Homebrew's install lacks /opt/homebrew/bin, so it goes first here.
    PATH=/opt/homebrew/bin:$PATH
    export PATH
    args "$@"
    [ "$(uname -s)" = Darwin ] || die "it sets up a Mac, not $(uname -s)"
    [ "$(uname -m)" = arm64 ] || die "it sets up a Mac with Apple silicon, not $(uname -m)"
    [ "$(id -u)" != 0 ] || die "run it as yourself, not root: it asks sudo where it needs root"
    tools
    clone
    if [ ! -e "$dir/.git" ]; then
        # Under --check, the clone is missing, so the wizard's stages wait for it.
        echo "$stages" | tr '|' '\n' | sed 's/.*/wizard.sh: &: after Clone/' >&2
        return 0
    fi
    [ -f "$dir/mac/wizard.sh" ] || die "Clone: $dir has no mac/wizard.sh: pull it first"
    sh "$dir/mac/wizard.sh" "$@"
}

# Under curl | sh, stdin is the rest of this script, so main reads the terminal, or nothing where
# there is none. The braces make a line cut short a syntax error, rather than main run without it.
tty=/dev/null
! (exec </dev/tty) 2>/dev/null || tty=/dev/tty
{ main "$@" <"$tty"; }
