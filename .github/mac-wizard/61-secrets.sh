#!/bin/bash
# ws's own secrets file and its identity's credential, on the stand-in ws that the cases before
# leave. With both, a run must start one unit that loads the credential ws-secrets, with Type=exec,
# and runs ws server ubuntu apply --secrets on ws's file. The stand-in ws must find the file and the
# credential in it. The run's stamp must name the file's SHA-256. A second run must start none, and
# a changed file must start one. The file without the credential must start one that runs with no
# --secrets, with the stamp naming none, and then none. No output may hold the file or the
# credential. ws client macos secrets must run ws server ubuntu secrets on ws, with a value on its
# stdin alone, and print what it prints.

set -u

wsbin=$HOME/.local/bin/ws
ws=$RUNNER_TEMP/ws
file="$ws/root"/etc/ws/secrets.age
credential="$ws/root"/etc/credstore.encrypted/ws-secrets
state=$ws/root/var/lib/ws-wizard
value='a stand-in value, for the case'
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard: a run of the wizard with no terminal. It prints the output, sets out and status, and sets
# units to the units that systemd-run started before it.
wizard() {
    units=$(wc -l <"$ws/run.log")
    out=$("$wsbin" client macos wizard </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    grep -qF "$value" <<<"$out" && error "the wizard printed the file or the credential"
}

# started N: the run before must have started N units.
started() {
    [ "$(wc -l <"$ws/run.log")" -eq $((units + $1)) ]
}

# stamped: the run's stamp must name the file's SHA-256.
stamped() {
    [ "$(sudo cut -d ' ' -f 4 "$state/run")" = "$(sudo shasum -a 256 "$file" | cut -d ' ' -f 1)" ]
}

if sudo test -e "$file" || sudo test -e "$credential"; then
    echo "::error::ws's secrets file or its credential is there before the case"
    exit 1
fi
sudo mkdir -p "${credential%/*}"
printf 'file: %s\n' "$value" | sudo tee "$file" >/dev/null
printf 'credential: %s\n' "$value" | sudo tee "$credential" >/dev/null
sudo rm -rf "$ws/seen" "$ws/creds"

wizard
last=$(tail -n 1 "$ws/run.log")
[ "$status" = 0 ] && started 1 && [[ $last == '-p LoadCredentialEncrypted=ws-secrets -p Type=exec '* ]] ||
    error "the run with the credential exited $status, or started no unit that loads ws-secrets with Type=exec"
[ "$(tail -n 1 "$ws/server.log")" = "--secrets $file" ] ||
    error "the unit ran ws server ubuntu apply other than with --secrets and ws's file"
sudo cmp -s "$file" "$ws/seen/secrets.age" || error "the unit saw another file than ws's"
sudo cmp -s "$credential" "$ws/seen/identity" || error "the unit saw another credential than ws's"
stamped || error "the run's stamp does not name the file's SHA-256"

wizard
[ "$status" = 0 ] || error "a second run exited $status"
started 0 || error "a second run started a unit"

echo more | sudo tee -a "$file" >/dev/null
wizard
started 1 && stamped && [ "$status" = 0 ] || error "a changed file exited $status, or started no unit that stamps its sum"

sudo rm -f "$credential"
wizard
[ "$status" = 0 ] && started 1 && [ "$(tail -n 1 "$ws/server.log")" = '' ] &&
    [ "$(sudo cut -d ' ' -f 4 "$state/run")" = - ] ||
    error "the file without the credential exited $status, or ran with --secrets, or stamped a sum"
wizard
started 0 && [ "$status" = 0 ] || error "a run with nothing changed exited $status, or started a unit"

"$wsbin" client macos secrets init && [ "$(tail -n 1 "$ws/secrets.log")" = init ] || error "init ran other than its own on ws"
out=$(printf '%s\n' "$value" | "$wsbin" client macos secrets set env MY_TOKEN 2>&1) || error "set env exited $?"
[ "$(tail -n 1 "$ws/secrets.log")" = 'set env -- MY_TOKEN' ] && [ "$(cat "$ws/secrets.in")" = "$value" ] ||
    error "set env ran other than ws server ubuntu secrets set env -- MY_TOKEN, with the value on stdin"
echo 'env MY_TOKEN' >"$ws/keys"
[ "$("$wsbin" client macos secrets keys 2>&1)" = 'env MY_TOKEN' ] || error "keys printed other than ws's keys"
grep -qF "$value" - "$ws/secrets.log" "$ws/log" "$ws/sudo.log" <<<"$out" && error "an argument or a log held the value"

sudo rm -rf "$file" "$ws/seen" "$ws/creds" "$ws/secrets.in" "$ws/keys"
exit "$fail"
