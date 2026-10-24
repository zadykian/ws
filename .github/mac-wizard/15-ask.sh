#!/bin/sh
# The Answers stage on a fresh Mac, which none of these runs may change. mac.conf stays as
# standin.sh wrote it, and no lock or file of the wizard's stays in $TMPDIR. With no terminal, a
# missing answer must stop a run and --check within 60 s, naming the key. A bad answer from the
# environment must stop --check, with the form the key takes. --check must show the environment's
# answers as the lines it would add to mac.conf. The same answers, typed at a terminal of script's
# after bad ones, must show the same lines, each bad one asked again with its form. Empty lines must
# take the defaults.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
tmp=${TMPDIR:-/tmp}
tmp=${tmp%/}
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# kept NAME: the run NAME must leave mac.conf and $TMPDIR as they were.
kept() {
    cmp -s "$conf" "$RUNNER_TEMP/mac.conf" || error "$1 changed mac.conf"
    for left in "$tmp/ws-wizard.lock" "$tmp"/ws.*; do
        [ ! -e "$left" ] || error "$1 left $left"
    done
}

# quick WHY COMMAND...: COMMAND, the wizard with no terminal, must exit 1 within 60 s, with WHY in
# what it prints. A watchdog ends it at 60 s, and its own sleep as it is ended.
quick() {
    why=$1
    shift
    "$@" </dev/null >"$RUNNER_TEMP/quick" 2>&1 &
    pid=$!
    (
        trap 'kill "$!"; exit' TERM
        sleep 60 &
        wait "$!"
        kill "$pid"
    ) >/dev/null 2>&1 &
    watch=$!
    wait "$pid"
    status=$?
    kill "$watch" 2>/dev/null
    out=$(cat "$RUNNER_TEMP/quick")
    if [ "$status" != 1 ] || ! printf '%s\n' "$out" | grep -qF "$why"; then
        error "$* exited $status, printing '$out', where 1 and '$why' were due"
    fi
    kept "$*"
}

# added FILE prints the lines that the diff of mac.conf adds, in the wizard's output FILE.
added() {
    tr -d '\r' <"$1" |
        awk -v f="+++ $conf" '$0 == f { on = 1; next } /^--- / { on = 0 } on && /^\+/'
}

# typed WHAT LINE...: wizard.sh --check, with the LINEs typed at a terminal of script's, must exit 0
# and show the lines of $RUNNER_TEMP/added. Its output, without script's carriage returns, goes in
# $RUNNER_TEMP/tty. macOS's script writes its stdin's end to the terminal ahead of the lines it
# still holds, so the end comes 5 s after them.
typed() {
    what=$1
    shift
    {
        printf '%s\n' "$@"
        sleep 5
    } | script -q /dev/null sh "$wizard" --check >"$RUNNER_TEMP/raw" 2>&1
    status=$?
    tr -d '\r' <"$RUNNER_TEMP/raw" >"$RUNNER_TEMP/tty"
    cat "$RUNNER_TEMP/tty"
    [ "$status" = 0 ] || error "wizard.sh --check with $what exited $status"
    added "$RUNNER_TEMP/tty" | cmp -s - "$RUNNER_TEMP/added" ||
        error "wizard.sh --check showed other lines for mac.conf than $what"
    kept "wizard.sh --check with $what"
}

cp "$conf" "$RUNNER_TEMP/mac.conf"
quick 'wizard.sh: Answers: no terminal to ask WS_LAN: set WS_LAN' sh "$wizard"
quick 'wizard.sh: Answers: no terminal to ask WS_LAN: set WS_LAN' sh "$wizard" --check
quick 'WS_LAN=10.77.0.0/30 is not an IPv4 network, as 10.88.0.0/24, clear of 10.77.0.0/30' \
    env WS_LAN=10.77.0.0/30 sh "$wizard" --check
quick "WS_LAN_ADDRESS=198.51.100.1 is not a host's IPv4 address inside WS_LAN, 192.0.2.0/24" \
    env WS_LAN=192.0.2.0/24 WS_LAN_ADDRESS=198.51.100.1 sh "$wizard" --check

printf '%s\n' '+WS_LAN=192.0.2.0/24' '+WS_LAN_ADDRESS=192.0.2.10' '+WS_GIT_NAME="Jane Doe"' \
    '+WS_GIT_EMAIL=jane@example.com' '+WS_TUNNEL_NETWORKS="198.51.100.0/24 203.0.113.0/24"' \
    '+WS_RESOLVERS=example.test=192.0.2.53' >"$RUNNER_TEMP/added"
env WS_LAN=192.0.2.0/24 WS_LAN_ADDRESS=192.0.2.10 WS_GIT_NAME='Jane Doe' \
    WS_GIT_EMAIL=jane@example.com WS_TUNNEL_NETWORKS='198.51.100.0/24 203.0.113.0/24' \
    WS_RESOLVERS=example.test=192.0.2.53 sh "$wizard" --check </dev/null >"$RUNNER_TEMP/env" 2>&1
status=$?
cat "$RUNNER_TEMP/env"
[ "$status" = 0 ] || error "wizard.sh --check with the environment's answers exited $status"
added "$RUNNER_TEMP/env" | cmp -s - "$RUNNER_TEMP/added" ||
    error "wizard.sh --check showed other lines for mac.conf than the environment's answers"
kept "wizard.sh --check with the environment's answers"

typed 'the answers typed' 10.77.0.0/30 192.0.2.0/24 198.51.100.1 '' 192.0.2.10 '"Jane Doe"' \
    jane@ jane@example.com '198.51.100.0/24 203.0.113.0/24' example.test=192.0.2.53
for asked in 'WS_LAN=10.77.0.0/30 is not an IPv4 network' \
    "WS_LAN_ADDRESS=198.51.100.1 is not a host's IPv4 address inside WS_LAN, 192.0.2.0/24" \
    'Answers: WS_LAN and WS_LAN_ADDRESS go together' 'WS_GIT_EMAIL=jane@ is not an email address' \
    "ws's home LAN (WS_LAN) [10.88.0.0/24, or none]: " \
    "ws's address on the LAN (WS_LAN_ADDRESS): "; do
    grep -qF "$asked" "$RUNNER_TEMP/tty" || error "wizard.sh asked with no '$asked'"
done
[ "$(grep -oF "(WS_LAN_ADDRESS): " "$RUNNER_TEMP/tty" | wc -l)" -eq 3 ] ||
    error 'wizard.sh asked WS_LAN_ADDRESS other than 3 times, for 2 bad answers'

# The defaults: the router's LAN and lease (docs/router.md), and the shell role's git identity.
printf '%s\n' '+WS_LAN=10.88.0.0/24' '+WS_LAN_ADDRESS=10.88.0.10' '+WS_GIT_NAME=zadykian' \
    '+WS_GIT_EMAIL=me@zadykian.dev' '+WS_TUNNEL_NETWORKS=' '+WS_RESOLVERS=' >"$RUNNER_TEMP/added"
typed 'six empty lines' '' '' '' '' '' ''
exit "$fail"
