#!/bin/sh
# The Cable stage, on the Mac that 48-hook-mac.sh leaves. ws's ports are a sysfs of the stand-in's,
# as in 41-remote.sh. While no port has carrier, a run and --check must print nothing, and a run
# read link once. The closing list of --update must name the dock, and enp7s0, which is down, for
# WS_CABLE_PORT. WS_CABLE_PORT must end the wait: enp7s0 must go in host_vars, and lack 10.77.0.1
# after the roles' run. The uplink there must stop a run, and none write an empty WS_CABLE_SERVICE
# alone. Once enp5s0 has carrier, a run with no terminal must stop, naming WS_CABLE_PORT, and
# --check say what it would do, neither writing ws's host_vars. With WS_CABLE_PORT, a run must name
# enp5s0 there, run bootstrap.sh --tags network,wireguard, and stop at the Mac's service.
#
# The Mac's services are a stand-in networksetup's, and ping a stand-in's. Typed at a terminal of
# script's after a bad one, an empty line must take the one service with a 169.254 address. setup.sh
# must give it 10.77.0.2, and ping reach 10.77.0.1, once. The next run must print nothing. With no
# answer from ping, a run must stop, naming WS_CABLE_SERVICE. WS_CABLE_SERVICE= must write that
# empty line, which the next run keeps. mac.conf and the stand-ins go back, and a run must print
# nothing.

set -u

wizard=$HOME/repository/ws/mac/wizard.sh
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
cable=$RUNNER_TEMP/cable
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

# logged FILE prints how many lines the stand-in FILE in cable holds.
logged() {
    cat "$cable/$1" 2>/dev/null | wc -l | tr -d ' '
}

cp "$conf" "$RUNNER_TEMP/cable.conf"
for each in enp4s0:1 enp5s0:0 enp6s0:0 enp7s0; do
    sudo mkdir -p "$net/${each%:*}/device"
    echo 1 | sudo tee "$net/${each%:*}/type" >/dev/null
    [ "$each" = "${each%:*}" ] || echo "${each#*:}" | sudo tee "$net/${each%:*}/carrier" >/dev/null
done
echo enp4s0 >"$ws/uplink"
# The Mac's services: an uplink's, and the display's adapter, which gives itself a 169.254 address
# where no DHCP server answers. networksetup -setmanual takes root alone, as through sudo, and no
# other service, so that no case reaches the runner's own.
mkdir -p "$cable"
printf '%s\n' 'DHCP Configuration' 'IP address: 198.51.100.20' 'Subnet mask: 255.255.255.0' \
    'Router: 198.51.100.1' >"$cable/info-ws_stand_in_Wi_Fi"
printf '%s\n' 'DHCP Configuration' 'IP address: 169.254.10.20' 'Subnet mask: 255.255.0.0' \
    'Router: (null)' >"$cable/info-ws_stand_in_adapter"
cat >/opt/homebrew/bin/networksetup <<EOF
#!/bin/sh
info='$cable'/info-\$(printf '%s' "\${2-}" | tr -c 'A-Za-z0-9' _)
case \$1:\${2-} in
-listallnetworkservices:)
    printf '%s\n' 'An asterisk (*) denotes that a network service is disabled.' \
        'ws stand-in Wi-Fi' 'ws stand-in adapter'
    ;;
-getinfo:'ws stand-in '*) cat "\$info" ;;
-setmanual:'ws stand-in adapter')
    [ "\$(id -u)" = 0 ] || exit 1
    echo "\$*" >>'$cable/setmanual.log'
    printf '%s\n' 'Manual Configuration' "IP address: \$3" "Subnet mask: \$4" 'Router: (null)' \
        >"\$info"
    ;;
*) echo "\${2-} is not a recognized network service." && exit 4 ;;
esac
EOF
# ping answers for 10.77.0.1 where both ends of the cable have their addresses, unless cable holds
# a file silent.
cat >/opt/homebrew/bin/ping <<EOF
#!/bin/sh
echo "\$*" >>'$cable/ping.log'
[ "\$*" = '-c 1 -t 3 10.77.0.1' ] && [ ! -e '$cable/silent' ] &&
    grep -qx 'IP address: 10.77.0.2' '$cable/info-ws_stand_in_adapter' &&
    grep -qx 10.77.0.1/30 '$ws/addr/enp5s0' 2>/dev/null
EOF
chmod 755 /opt/homebrew/bin/networksetup /opt/homebrew/bin/ping

links=$(grep -c '^root: sh -s -- link$' "$ws/log")
wizard run
[ "$status" = 0 ] && [ -z "$out" ] &&
    [ "$(grep -c '^root: sh -s -- link$' "$ws/log")" = $((links + 1)) ] ||
    error "wizard.sh with no port with carrier exited $status, or read link other than once"
