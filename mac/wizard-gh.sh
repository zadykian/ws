# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's stage gh on the Mac: GitHub's CLI, from Homebrew where missing, logged in to
# github.com with the scopes that add the Mac's keys. setup.sh, which runs after, adds git's signing
# key with that login. wizard.sh sources it after wizard-keys.sh, and runs the stage after the Mac
# key. It calls the functions of lib.sh, setup-lib.sh and setup-git.sh.

# gh's install is setup-git.sh's git_gh, sourced from dir, mac/'s path, as wizard.sh sets it.
# shellcheck source=setup-git.sh disable=SC2154 # dir is wizard.sh's
. "$dir/setup-git.sh"

# The scopes of gh's login that gh ssh-key add needs, for a key that logs in and for one that signs.
gh_scopes=write:public_key,write:ssh_signing_key

# gh_after prints Tools, bootstrap.sh's stage, where PATH has neither gh nor brew to install it.
gh_after() {
    command -v gh >/dev/null || command -v brew >/dev/null || echo Tools
}

# gh_done succeeds where gh is logged in to github.com.
gh_done() {
    command -v gh >/dev/null && gh auth status -h github.com >/dev/null 2>&1
}

# gh_do installs gh where missing, and logs it in where it has no login. --check says what it would
# run.
gh_do() {
    git_gh "gh on the Mac"
    if [ ! -x "$gh" ] || ! "$gh" auth status -h github.com >/dev/null 2>&1; then
        gh_web login -p https -w
    fi
}

# gh_web ACTION ARG... runs gh auth ACTION for github.com with ARGs and the scopes. Its device flow
# shows a code to enter in the browser, at github.com/login/device. --check says so. With no
# terminal, it stops, naming the command.
gh_web() {
    gh_run="gh auth $* -h github.com -s $gh_scopes"
    if ws_checking; then
        ws_say "gh on the Mac: would run $gh_run"
        return 0
    fi
    [ -t 0 ] || ws_die "gh on the Mac: gh lacks a login to github.com with the scopes of the" \
        "Mac's keys, and no terminal is there for $gh_run: run it, then the wizard again"
    "$gh" auth "$@" -h github.com -s "$gh_scopes" >&2 || ws_die "gh on the Mac: $gh_run failed"
}

# gh_fail says what to do where gh still has no login to github.com.
gh_fail() {
    echo "gh still has no login to github.com, as above; run the wizard again"
}
