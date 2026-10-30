#!/bin/sh
# Sets up a new Mac to work on ws, the server this repository sets up. Run it as yourself, there:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/mac/bootstrap.sh | sh
#
# It installs Apple's Command Line Tools through their dialog, and Homebrew from the Homebrew.pkg
# pinned below. It clones the repository into ~/repository/ws, or leaves a clone there as it is,
# which --update pulls. Then it runs the clone's mac/wizard.sh (docs/mac-wizard.md) with its own
# arguments, given as `sh -s -- --check`, under which neither changes anything. A second run prints
# nothing. caffeinate keeps the Mac from idle sleep while it runs.
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

repository=https://github.com/zadykian/ws
dir=$HOME/repository/ws
# Homebrew's release: its Homebrew.pkg, that file's SHA-256, and its signer, as pkgutil names it.
brew_version=7.0.10
brew_url=https://github.com/Homebrew/brew/releases/download/$brew_version/Homebrew.pkg
brew_sha256=ae13b8b0ac910730d3e982a9e7518288d17fd0ad272e42f412027e8ab4555575
brew_signer='Developer ID Installer: Patrick Linnane (927JGANW46)'
# The wizard's stages, split by |, as mac/wizard.sh's main runs them, for --check without a clone.
stages="Answers|Mac key|gh on the Mac|setup.sh|Host key|Root's key|ws's run|\
claude on ws|Tunnel|Hook on the Mac|Hook on ws|Word list|Cable"

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

# clt installs the Command Line Tools through the dialog that xcode-select --install opens, where
# none is open yet, and waits up to 60 minutes for them. A dialog cancelled opens at the next run.
clt() {
    clt_done || would Tools 'run xcode-select --install, for the Command Line Tools' && return 0
    say "Tools: click Install in the Command Line Tools' dialog; waiting up to 60 minutes"
    xcode-select --install 2>/dev/null || true
    waited=0
    until clt_done; do
        [ "$waited" -lt 3600 ] || die "Tools: no Command Line Tools after 60 minutes"
        sleep 10
        waited=$((waited + 10))
    done
}

# brew_check PKG stops unless PKG has the pinned SHA-256 and signer. The signer is the first of
# the chain that pkgutil prints, after a status of Apple's Developer ID.
brew_check() {
    sum=$(shasum -a 256 "$1") || die "Tools: cannot read $1"
    [ "${sum%% *}" = "$brew_sha256" ] || die "Tools: $1's SHA-256 is ${sum%% *}, not the pinned one"
    signer=$(pkgutil --check-signature "$1" |
        sed -n '/^ *Status: .* issued by Apple for distribution$/,$s/^ *1\. //p')
    [ "$signer" = "$brew_signer" ] || die "Tools: $1 is signed by '$signer', not '$brew_signer'"
}

# homebrew installs Homebrew from the pinned Homebrew.pkg where /opt/homebrew/bin/brew doesn't run.
homebrew() {
    /opt/homebrew/bin/brew --version >/dev/null 2>&1 ||
        would Tools "install Homebrew $brew_version from its Homebrew.pkg, through sudo" && return 0
    say "Tools: installing Homebrew $brew_version"
    pkg=$(mktemp -d "${TMPDIR:-/tmp}/ws-bootstrap.XXXXXXXX")/Homebrew.pkg ||
        die "Tools: cannot make a directory in ${TMPDIR:-/tmp}"
    trap 'rm -rf "${pkg%/*}"' EXIT
    trap 'exit 1' HUP INT TERM
    curl -fsSL -o "$pkg" "$brew_url" || die "Tools: cannot download $brew_url"
    brew_check "$pkg"
    sudo installer -pkg "$pkg" -target / >&2 || die "Tools: sudo installer failed on $pkg"
    /opt/homebrew/bin/brew --version >/dev/null 2>&1 || die "Tools: no brew runs after the install"
    rm -rf "${pkg%/*}"
}

# clone clones the repository into dir, or leaves a clone there as it is, with core.hooksPath at its
# .githooks by an absolute path (docs/checks.md, The hooks). Under --check, git may wait for Tools.
clone() {
    if [ ! -e "$dir/.git" ]; then
        [ ! -e "$dir" ] || die "Clone: $dir is there, but is not a clone: move it away"
        would Clone "clone $repository into $dir" && return 0
        say "Clone: cloning $repository into $dir"
        git clone -q "$repository" "$dir" || die "Clone: cannot clone $repository into $dir"
    elif ! clt_done; then
        say "Clone: after Tools"
        return 0
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
    export PATH="/opt/homebrew/bin:$PATH"
    args "$@"
    [ "$(uname -s)" = Darwin ] || die "it sets up a Mac, not $(uname -s)"
    [ "$(uname -m)" = arm64 ] || die "it sets up a Mac with Apple silicon, not $(uname -m)"
    [ "$(id -u)" != 0 ] || die "run it as yourself, not root: it asks sudo where it needs root"
    caffeinate -i -w $$ </dev/null >/dev/null 2>&1 &
    clt
    homebrew
    clone
    if [ ! -e "$dir/.git" ] || ! clt_done; then
        # Under --check, the clone or its git is missing, so the wizard's stages wait for them.
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
