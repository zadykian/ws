#!/bin/bash
# Prints the autoinstall file of a release: ubuntu/autoinstall.yaml with its blanks VERSION
# and SHA256 filled, and `#cloud-config` as its first line. The installer's NoCloud source needs
# that line, and yamllint refuses it in the tree. VERSION is the release's tag, and SHA256 the
# SHA-256 of the release's install.sh. The maintainer's blanks stay. Usage: autoinstall.sh TAG SUM
set -euo pipefail

usage() {
    echo "usage: autoinstall.sh TAG SUM, as v0.13.0 and the SHA-256 of that release's install.sh" >&2
    exit 2
}

[ $# -eq 2 ] || usage
tag=$1
sum=$2
# The tag goes into a download's address and a command line, so only a release's form passes.
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]] || usage
[[ $sum =~ ^[0-9a-f]{64}$ ]] || usage

file=$(dirname "$0")/../ubuntu/autoinstall.yaml
echo '#cloud-config'
sed -e "s/\bVERSION\b/$tag/g" -e "s/\bSHA256\b/$sum/g" "$file"
