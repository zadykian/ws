#!/bin/sh
# The Cable stage, on the Mac that 46-gh.sh leaves. ws's ports are a sysfs of the stand-in's,
# as in 41-remote.sh. While no port has carrier, a run and --check must print nothing, and a run
# read link once. The closing list of --update must name the dock, and enp7s0, which is down, for
# WS_CABLE_PORT. WS_CABLE_PORT must end the wait: enp7s0 must go in host_vars, and lack 10.77.0.1
# after the roles' run. The uplink there must stop a run, and none write an empty WS_CABLE_SERVICE
# alone. Once enp5s0 has carrier, a run with no terminal must stop, naming WS_CABLE_PORT, and
# --check say what it would do, neither writing ws's host_vars. With WS_TARGET=cloud, a run must
# skip the stage, and its closing list name the provider's firewall, but neither the dock nor the
# router's forwards. With WS_TARGET=mac-vm, --check must skip it too. With WS_CABLE_PORT, a run
# must name enp5s0 there, and run ws server ubuntu apply --only network,wireguard. It must read its
# late 10.77.0.1 in one login of the files script's wait, and stop at the Mac's service.
#
# The Mac's services are a stand-in networksetup's, and ping a stand-in's. Typed at a terminal of
# script's after a bad one, an empty line must take the one service with a 169.254 address. ws
# client macos apply must give it 10.77.0.2, and ping reach 10.77.0.1, once, and mac.conf keep
# enp5s0 as WS_CABLE_PORT. The next run must print nothing, and read no link. Without host_vars'
# port and the roles' netplan file, as after ws's install, and undocked, --check must wait on ws's
# run. A run must name enp5s0 there again, run the roles, and leave mac.conf. Without the netplan
# file alone, as after a roles' run that failed, a run must run them again. With no answer from
# ping, a run must stop, naming WS_CABLE_SERVICE. WS_CABLE_SERVICE= must write that empty line,
# which the next run keeps. mac.conf and the stand-ins go back, and after a run of ws server ubuntu
# apply, as host_vars lost the port, a run must print nothing. A run after host_vars changed since
# the last full run runs that first.

set -u
# shellcheck source=.github/mac-conf.sh
. "$(dirname "$0")/../mac-conf.sh"

# The one line of the gh stage that a run says where the account holds the Mac's key.
known="ws client macos wizard: gh on the Mac: the Mac's key is among GitHub's authentication keys already"

wsbin=$HOME/.local/bin/ws
conf=$HOME/.config/ws/mac.conf
ws=$RUNNER_TEMP/ws
cable=$RUNNER_TEMP/cable
net=$ws/root/sys/class/net
vars="$ws/root"/etc/ws/host_vars.yml
netplan=$ws/root/etc/netplan/60-direct-link.yaml
fail=0

# unnamed: host_vars names no port, whatever else it holds: a run's ws_user stays there.
unnamed() {
    ! sudo grep -qs '^network_direct_link_interface:' "$vars"
}

# unname takes out of host_vars what a new install lacks, and keeps ws_user.
unname() {
    unnamed || sudo sed -i '' -e '/^network_direct_link_interface:/d' \
        -e '/^security_outbound:/d' "$vars"
}

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
    out=$(env "$@" "$wsbin" client macos wizard ${mode:+"$mode"} </dev/null 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# logged FILE prints how many lines the stand-in FILE in cable holds.
logged() {
    cat "$cable/$1" 2>/dev/null | wc -l | tr -d ' '
}

conf_print "$conf" >"$RUNNER_TEMP/cable.conf"
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
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] &&
    [ "$(grep -c '^root: sh -s -- link$' "$ws/log")" = $((links + 1)) ] ||
    error "ws client macos wizard with no port with carrier exited $status, or read link other than once"
wizard --check
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "ws client macos wizard --check with no carrier exited $status"
wizard --update
printf '%s\n' "$out" | grep -qx -e "- The cable: dock the Mac at the display .* steps\. Where the \
cable plugs into enp7s0, which ws keeps down, run the line with WS_CABLE_PORT=PORT in the \
environment\. Without the cable, run the line once with WS_CABLE_SERVICE= in the environment \
(roles/client/macos/stages/README\.md, The cable)\." && [ "$status" = 0 ] ||
    error "ws client macos wizard --update undocked exited $status, or named no dock, or no enp7s0"
wizard run WS_CABLE_PORT=enp4s0
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: WS_CABLE_PORT comes from the environment, and \
'enp4s0' is none of enp5s0, enp6s0, enp7s0: set it to one, or to nothing for none" &&
    unnamed && [ "$status" = 1 ] || error "ws client macos wizard with the uplink's port exited $status"
wizard run WS_CABLE_PORT=
unnamed && [ "$status" = 0 ] && [ "$(conf_print "$conf" | tail -n 1)" = WS_CABLE_SERVICE= ] ||
    error "ws client macos wizard with WS_CABLE_PORT= exited $status, or wrote another line, or host_vars"
