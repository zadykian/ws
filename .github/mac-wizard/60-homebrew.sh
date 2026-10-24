#!/bin/sh
# Homebrew's pinned .pkg, which bootstrap.sh's runs leave alone on the runner, whose brew runs.
# bootstrap.sh's functions, all but its last line, which runs main, must download it and pass its
# brew_check. That is the SHA-256 and signer that a run checks before sudo installer, which this
# never runs. brew_check must refuse the .pkg cut short, and another signer than pkgutil's, which
# the log shows.

set -u

bootstrap=$HOME/repository/ws/mac/bootstrap.sh
tmp=$(mktemp -d "$RUNNER_TEMP/homebrew.XXXXXX")
pinned=$(sed -n "s/^brew_signer='\(.*\)'\$/\1/p" "$bootstrap")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# refused WHY PKG [SIGNER]: brew_check must exit 1, with WHY in what it prints.
refused() {
    why=$1
    shift
    out=$(SIGNER=${2-} sh "$tmp/check.sh" "$1" 2>&1)
    status=$?
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF "$why"; then
        error "brew_check $* exited $status, printing '$out', where 1 and '$why' were due"
    fi
}

# check.sh PKG: bootstrap.sh's functions and values, then brew_check PKG, with the pinned signer or
# SIGNER's. Its functions and values, piped to sh, print the .pkg's URL too.
# shellcheck disable=SC2016 # the scripts expand them
{
    sed '$d' "$bootstrap"
    echo 'brew_signer=${SIGNER:-$brew_signer}'
    echo 'brew_check "$1"'
} >"$tmp/check.sh"
# shellcheck disable=SC2016 # sh expands it
url=$({
    sed '$d' "$bootstrap"
    echo 'echo "$brew_url"'
} | sh)
curl -fsSL -o "$tmp/Homebrew.pkg" "$url" || error "cannot download $url"
pkgutil --check-signature "$tmp/Homebrew.pkg"
sh "$tmp/check.sh" "$tmp/Homebrew.pkg" ||
    error "the pinned Homebrew.pkg failed brew_check, with status $?"
head -c 1000000 "$tmp/Homebrew.pkg" >"$tmp/cut.pkg"
refused "$tmp/cut.pkg's SHA-256 is " "$tmp/cut.pkg"
refused "is signed by '$pinned', not 'Developer ID Installer: Jane Doe (ABCDE12345)'" \
    "$tmp/Homebrew.pkg" 'Developer ID Installer: Jane Doe (ABCDE12345)'
rm -rf "$tmp"
exit "$fail"
