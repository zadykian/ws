#!/bin/sh
# gh's and claude's logins on the stand-in ws at a terminal, on the Mac that 42-login.sh leaves.
# With both logged out, and ws's signing key off gh's account, a run with no terminal must open
# nothing. A run of the wizard at a terminal of script's must open github.com/login/device and run
# gh auth login in a session of ssh -t. Then it must run bootstrap.sh --tags devtools, which adds
# the key, and claude auth login through bash -lc in another session, and list what stays by hand.
# Each login of the stand-ins takes a terminal. The next run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
ws=$RUNNER_TEMP/ws
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# A stand-in open, first on the wizard's PATH, logs its arguments, where macOS's would open a
# browser on the runner. macOS puts /usr/local/bin first on a login shell's PATH, so
# the stand-in claude goes there for claude's session, as root's ~/.local/bin would on ws.
if [ -e /usr/local/bin/claude ]; then
    echo '::error::the runner has a claude in /usr/local/bin already'
    exit 1
fi
cat >/opt/homebrew/bin/open <<EOF
#!/bin/sh
echo "\$*" >>'$RUNNER_TEMP/open'
EOF
chmod 755 /opt/homebrew/bin/open
sudo ln -s "$(cd "$(dirname "$0")" && pwd)/ws-standin.sh" /usr/local/bin/claude
rm "$ws/gh" "$ws/claude"
: >"$ws/gh.keys"
sh "$wizard" </dev/null >/dev/null 2>&1
[ ! -s "$RUNNER_TEMP/open" ] || error "wizard.sh with no terminal opened $(cat "$RUNNER_TEMP/open")"
lines=$(wc -l <"$ws/log")
# script gives the wizard a terminal of its own, which each session of ssh -t takes.
{ sleep 5; } | script -q /dev/null sh "$wizard" >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
tail -n "+$((lines + 1))" "$ws/log" | tee "$RUNNER_TEMP/sessions"
[ "$status" = 0 ] || error "wizard.sh at a terminal exited $status"
[ "$(cat "$RUNNER_TEMP/open" 2>/dev/null)" = https://github.com/login/device ] ||
    error "wizard.sh opened other than github.com/login/device, once"
for want in 'gh auth login -h github.com -p https -w -s write:ssh_signing_key' \
    'sh -s -- run --tags devtools' 'exec bash -lc "claude auth login"'; do
    grep -qxF "root: $want" "$RUNNER_TEMP/sessions" || error "no session on ws ran '$want'"
done
grep -qF 'What stays by hand:' "$RUNNER_TEMP/tty" &&
    grep -qxF "$(cut -d ' ' -f 1-2 "$ws/signing.pub")" "$ws/gh.keys" &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags devtools' ] &&
    [ -f "$ws/gh" ] && [ -f "$ws/claude" ] ||
    error "wizard.sh left gh or claude logged out on ws, or the key off, or nothing by hand"
rm -f /opt/homebrew/bin/open "$RUNNER_TEMP/open" "$RUNNER_TEMP/sessions"
sudo rm /usr/local/bin/claude
out=$(sh "$wizard" </dev/null 2>&1)
status=$?
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status: '$out'"
exit "$fail"
