# shellcheck shell=sh
# git_gh, which wizard-gh.sh and wizard-hook.sh source. The git step of mac/setup.sh is ws's
# `ws setup git`. It calls the functions of lib.sh.

# git_gh [STAGE] sets gh to the path of GitHub's CLI, which Homebrew installs where PATH has none.
# Under --check, it is the path Homebrew would give it. git runs gh in whatever directory it
# works in, so the path must be absolute, not one that a relative directory of PATH gives. Its
# messages name STAGE, where the wizard's stage gives one.
# shellcheck disable=SC2120 # wizard-gh.sh's stage passes its name
git_gh() {
    if ! gh=$(command -v gh); then
        brew=$(command -v brew) || ws_die "${1:+$1: }install Homebrew first: https://brew.sh"
        if ws_checking; then
            ws_say "${1:+$1: }would run brew install gh"
            gh=$("$brew" --prefix)/bin/gh
            return 0
        fi
        ws_say "${1:+$1: }installing gh"
        "$brew" install gh >&2 || ws_die "${1:+$1: }brew install gh failed"
        gh=$(command -v gh) || ws_die "${1:+$1: }brew install gh left no gh on PATH"
    fi
    case $gh in
    /*) ;;
    *) ws_die "${1:+$1: }PATH gives gh as $gh, which git would look for in each repository:" \
        "drop PATH's relative directories" ;;
    esac
}
