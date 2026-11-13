#!/bin/sh
# Homebrew's pinned .pkg, which the Homebrew stage leaves alone on the runner, whose brew runs. The
# pins come from homebrew.go by sed. The real .pkg must pass the SHA-256 check and give the pinned
# signer through pkgutil, whose output pkgSigner reads as the sed below does, which the log shows.
# The .pkg cut short must fail the sum, and another signer than pkgutil's must differ from the pin.

set -u

source=$HOME/repository/ws/roles/client/macos/stages/tools/homebrew.go
tmp=$(mktemp -d "$RUNNER_TEMP/homebrew.XXXXXX")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# pin NAME prints the value of the constant NAME in homebrew.go.
pin() {
    sed -n "s/^[[:space:]]*$1 *= *\"\(.*\)\"\$/\1/p" "$source"
}

# signer prints the first of the chain that pkgutil prints for the .pkg in $1, after a status of
# Apple's Developer ID: pkgSigner's rule.
signer() {
    pkgutil --check-signature "$1" |
        sed -n '/^ *Status: .* issued by Apple for distribution$/,$s/^ *1\. //p'
}

version=$(pin brewVersion)
sum=$(pin brewSHA256)
pinned=$(pin brewSigner)
# brewURL is a literal, brewVersion and a literal.
url=$(sed -n "s|^[[:space:]]*brewURL *= *\"\([^\"]*\)\" + brewVersion + \"\([^\"]*\)\"\$|\1$version\2|p" "$source")
[ -n "$version" ] && [ -n "$sum" ] && [ -n "$pinned" ] && [ -n "$url" ] || error "homebrew.go lacks a pin"
curl -fsSL -o "$tmp/Homebrew.pkg" "$url" || error "cannot download $url"
pkgutil --check-signature "$tmp/Homebrew.pkg"
got=$(shasum -a 256 "$tmp/Homebrew.pkg")
[ "${got%% *}" = "$sum" ] || error "the pinned Homebrew.pkg has the SHA-256 ${got%% *}, not $sum"
[ "$(signer "$tmp/Homebrew.pkg")" = "$pinned" ] ||
    error "the pinned Homebrew.pkg is signed by '$(signer "$tmp/Homebrew.pkg")', not '$pinned'"
head -c 1000000 "$tmp/Homebrew.pkg" >"$tmp/cut.pkg"
got=$(shasum -a 256 "$tmp/cut.pkg")
[ "${got%% *}" != "$sum" ] || error "the cut .pkg has the pinned SHA-256"
[ "$pinned" != 'Developer ID Installer: Jane Doe (ABCDE12345)' ] || error "the pinned signer is the test's"
rm -rf "$tmp"
exit "$fail"
