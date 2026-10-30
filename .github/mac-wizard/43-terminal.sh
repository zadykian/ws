#!/bin/sh
# claude's login on the stand-in ws at a terminal, on the Mac that 42-login.sh leaves. With it
# logged out, a run of the wizard at a terminal of script's must run claude auth login through
# bash -lc in a session of ssh -t, and list what stays by hand. The stand-in claude takes a
# terminal. The next run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# macOS puts /usr/local/bin first on a login shell's PATH, so the stand-in claude goes there for
# claude's session, as root's ~/.local/bin would on ws.
if [ -e /usr/local/bin/claude ]; then
    echo '::error::the runner has a claude in /usr/local/bin already'
    exit 1
fi
sudo ln -s "$(cd "$(dirname "$0")" && pwd)/ws-standin.sh" /usr/local/bin/claude
rm "$ws/claude"
sh "$wizard" </dev/null >/dev/null 2>&1
lines=$(wc -l <"$ws/log")
# script gives the wizard a terminal of its own, which the session of ssh -t takes.
{ sleep 5; } | script -q /dev/null sh "$wizard" >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
tail -n "+$((lines + 1))" "$ws/log" | tee "$RUNNER_TEMP/sessions"
[ "$status" = 0 ] || error "wizard.sh at a terminal exited $status"
grep -qxF 'root: exec bash -lc "claude auth login"' "$RUNNER_TEMP/sessions" ||
    error "no session on ws ran claude auth login"
grep -qF 'What stays by hand:' "$RUNNER_TEMP/tty" && [ -f "$ws/claude" ] ||
    error "wizard.sh left claude logged out on ws, or listed nothing by hand"
rm -f "$RUNNER_TEMP/sessions"
sudo rm /usr/local/bin/claude
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status: '$out'"
exit "$fail"
