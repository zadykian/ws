#!/bin/sh
# ws's run on the Mac that 33-root.sh leaves. The first run must have run ws's playbook once, and
# each run of the wizard since must have read ws's status alone. A run must read it once, and once
# more after a run of the playbook. Once ws's run stamp is gone, a run whose unit is cut off before
# its stamp must stop at the stage. The next must start a unit of its own. That one must end with
# what stays by hand: ws's fans' driver, as the stand-in's Secure Boot refuses it, and the router's
# forwards, for WS_REMOTE. A run while another's unit runs, as one that a dropped ssh left, must
# follow that unit, and start none. With WS_TARGET=cloud, a run that follows such a unit, without
# -e, must then start one with -e security_outbound=false, and name that in ws's host_vars. The next
# run must start none. Without the line, a run must write it alone. An older wizard's stamp of two
# words must read as a full run's. --check --update, with no WS_TARGET in the environment, and
# --check on a cloud VM must name the run that each would start.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
remote_sh=$HOME/repository/ws/mac/wizard-remote.sh
ws=$RUNNER_TEMP/ws
stamp=$ws/root/var/lib/ws-wizard/run
vars=$ws/root/root/repository/ws/host_vars/localhost.yml
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard: a run of the wizard with no terminal. It prints its stderr, then its stdout, the closing
# list, and sets status. It sets units and reads to the units that systemd-run started before it,
# and to the reads of ws's status before it.
wizard() {
    units=$(wc -l <"$ws/run.log")
    reads=$(grep -c '^root: sh -s -- status$' "$ws/log")
    sh "$wizard" </dev/null >"$RUNNER_TEMP/run.out" 2>"$RUNNER_TEMP/run.err"
    status=$?
    cat "$RUNNER_TEMP/run.err" "$RUNNER_TEMP/run.out"
}

# started N READS: the run before must have started N units, and read ws's status READS times.
started() {
    [ "$(wc -l <"$ws/run.log")" -eq $((units + $1)) ] &&
        [ "$(grep -c '^root: sh -s -- status$' "$ws/log")" -eq $((reads + $2)) ]
}

grep -q '^root: sh -s -- status$' "$ws/log" &&
    [ "$(grep -c '^root: sh -s -- run$' "$ws/log")" = 1 ] ||
    error "the wizard ran ws's playbook other than once, or read ws's status no time since"
sudo rm "$stamp"
touch "$ws/bootstrap.cut" "$ws/sb" "$ws/nct6687"
wizard
grep -q "^wizard\.sh: ws's run: ws's playbook stopped, " "$RUNNER_TEMP/run.err" && started 1 1 &&
    [ "$status" = 1 ] && [ ! -f "$stamp" ] || error "a run cut off exited $status, or stamped run"
rm "$ws/bootstrap.cut"
wizard
grep -q ' 0 on$' "$stamp" && started 1 2 && [ "$status" = 0 ] || error "the next run exited $status"
if ! grep -q "^- ws's fans' driver: " "$RUNNER_TEMP/run.out" ||
    ! grep -q "^- The router's forwards to ws, UDP 51820 and TCP 22022, for localhost," \
        "$RUNNER_TEMP/run.out"; then
    error "the run after it named no fans' driver or forwards by hand"
fi
rm "$ws/sb" "$ws/nct6687"

sudo rm "$stamp"
# Its unit runs 20 s, as the wizard takes some seconds to reach its stage, more on a slow runner.
echo 20 >"$ws/bootstrap.wait"
ssh -o BatchMode=yes ws-ssh sh -s -- run <"$remote_sh" >/dev/null 2>&1 &
first=$!
tries=0
until ls "$ws"/units/*.on >/dev/null 2>&1 || [ "$tries" = 20 ]; do
    sleep 1
    tries=$((tries + 1))
done
# The ssh drops, and its unit goes on.
kill "$first"
wizard
grep -q '^wizard-remote\.sh: following ws-wizard-' "$RUNNER_TEMP/run.err" &&
    grep -q ' 0 on$' "$stamp" && started 0 2 && [ "$status" = 0 ] ||
    error "a run during a dropped ssh's unit exited $status, or started one"
rm "$ws/bootstrap.wait"
sudo sh -c 'cut -d " " -f 1-2 "$1" >"$1.old" && mv "$1.old" "$1"' sh "$stamp"
wizard
started 0 1 && [ "$status" = 0 ] || error "a run on an older stamp of two words exited $status"
out=$(env -u WS_TARGET sh "$wizard" --check --update </dev/null 2>&1)
printf '%s\n' "$out" | grep -q "^wizard\.sh: ws's run: would pull ws's clone " ||
    error "wizard.sh --check --update named no run of the playbook: $out"

WS_CONF=$RUNNER_TEMP/cloud.conf
export WS_CONF
sed 's/^WS_TARGET=.*/WS_TARGET=cloud/' ~/.config/ws/mac.conf >"$WS_CONF"
out=$(sh "$wizard" --check </dev/null 2>&1)
printf '%s\n' "$out" | grep -q 'with -e security_outbound=false, then name that in ' ||
    error "wizard.sh --check on a cloud VM named no run with -e: $out"
echo 20 >"$ws/bootstrap.wait"
ssh -o BatchMode=yes ws-ssh sh -s -- run <"$remote_sh" >/dev/null 2>&1 &
tries=0
until ls "$ws"/units/*.on >/dev/null 2>&1 || [ "$tries" = 20 ]; do
    sleep 1
    tries=$((tries + 1))
done
wizard
rm "$ws/bootstrap.wait"
grep -q '^wizard-remote\.sh: following ws-wizard-' "$RUNNER_TEMP/run.err" &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '-e security_outbound=false' ] && started 1 3 &&
    grep -q ' 0 off$' "$stamp" && sudo grep -qx 'security_outbound: false' "$vars" &&
    [ "$status" = 0 ] || error "a cloud VM's run after a unit without -e exited $status"
wizard
started 0 1 && [ "$status" = 0 ] || error "the next run with WS_TARGET=cloud exited $status"
sudo rm "$vars"
wizard
started 0 2 && sudo grep -qx 'security_outbound: false' "$vars" && [ "$status" = 0 ] ||
    error "a cloud VM's run without the line exited $status, or ran a playbook"
sudo rm -r "${vars%/*}"
rm "$WS_CONF"
unset WS_CONF
exit "$fail"