conf_write "$conf" <"$RUNNER_TEMP/cable.conf"
wizard run WS_CABLE_PORT=enp7s0
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: ws's enp7s0 lacks 10\.77\.0\.1 after \
ws server ubuntu apply --only network,wireguard: plug the Mac's cable into it, or mend what that printed, \
then run the wizard again" && sudo grep -qx 'network_direct_link_interface: "enp7s0"' "$vars" &&
    [ "$status" = 1 ] || error "ws client macos wizard with WS_CABLE_PORT=enp7s0 exited $status, or no host_vars"
unname
echo 1 | sudo tee "$net/enp5s0/carrier" >/dev/null
wizard run
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: no terminal to ask WS_CABLE_PORT: set \
WS_CABLE_PORT, or set it to nothing for none" && unnamed && [ "$status" = 1 ] ||
    error "ws client macos wizard docked with no terminal exited $status, or named no WS_CABLE_PORT"
conf_print "$conf" | sed -e 's/^WS_TARGET=.*/WS_TARGET=cloud/' -e '/^WS_CABLE_/d' |
    conf_write "$RUNNER_TEMP/confs/cloud/mac.conf"
wizard run WS_CONF="$RUNNER_TEMP/confs/cloud/mac.conf"
firewall="- The provider's firewall: UDP 51820 open to ws, for the tunnel (roles/client/macos/conf/README.md)."
if [ "$status" != 0 ] || ! printf '%s\n' "$out" | grep -qxF -e "$firewall" ||
    printf '%s\n' "$out" | grep -q -e '^ws client macos wizard: Cable: ' -e '^- The cable: ' \
        -e '^- The router' || sudo grep -q '^network_direct_link_interface:' "$vars"; then
    error "ws client macos wizard with WS_TARGET=cloud exited $status, or took the cable's steps"
fi
unname
rm -r "$RUNNER_TEMP/confs/cloud"
conf_print "$conf" | sed -e 's/^WS_TARGET=.*/WS_TARGET=mac-vm/' -e 's|^WS_LAN=.*|WS_LAN=192.0.2.0/24|' \
    -e '/^WS_REMOTE/d' -e '/^WS_CABLE_/d' | conf_write "$RUNNER_TEMP/confs/vm/mac.conf"
conf_set "$RUNNER_TEMP/confs/vm/mac.conf" WS_LAN_ADDRESS=192.0.2.10
wizard --check WS_CONF="$RUNNER_TEMP/confs/vm/mac.conf"
if [ "$status" != 0 ] || printf '%s\n' "$out" | grep -q '^ws client macos wizard: Cable: '; then
    error "ws client macos wizard --check with WS_TARGET=mac-vm exited $status, or took the cable's steps"
fi
rm -r "$RUNNER_TEMP/confs/vm"
wizard --check
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: would ask ws's port of the cable, name it \
in /etc/ws/host_vars\.yml on ws, and run ws server ubuntu apply --only network,wireguard there" &&
    unnamed && [ "$status" = 0 ] || error "ws client macos wizard --check docked exited $status"
# ws's ip, the stand-in's, but that the first read of enp5s0's addresses after the roles' run finds
# none. On ws, networkd likewise gives it 10.77.0.1 a moment after their networkctl reload. The
# files script's wait must read ip again in the same login, and find the address.
mkdir "$cable/bin"
mv "$ws/bin/ip" "$cable/bin/ip"
cat >"$ws/bin/ip" <<EOF
#!/bin/sh
[ "\$*" != '-o -4 address show dev enp5s0' ] || [ ! -e '$ws/addr/enp5s0' ] ||
    [ -e '$cable/late' ] || { touch '$cable/late' && exit 0; }
exec '$cable/bin/ip' "\$@"
EOF
chmod 755 "$ws/bin/ip"
links=$(grep -c '^root: sh -s -- link$' "$ws/log")
wizard run WS_CABLE_PORT=enp5s0
sudo grep -qx 'network_direct_link_interface: "enp5s0"' "$vars" &&
    conf_print "$conf" | cmp -s - "$RUNNER_TEMP/cable.conf" &&
    printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: no terminal to ask WS_CABLE_SERVICE: set \
WS_CABLE_SERVICE, or set it to nothing for none" && [ "$status" = 1 ] &&
    [ "$(tail -n 1 "$ws/server.log")" = '--only network,wireguard' ] && [ -e "$cable/late" ] &&
    [ "$(grep -c '^root: sh -s -- link$' "$ws/log")" = $((links + 1)) ] &&
    [ "$(grep -c '^root: sh -s -- wait enp5s0$' "$ws/log")" = 1 ] ||
    error "ws client macos wizard with WS_CABLE_PORT exited $status, or wrote no host_vars, ran no roles, or" \
        "read link other than once and wait other than once"
