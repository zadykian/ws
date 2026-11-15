# shellcheck shell=sh
# The cases that want a silent run of ws client macos apply source this file. With git on, a run
# that changes nothing still says that GitHub lists git's key already. Sourced, never run.

# unlisted TEXT prints TEXT less each line that ends in what ws client macos apply says of a key that
# GitHub lists already. It matches the end of a line, so that a prefix, as the wizard's, passes.
unlisted() {
    printf '%s\n' "$1" | grep -v ": git's key is among GitHub's signing keys already\$" || true
}
