#!/bin/sh
# The wizard's remote script on the stand-in ws, on the Mac that 40-again.sh leaves, sent to root
# over ssh as the wizard sends it. status must print the stamps and each state of the stand-ins:
# with wg0's key, the Mac's key for it, and Secure Boot on that refuses the fans' driver, then with
# gh's and claude's logins, and Secure Boot off. The stand-ins' state goes back as it was.

set -u

remote_sh=$HOME/repository/ws/mac/wizard-remote.sh
ws=$RUNNER_TEMP/ws
stamps=$ws/root/var/lib/ws-wizard
peer=$ws/root/etc/wireguard/mac.pub
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# remote OP [ARG...]: the remote script's OP on ws, as the wizard runs it. It prints the output, and
# sets out and status.
remote() {
    out=$(ssh -o BatchMode=yes ws-ssh sh -s -- "$@" <"$remote_sh" 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# stamp NAME prints the stamp NAME, or - where it is missing.
stamp() {
    cat "$stamps/$1" 2>/dev/null || echo -
}

# states WANT...: status must print the stamps, no unit that runs, then the lines WANT.
states() {
    remote status
    want=$(printf '%s\n' "run $(stamp run)" "last $(stamp last)" 'active -' "$@")
    [ "$status" = 0 ] && [ "$out" = "$want" ] ||
        error "status exited $status, printing other than '$want'"
}

key=$(head -c 32 /dev/urandom | base64)
mac=$(head -c 32 /dev/urandom | base64)
echo "$key" >"$ws/wg0"
touch "$ws/sb" "$ws/nct6687"
sudo mkdir -p "${peer%/*}"
echo "$mac" | sudo tee "$peer" >/dev/null
states "mac.pub $mac" "wg0 $key" 'gh login' 'claude login' 'fans refused'
touch "$ws/gh" "$ws/claude"
rm "$ws/sb" "$ws/wg0"
sudo rm -r "${peer%/*/*}"
states 'mac.pub -' 'wg0 -' 'gh key' 'claude ok' 'fans ok'
rm "$ws/gh" "$ws/claude" "$ws/nct6687"
exit "$fail"
