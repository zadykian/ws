#!/bin/sh
# The stand-ins of the wizard's job, made once before checks.sh runs the cases: what a new Mac and
# its user give the wizard, on GitHub's macOS runner.

set -eu

sw_vers
# setup.sh stops where a directory above root's copies isn't root's alone, as on a Mac whose
# /usr/local an old Homebrew took. Where the runner's image gave /usr/local or its libexec to its
# user, they become root's, as in mac.yml: their contents stay as they are.
for dir in /usr/local /usr/local/libexec; do
    [ -d "$dir" ] || continue
    if [ "$(stat -f '%u %Lp' "$dir")" != "0 755" ]; then
        echo "::warning::$dir was $(stat -f '%Su:%Sg %Lp' "$dir"); made root's, 755"
        sudo chown root:wheel "$dir"
        sudo chmod 755 "$dir"
    fi
done
# mac.conf's answers: ws by the router's name and port alone, as a Mac away from home has it.
mkdir -p ~/.config/ws
printf '%s\n' WS_REMOTE=localhost WS_REMOTE_PORT=22022 >~/.config/ws/mac.conf
# dash, which macOS has as /bin/dash, from Homebrew where the runner lacks it.
command -v dash >/dev/null || brew install dash
# PATH as a shell that predates Homebrew has it, without the directories that hold brew, for the
# cases that give it to the wizard.
path=
IFS=:
for each in $PATH; do
    [ -x "$each/brew" ] || path=$path${path:+:}$each
done
unset IFS
if (PATH=$path && command -v brew); then
    echo "::error::brew is still on PATH, $path"
    exit 1
fi
echo "$path" >"$RUNNER_TEMP/path"