mv -f "$cable/bin/ip" "$ws/bin/ip"
# On macOS, sudo, which the stages' checks run, takes what the terminal holds. So the lines go in
# once the question shows, or after 120 s. script's input ends 5 s after them, once it wrote them.
# shellcheck disable=SC2094 # the loop reads the question that script writes to tty, by design
{
    for _ in $(seq 120); do
        ! grep -qF '(WS_CABLE_SERVICE) [' "$RUNNER_TEMP/tty" 2>/dev/null || break
        sleep 1
    done
    printf '%s\n' 'ws stand-in absent' ''
    sleep 5
} | script -q /dev/null "$wsbin" --plain client macos wizard >"$RUNNER_TEMP/tty" 2>&1
status=$?
tr -d '\r' <"$RUNNER_TEMP/tty"
grep -qF "(WS_CABLE_SERVICE) [ws stand-in adapter, or none]: " "$RUNNER_TEMP/tty" &&
    grep -qF "ws client macos wizard: Cable: 'ws stand-in absent' is none of those" "$RUNNER_TEMP/tty" &&
    [ "$status" = 0 ] && [ "$(conf_print "$conf" | tail -n 1)" = 'WS_CABLE_SERVICE="ws stand-in adapter"' ] &&
    conf_print "$conf" | grep -qx WS_CABLE_PORT=enp5s0 &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 1' ] ||
    error "ws client macos wizard at a terminal exited $status, set the adapter or pinged other than once, or" \
        "kept no port"
links=$(grep -c '^root: sh -s -- link$' "$ws/log")
wizard run
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] &&
    [ "$(grep -c '^root: sh -s -- link$' "$ws/log")" = "$links" ] ||
    error "the next run exited $status, or read link"
# ws's install again, undocked: host_vars' port and the netplan file go, and enp5s0 loses carrier.
kept=$(conf_print "$conf")
runs=$(grep -c '' "$ws/server.log")
unname
sudo rm -f "$netplan"
echo 0 | sudo tee "$net/enp5s0/carrier" >/dev/null
wizard --check
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: after ws's run" && unnamed &&
    [ "$status" = 0 ] || error "ws client macos wizard --check without host_vars exited $status"
wizard run
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: enp5s0 goes in /etc/ws/host_vars\.yml on ws \
again, and ws server ubuntu apply --only network,wireguard runs there in a unit of its own" &&
    sudo grep -qx 'network_direct_link_interface: "enp5s0"' "$vars" && [ "$status" = 0 ] &&
    [ "$(grep -c '' "$ws/server.log")" = $((runs + 2)) ] && [ "$(conf_print "$conf")" = "$kept" ] ||
    error "ws client macos wizard without host_vars exited $status, named no enp5s0, or changed mac.conf"
sudo rm -f "$netplan"
wizard run
# root alone reads the netplan file's tree, which the files script's peer makes with umask 077.
sudo test -e "$netplan" && [ "$status" = 0 ] &&
    [ "$(grep -c '' "$ws/server.log")" = $((runs + 4)) ] ||
    error "ws client macos wizard without the netplan file exited $status, ran the roles other than once, or" \
        "wrote no netplan file"
echo 1 | sudo tee "$net/enp5s0/carrier" >/dev/null
conf_write "$conf" <"$RUNNER_TEMP/cable.conf"
touch "$cable/silent"
wizard run "WS_CABLE_SERVICE=ws stand-in adapter"
printf '%s\n' "$out" | grep -qx "ws client macos wizard: Cable: ping -c 1 -t 3 10\.77\.0\.1, ws's end of the \
cable, got no answer: mac\.conf's WS_CABLE_SERVICE must name the service of the adapter that the \
cable plugs into" && [ "$status" = 1 ] &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 2' ] ||
    error "ws client macos wizard with no answer from ping exited $status, or pinged other than once"
rm "$cable/silent"
conf_write "$conf" <"$RUNNER_TEMP/cable.conf"
wizard run WS_CABLE_SERVICE=
[ "$status" = 0 ] && [ "$(conf_print "$conf" | tail -n 1)" = WS_CABLE_SERVICE= ] &&
    [ "$(logged setmanual.log) $(logged ping.log)" = '1 2' ] ||
    error "ws client macos wizard with WS_CABLE_SERVICE= exited $status, or wrote another line, or acted"
wizard run
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "the run after none exited $status"

conf_write "$conf" <"$RUNNER_TEMP/cable.conf"
rm -rf /opt/homebrew/bin/networksetup /opt/homebrew/bin/ping "$RUNNER_TEMP/cable.conf" \
    "$RUNNER_TEMP/tty" "$ws/uplink"
unname
sudo rm -rf "$cable" "$ws/root/sys" "$ws/addr" "${netplan%/*}"
wizard run
[ "$status" = 0 ] && [ "$(tail -n 1 "$ws/server.log")" = '' ] ||
    error "the run after host_vars lost the port exited $status, or ran no ws server ubuntu apply"
wizard run
[ "$status" = 0 ] && [ -z "${out#"$known"}" ] || error "ws client macos wizard after it exited $status"
exit "$fail"
