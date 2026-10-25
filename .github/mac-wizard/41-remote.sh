#!/bin/sh
# The wizard's remote script on the stand-in ws, on the Mac that 40-again.sh leaves, sent to root
# over ssh as the wizard sends it. status must print the stamps and each state of the stand-ins.
# First come wg0's key, a Mac's key it lacks, Secure Boot that refuses the fans' driver, and no
# logins. Then gh's and claude's logins, with ws's signing key listed, and Secure Boot off. A full
# run must run bootstrap.sh without arguments in a unit of its own, with root's HOME and a UTF-8
# locale, print its log and exit 0. It must stamp the clone's commit and 0 in run, and the unit and
# 0 in last. A tagged run after it must start its own unit, exit with its status, and leave run as
# it was. A run during another's unit must follow that unit and start none. A full run whose unit is
# cut off must exit 1 and remove the run stamp that the one before left. The next must start a unit
# of its own. The stand-ins' state goes back.

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
rm "$ws/gh" "$ws/claude"
states "mac.pub $mac" "wg0 $key" 'gh login' 'claude login' 'fans refused' 'peer -'
touch "$ws/gh" "$ws/claude"
rm "$ws/sb" "$ws/wg0"
sudo rm -r "${peer%/*/*}"
states 'mac.pub -' 'wg0 -' 'gh ok' 'claude ok' 'fans ok' 'peer -'
rm "$ws/nct6687"

# units prints how many units systemd-run started, and unit the last one's name.
units() {
    wc -l <"$ws/run.log"
}
unit() {
    sed -n 's/.* --unit=\([^ ]*\) .*/\1/p' "$ws/run.log" | tail -n 1
}

sha=$(sudo git -C "$ws/root/root/repository/ws" rev-parse HEAD)
n=$(units)
remote run
full=$(unit)
printf '%s\n' "$out" | grep -qx 'PLAY RECAP' && [ "$status" = 0 ] &&
    [ "$(units)" -eq $((n + 1)) ] && [ "$(stamp run) / $(stamp last)" = "$sha 0 / $full 0" ] &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '' ] || error "a full run exited $status"
tail -n 1 "$ws/run.log" | grep -q -- '--collect .* --setenv=LC_ALL=C.UTF-8 --setenv=HOME=/root ' ||
    error "a full run's unit lacked root's HOME or a UTF-8 locale"
echo 3 >"$ws/bootstrap.status"
remote run --tags devtools
[ "$status" = 3 ] && [ "$(units)" -eq $((n + 2)) ] && [ "$(unit)" != "$full" ] &&
    [ "$(stamp run) / $(stamp last)" = "$sha 0 / $(unit) 3" ] &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags devtools' ] ||
    error "a tagged run exited $status"
rm "$ws/bootstrap.status"
echo 4 >"$ws/bootstrap.wait"
ssh -o BatchMode=yes ws-ssh sh -s -- run <"$remote_sh" >"$RUNNER_TEMP/first" 2>&1 &
first=$!
tries=0
until ls "$ws"/units/*.on >/dev/null 2>&1 || [ "$tries" = 20 ]; do
    sleep 1
    tries=$((tries + 1))
done
n=$(units)
remote run --tags other
wait "$first"
first=$?
printf '%s\n' "$out" | grep -q '^wizard-remote\.sh: following ws-wizard-' &&
    ! grep -q other "$ws/bootstrap.log" && [ "$first" = 0 ] && [ "$status" = 0 ] &&
    [ "$(units)" -eq "$n" ] || error "a run during another's unit exited $status, or started one"
rm "$ws/bootstrap.wait" "$RUNNER_TEMP/first"
touch "$ws/bootstrap.cut"
remote run
printf '%s\n' "$out" | grep -q ' ended before its run wrote its status' && [ "$status" = 1 ] &&
    [ "$(stamp run)" = - ] || error "a full run cut off exited $status, leaving run '$(stamp run)'"
rm "$ws/bootstrap.cut"
n=$(units)
remote run
[ "$status" = 0 ] && [ "$(units)" -eq $((n + 1)) ] && [ "$(stamp run)" = "$sha 0" ] ||
    error "the run after it exited $status, with run '$(stamp run)'"
exit "$fail"
