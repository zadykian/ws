#!/bin/bash
# Make /usr/local root's, as on a Mac.
# setup.sh stops where a directory above root's copies isn't root's alone, as on a Mac whose
# /usr/local an old Homebrew took. Where the runner's image gave /usr/local or its libexec to
# its user, they become root's, as on a Mac with Homebrew in /opt/homebrew: their contents
# stay as they are.

set -eo pipefail

for dir in /usr/local /usr/local/libexec; do
    [ -d "$dir" ] || continue
    if [ "$(stat -f '%u %Lp' "$dir")" != "0 755" ]; then
        echo "::warning::$dir was $(stat -f '%Su:%Sg %Lp' "$dir"); made root's, 755"
        sudo chown root:wheel "$dir"
        sudo chmod 755 "$dir"
    fi
done
