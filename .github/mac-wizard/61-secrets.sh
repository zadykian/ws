#!/bin/bash
# The secrets, on the Mac that the cases before leave, with a throwaway keychain as the default one.
# With the Mac's file, a run must start one unit that loads the identity as the credential
# ws-secrets, with Type=exec, and runs ws server apply --secrets on ws's copy of the file. The
# stand-in ws must find the Mac's file and the Keychain's identity in it, both modes 600 and the
# identity on a tmpfs path, which the run removes. The run's stamp must name the file's SHA-256. A
# second run must start none, a changed file must start one, and a removed one must start one that
# runs with no --secrets, with ws's copy gone and the stamp naming none. No output may hold a key
# or a value. The keychain and the file go back.

set -u

wsbin=$HOME/.local/bin/ws
file=$HOME/.config/ws/secrets.age
ws=$RUNNER_TEMP/ws
state=$ws/root/var/lib/ws-wizard
keychain=$RUNNER_TEMP/wizard-secrets.keychain-db
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
    out=$("$wsbin" mac wizard </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
    grep -qF -e "$value" -e AGE-SECRET-KEY <<<"$out" && error "the wizard printed a key or a value"
}

# started N: the run before must have started N units.
started() {
    [ "$(wc -l <"$ws/run.log")" -eq $((units + $1)) ]
}

if [ -e "$file" ]; then
    echo "::error::$file is there before the case"
    exit 1
fi
list=$(security list-keychains -d user | tr -d '"')
default=$(security default-keychain -d user | tr -d '" ')
security create-keychain -p ws "$keychain"
security set-keychain-settings "$keychain"
security unlock-keychain -p ws "$keychain"
# shellcheck disable=SC2086 # the keychains' paths hold no blank
security list-keychains -d user -s "$keychain" $list
security default-keychain -d user -s "$keychain"

"$wsbin" mac secrets init >/dev/null 2>&1 || error "ws mac secrets init failed"
printf '%s\n' "$value" | "$wsbin" mac secrets set env MY_TOKEN >/dev/null 2>&1 || error "ws mac secrets set failed"
sudo rm -rf "$ws/seen" "$ws/creds"

wizard
last=$(tail -n 1 "$ws/run.log")
[ "$status" = 0 ] && started 1 && [[ $last == '-p LoadCredential=ws-secrets:'*' -p Type=exec '* ]] ||
    error "the run with the file exited $status, or started no unit that loads ws-secrets with Type=exec"
[ "$(tail -n 1 "$ws/server.log")" = "--secrets $state/secrets.age" ] ||
    error "the unit ran ws server apply other than with --secrets and ws's copy of the file"
sudo cmp -s "$file" "$ws/seen/secrets.age" || error "the unit saw another file than the Mac's"
sudo cmp -s "$file" "$state/secrets.age" || error "ws's copy of the file is not the Mac's"
security find-generic-password -s ws-secrets -w | sudo cmp -s - "$ws/seen/identity" ||
    error "the credential is not the Keychain's identity"
[ "$(sudo stat -f %Lp "$state/secrets.age")" = 600 ] || error "ws's copy has a mode other than 600"
[ "$(sudo stat -f %Lp "$ws/root/run/ws-wizard")" = 700 ] || error "the identity's directory has a mode other than 700"
sudo test ! -e "$ws/root/run/ws-wizard/identity" || error "the run left the identity on ws"
[ "$(sudo cut -d ' ' -f 4 "$state/run")" = "$(shasum -a 256 "$file" | cut -d ' ' -f 1)" ] ||
    error "the run's stamp does not name the file's SHA-256"

wizard
[ "$status" = 0 ] || error "a second run exited $status"
started 0 || error "a second run started a unit"

printf '%s\n' "$value" | "$wsbin" mac secrets set env OTHER >/dev/null 2>&1 || error "the second set failed"
wizard
[ "$status" = 0 ] && started 1 && [ "$(sudo cut -d ' ' -f 4 "$state/run")" = "$(shasum -a 256 "$file" | cut -d ' ' -f 1)" ] ||
    error "a changed file exited $status, or started no unit that stamps its sum"

rm -f "$file"
wizard
[ "$status" = 0 ] && started 1 && [ "$(tail -n 1 "$ws/server.log")" = '' ] &&
    sudo test ! -e "$state/secrets.age" && [ "$(sudo cut -d ' ' -f 4 "$state/run")" = - ] ||
    error "the run without the file exited $status, or ran with --secrets, or left ws's copy or a sum"

# shellcheck disable=SC2086 # as above
security list-keychains -d user -s $list
security default-keychain -d user -s "$default"
security delete-keychain "$keychain"
sudo rm -rf "$ws/seen" "$ws/creds"
exit "$fail"
