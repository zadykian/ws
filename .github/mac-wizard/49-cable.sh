#!/bin/sh
# The Cable stage, on the Mac that 48-hook-mac.sh leaves. ws's ports are a sysfs of the stand-in's,
# as in 41-remote.sh. While no port has carrier, a run and --check must print nothing, and a run
# read link once, and --update's closing list name the dock. WS_CABLE_PORT must end the wait:
# enp7s0, which is down, must go in host_vars, and lack 10.77.0.1 after the roles' run.
# Once enp5s0 has carrier, a run with no terminal must stop, naming WS_CABLE_PORT, and --check say
# what it would do, neither writing ws's host_vars. With WS_CABLE_PORT, a run must name enp5s0
# there, run bootstrap.sh --tags network,wireguard, and stop at the Mac's service.
#
# WS_CABLE_SERVICE= must write that empty line, which the next run keeps. mac.conf and the stand-ins
# go back, and a run must print nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
net=$ws/root/sys/class/net
vars=$ws/root/root/repository/ws/host_vars/localhost.yml
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# wizard run|--check [NAME=VALUE...]: a run of the wizard, or --check, with no terminal, given the
# variables. It prints the output, and sets out and status.
wizard() {
    mode=${1#run}
    shift
    out=$(env "$@" sh "$wizard" ${mode:+"$mode"} </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

cp "$conf" "$RUNNER_TEMP/cable.conf"
for each in enp4s0:1 enp5s0:0 enp6s0:0 enp7s0; do
    sudo mkdir -p "$net/${each%:*}/device"
    echo 1 | sudo tee "$net/${each%:*}/type" >/dev/null
    [ "$each" = "${each%:*}" ] || echo "${each#*:}" | sudo tee "$net/${each%:*}/carrier" >/dev/null
done
echo enp4s0 >"$ws/uplink"

links=$(grep -c '^root: sh -s -- link$' "$ws/log")
wizard run
[ "$status" = 0 ] && [ -z "$out" ] &&
    [ "$(grep -c '^root: sh -s -- link$' "$ws/log")" = $((links + 1)) ] ||
    error "wizard.sh with no port with carrier exited $status, or read link other than once"
wizard --check
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh --check with no carrier exited $status"
wizard --update
printf '%s\n' "$out" | grep -qx -e "- The cable: dock the Mac at the display .* steps\. Without \
the cable, run the line once with WS_CABLE_SERVICE= in the environment (docs/mac-wizard\.md, The \
cable)\." && [ "$status" = 0 ] || error "wizard.sh --update undocked exited $status, or no dock"
wizard run WS_CABLE_PORT=enp7s0
printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: ws's enp7s0 lacks 10\.77\.0\.1 after \
bootstrap\.sh --tags network,wireguard: plug the Mac's cable into it, or mend what that printed, \
then run the wizard again" && grep -qx 'network_direct_link_interface: "enp7s0"' "$vars" &&
    [ "$status" = 1 ] || error "wizard.sh with WS_CABLE_PORT=enp7s0 exited $status, or no host_vars"
sudo rm -f "$vars"
echo 1 | sudo tee "$net/enp5s0/carrier" >/dev/null
wizard run
printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: no terminal to ask WS_CABLE_PORT: set \
WS_CABLE_PORT, or set it to nothing for none" && [ "$status" = 1 ] && [ ! -e "$vars" ] ||
    error "wizard.sh docked with no terminal exited $status, or named no WS_CABLE_PORT"
wizard --check
printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: would ask ws's port of the cable, name it \
in host_vars/localhost\.yml on ws, and run bootstrap\.sh --tags network,wireguard there" &&
    [ "$status" = 0 ] && [ ! -e "$vars" ] || error "wizard.sh --check docked exited $status"
wizard run WS_CABLE_PORT=enp5s0
printf '%s\n' --- 'network_direct_link_interface: "enp5s0"' | cmp -s - "$vars" &&
    cmp -s "$conf" "$RUNNER_TEMP/cable.conf" &&
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: ws's enp5s0 holds 10\.77\.0\.1: set \
WS_CABLE_SERVICE in mac\.conf to the Mac's service of the cable (docs/mac\.md, The cable), or to \
nothing for none" && [ "$status" = 1 ] &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags network,wireguard' ] ||
    error "wizard.sh with WS_CABLE_PORT exited $status, or wrote no host_vars, or ran no roles"
wizard run WS_CABLE_SERVICE=
[ "$status" = 0 ] && [ "$(tail -n 1 "$conf")" = WS_CABLE_SERVICE= ] ||
    error "wizard.sh with WS_CABLE_SERVICE= exited $status, or wrote another line"
wizard run
[ "$status" = 0 ] && [ -z "$out" ] || error "the run after none exited $status"

cp "$RUNNER_TEMP/cable.conf" "$conf"
rm -f "$RUNNER_TEMP/cable.conf" "$ws/uplink"
sudo rm -rf "$ws/root/sys" "$ws/addr" "${vars%/*}"
wizard run
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