wizard --check
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh --check with no carrier exited $status"
wizard --update
printf '%s\n' "$out" | grep -qx -e "- The cable: dock the Mac at the display .* steps\. Where the \
cable plugs into enp7s0, which ws keeps down, run the line with WS_CABLE_PORT=PORT in the \
environment\. Without the cable, run the line once with WS_CABLE_SERVICE= in the environment \
(docs/mac-wizard\.md, The cable)\." && [ "$status" = 0 ] ||
    error "wizard.sh --update undocked exited $status, or named no dock, or no enp7s0"
wizard run WS_CABLE_PORT=enp4s0
printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: WS_CABLE_PORT comes from the environment, and \
'enp4s0' is none of enp5s0, enp6s0, enp7s0: set it to one, or to nothing for none" &&
    [ "$status" = 1 ] && [ ! -e "$vars" ] || error "wizard.sh with the uplink's port exited $status"
wizard run WS_CABLE_PORT=
[ "$status" = 0 ] && [ "$(tail -n 1 "$conf")" = WS_CABLE_SERVICE= ] && [ ! -e "$vars" ] ||
    error "wizard.sh with WS_CABLE_PORT= exited $status, or wrote another line, or host_vars"
cp "$RUNNER_TEMP/cable.conf" "$conf"
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
    printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: no terminal to ask WS_CABLE_SERVICE: set \
WS_CABLE_SERVICE, or set it to nothing for none" && [ "$status" = 1 ] &&
    [ "$(tail -n 1 "$ws/bootstrap.log")" = '--tags network,wireguard' ] ||
    error "wizard.sh with WS_CABLE_PORT exited $status, or wrote no host_vars, or ran no roles"
# On macOS, sudo, which the stages' checks run, takes what the terminal holds, so the lines go in
# once the question shows, or after 120 s. script's input ends 5 s after them, once it wrote them.
# shellcheck disable=SC2094 # the loop reads the question that script writes to tty, by design
{
    for _ in $(seq 120); do
        ! grep -qF '(WS_CABLE_SERVICE) [' "$RUNNER_TEMP/tty" 2>/dev/null || break
        sleep 1
    done
    printf '%s\n' 'ws stand-in absent' ''
    sleep 5
} | script -q /dev/null sh "$wizard" >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
grep -qF "(WS_CABLE_SERVICE) [ws stand-in adapter, or none]: " "$RUNNER_TEMP/tty" &&
    grep -qF "wizard.sh: Cable: 'ws stand-in absent' is none of those" "$RUNNER_TEMP/tty" &&
    [ "$status" = 0 ] && [ "$(tail -n 1 "$conf")" = 'WS_CABLE_SERVICE="ws stand-in adapter"' ] &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 1' ] ||
    error "wizard.sh at a terminal exited $status, or set the adapter or pinged other than once"
wizard run
[ "$status" = 0 ] && [ -z "$out" ] || error "the next run exited $status"
cp "$RUNNER_TEMP/cable.conf" "$conf"
touch "$cable/silent"
wizard run "WS_CABLE_SERVICE=ws stand-in adapter"
printf '%s\n' "$out" | grep -qx "wizard\.sh: Cable: ping -c 1 -t 3 10\.77\.0\.1, ws's end of the \
cable, got no answer: mac\.conf's WS_CABLE_SERVICE must name the service of the adapter that the \
cable plugs into" && [ "$status" = 1 ] &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 2' ] ||
    error "wizard.sh with no answer from ping exited $status, or pinged other than once"
rm "$cable/silent"
cp "$RUNNER_TEMP/cable.conf" "$conf"
wizard run WS_CABLE_SERVICE=
[ "$status" = 0 ] && [ "$(tail -n 1 "$conf")" = WS_CABLE_SERVICE= ] &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 2' ] ||
    error "wizard.sh with WS_CABLE_SERVICE= exited $status, or wrote another line, or acted"
wizard run
[ "$status" = 0 ] && [ -z "$out" ] || error "the run after none exited $status"

cp "$RUNNER_TEMP/cable.conf" "$conf"
rm -rf /opt/homebrew/bin/networksetup /opt/homebrew/bin/ping "$RUNNER_TEMP/cable.conf" \
    "$RUNNER_TEMP/tty" "$ws/uplink"
sudo rm -rf "$cable" "$ws/root/sys" "$ws/addr" "${vars%/*}"
wizard run
[ "$status" = 0 ] && [ -z "$out" ] || error "wizard.sh after it exited $status"
exit "$fail"
