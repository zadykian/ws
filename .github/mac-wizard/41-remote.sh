#!/bin/sh
# The wizard's remote script on the stand-in ws, on the Mac that 40-again.sh leaves, sent to root
# over ssh as the wizard sends it. status must print the stamps and each state of the stand-ins.
# First come wg0's key, a Mac's key it lacks, Secure Boot that refuses the fans' driver, and no
# login of claude's. Then its login, with no signing key in root's git config, and Secure Boot
# off. A full run must run ws server apply without arguments in a unit of its own, with root's HOME and
# a UTF-8 locale, print its log and exit 0. It must stamp ws's version, 0 and on in run, and the unit
# and 0 in last. A tagged run after it must start its own unit, exit with its status, and leave run
# as it was. A run during another's unit must follow that unit and start none. A full run whose unit
# is cut off must exit 1 and remove the run stamp that the one before left. The next must start a
# unit of its own, as must one after outbound, which stamps off in run, as /etc/ws/host_vars.yml,
# a file of mode 0600 in a directory of 0755, names security_outbound: false. One with an argument, as
# --diff, must leave that stamp. The files script's arch must print amd64 or arm64. Its stage must
# stop without a tag, and print the path for a command of another repository's release. Its install
# of the command sent there must stop where the SHA-256 is another, and else leave the command at
# bin in state, root's alone, mode 700, which must run as root over ssh, its stdin empty. Its stamp
# must follow, which status must list. Its install again must remove the stamp, and with nothing
# sent stop. Its have must print the command's path, but not another tag's. Its words must print git's hash of root's word list, or
# - without one, and the list for that hash alone. Its link must list ws's wired ports but the
# uplink and a bridge's member, with their carrier and IPv4 addresses. link PORT must name one in
# ws's /etc/ws/host_vars.yml once, and ws server apply --only network,wireguard then give it 10.77.0.1, and
# status's cable name it. The remote script's outbound must add security_outbound: false there once,
# which status must then read as off. The last line counts, as in YAML: Off with a comment by hand
# must read as off, and stay, and true after it as on. Through a link, outbound must stop, but where
# the linked file holds the line. The stand-ins' state goes back.

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

# wsv prints the version that the stand-in ws prints, or - where it prints none.
wsv() {
    cat "$ws/ws.version" 2>/dev/null || echo -
}

# states WANT...: status must print the stamps, no unit that runs, ws's version, then the lines WANT.
states() {
    remote status
    want=$(printf '%s\n' "run $(stamp run)" "last $(stamp last)" 'active -' "ws $(wsv)" "$@")
    [ "$status" = 0 ] && [ "$out" = "$want" ] ||
        error "status exited $status, printing other than '$want'"
}

# The Tunnel stage's wg0 and mac.pub go aside, for the states, and come back after.
mv "$ws/wg0" "$RUNNER_TEMP/wg0"
sudo mv "$peer" "$RUNNER_TEMP/mac.pub"
key=$(head -c 32 /dev/urandom | base64)
mac=$(head -c 32 /dev/urandom | base64)
echo "$key" >"$ws/wg0"
touch "$ws/sb" "$ws/nct6687"
echo "$mac" | sudo tee "$peer" >/dev/null
rm "$ws/claude"
states "mac.pub $mac" "wg0 $key" "signing $(cut -d ' ' -f 1-2 "$ws/signing.pub")" \
    "hostname $(hostname)" 'claude login' 'fans refused' 'cable -' 'peer -' \
    'hook -' 'outbound on'
touch "$ws/claude"
rm "$ws/sb" "$ws/wg0"
sudo rm "$peer"
sudo git config -f "$ws/root/root/.gitconfig" --unset user.signingkey
states 'mac.pub -' 'wg0 -' 'signing none' "hostname $(hostname)" 'claude ok' 'fans ok' 'cable -' \
    'peer -' 'hook -' 'outbound on'
rm "$ws/nct6687"
sudo git config -f "$ws/root/root/.gitconfig" user.signingkey \
    "$ws/root/root/.ssh/id_ed25519_signing.pub"
mv "$RUNNER_TEMP/wg0" "$ws/wg0"
sudo mv "$RUNNER_TEMP/mac.pub" "$peer"

# units prints how many units systemd-run started, and unit the last one's name.
units() {
    wc -l <"$ws/run.log"
}
unit() {
    sed -n 's/.* --unit=\([^ ]*\) .*/\1/p' "$ws/run.log" | tail -n 1
}

# The stamp's version is the stand-in ws's, which the wizard's install left in ws.version.
sha=$(wsv)
n=$(units)
remote run
full=$(unit)
printf '%s\n' "$out" | grep -qx 'PLAY RECAP' && [ "$status" = 0 ] &&
    [ "$(units)" -eq $((n + 1)) ] && [ "$(stamp run) / $(stamp last)" = "$sha 0 on / $full 0" ] &&
    [ "$(tail -n 1 "$ws/server.log")" = '' ] || error "a full run exited $status"
tail -n 1 "$ws/run.log" | grep -q -- '--collect .* --setenv=LC_ALL=C.UTF-8 --setenv=HOME=/root ' ||
    error "a full run's unit lacked root's HOME or a UTF-8 locale"
echo 3 >"$ws/server.status"
remote run --only devtools
[ "$status" = 3 ] && [ "$(units)" -eq $((n + 2)) ] && [ "$(unit)" != "$full" ] &&
    [ "$(stamp run) / $(stamp last)" = "$sha 0 on / $(unit) 3" ] &&
    [ "$(tail -n 1 "$ws/server.log")" = '--only devtools' ] ||
    error "a tagged run exited $status"
rm "$ws/server.status"
echo 4 >"$ws/server.wait"
ssh -o BatchMode=yes ws-ssh sh -s -- run <"$remote_sh" >"$RUNNER_TEMP/first" 2>&1 &
first=$!
tries=0
until ls "$ws"/units/*.on >/dev/null 2>&1 || [ "$tries" = 20 ]; do
    sleep 1
    tries=$((tries + 1))
done
n=$(units)
remote run --only other
wait "$first"
first=$?
printf '%s\n' "$out" | grep -q '^wizard-remote\.sh: following ws-wizard-' &&
    ! grep -q other "$ws/server.log" && [ "$first" = 0 ] && [ "$status" = 0 ] &&
    [ "$(units)" -eq "$n" ] || error "a run during another's unit exited $status, or started one"
rm "$ws/server.wait" "$RUNNER_TEMP/first"
touch "$ws/server.cut"
remote run
printf '%s\n' "$out" | grep -q ' ended before its run wrote its status' && [ "$status" = 1 ] &&
    [ "$(stamp run)" = - ] || error "a full run cut off exited $status, leaving run '$(stamp run)'"
rm "$ws/server.cut"
n=$(units)
remote run
[ "$status" = 0 ] && [ "$(units)" -eq $((n + 1)) ] && [ "$(stamp run)" = "$sha 0 on" ] ||
    error "the run after it exited $status, with run '$(stamp run)'"
sudo rm "$stamps/run"
remote outbound
remote run
[ "$status" = 0 ] && [ "$(units)" -eq $((n + 2)) ] && [ "$(stamp run)" = "$sha 0 off" ] ||
    error "a full run after outbound exited $status, with run '$(stamp run)'"
host_vars="$ws/root"/etc/ws
sudo grep -qx 'security_outbound: false' "$host_vars/host_vars.yml" ||
    error "outbound named no security_outbound: false in host_vars"
[ "$(sudo stat -f %Lp "$host_vars")" = 755 ] &&
    [ "$(sudo stat -f %Lp "$host_vars/host_vars.yml")" = 600 ] ||
    error "outbound made /etc/ws or its host_vars.yml with another mode"
sudo rm -r "$host_vars"
remote run --diff
[ "$status" = 0 ] && [ "$(stamp run)" = "$sha 0 off" ] || error "a run with --diff stamped run"

# files OP [ARG...]: the files script's OP, as remote runs the remote script's.
files() {
    out=$(ssh -o BatchMode=yes ws-ssh sh -s -- "$@" <"${remote_sh%/*}/wizard-remote-files.sh" 2>&1)
    status=$?
    printf '%s\n' "$out"
}

# send REPO: the files script's stage op names a path on ws for the command of example/REPO@v1.0.0,
# which goes there as the wizard sends it. The command is hook-standin.sh, as the Mac's stand-in gh
# downloads it. It sets sum to the command's SHA-256.
send() {
    src=$RUNNER_TEMP/$1
    cp "$(cd "$(dirname "$0")" && pwd)/hook-standin.sh" "$src"
    sum=$(shasum -a 256 <"$src" | cut -d ' ' -f 1)
    files stage "example/$1@v1.0.0"
    [ "$status" = 0 ] && [ "$out" = "$stamps/hook-$1.bin" ] ||
        error "the files script's stage of example/$1 exited $status, printing '$out'"
    ssh -o BatchMode=yes ws-ssh "umask 077 && cat >$out" <"$src"
}

bin=$stamps/bin/hook-v1.0.0
files arch
case $out in amd64 | arm64) ;; *) error "the files script's arch printed '$out'" ;; esac
files stage example/hook
printf '%s\n' "$out" | grep -qF "is not a release" && [ "$status" = 1 ] ||
    error "the files script's stage with no tag exited $status"
send hook
files install example/hook@v1.0.0 "$(printf %s bad | shasum -a 256 | cut -d ' ' -f 1)"
[ "$status" = 1 ] && printf '%s\n' "$out" | grep -qF 'not the one the Mac checked' &&
    [ ! -e "$stamps/hook-hook.bin" ] && [ ! -e "$bin" ] ||
    error "the files script's install of another sum exited $status, or installed"
send hook
files install example/hook@v1.0.0 "$sum"
[ "$status" = 0 ] && [ "$out" = "wizard-remote-files.sh: installed example/hook@v1.0.0 at $bin" ] &&
    [ "$(sudo stat -f '%Lp %Su' "$bin")" = '700 root' ] &&
    [ "$(sudo stat -f '%Lp' "${bin%/*}")" = 700 ] && [ ! -e "$stamps/hook-hook.bin" ] ||
    error "the files script's install exited $status, or left the command other than root's alone"
ssh -o BatchMode=yes ws-ssh /var/lib/ws-wizard/bin/hook-v1.0.0 apply </dev/null &&
    grep -qx 'root: /var/lib/ws-wizard/bin/hook-v1.0.0 apply' "$ws/log" &&
    [ "$(cat "$ws/hook.args")" = apply ] &&
    [ "$(cat "$ws/apply.log")" = "$(shasum -a 256 </dev/null | cut -d ' ' -f 1)" ] ||
    error "the command ran other than as root, once, with its stdin empty"
files stamp example/hook@v1.0.0
[ "$status" = 0 ] && [ "$(stamp hook-hook)" = "$sum 0 example/hook@v1.0.0" ] ||
    error "the files script's stamp exited $status, leaving '$(stamp hook-hook)'"
files have example/hook@v1.0.0
[ "$out" = "$bin" ] || error "the files script's have printed '$out'"
remote status
printf '%s\n' "$out" | grep -qx 'hook example/hook@v1.0.0' || error "status listed no example/hook"
send hook
files install example/hook@v1.0.0 "$sum"
[ "$status" = 0 ] && [ "$(stamp hook-hook)" = - ] ||
    error "the files script's install again exited $status, or left the stamp"
files install example/hook@v1.0.0 "$sum"
printf '%s\n' "$out" | grep -qF 'the Mac sent none' && [ "$status" = 1 ] ||
    error "the files script's install with nothing sent exited $status"
files have example/hook@v1.0.1
[ "$status" = 0 ] && [ -z "$out" ] || error "the files script's have of another tag exited $status, printing '$out'"
rm -f "$RUNNER_TEMP/hook"
list=$ws/root/root/.config/ws/forbidden-words
files words
[ "$status" = 0 ] && [ "$out" = - ] || error "the files script's words with no list exited $status"
sudo mkdir -p "${list%/*}"
printf '%s\n' '# stand-in words' example-word 'stand-in phrase' | sudo tee "$list" >/dev/null
sum=$(git hash-object --stdin <"$list")
files words
[ "$status" = 0 ] && [ "$out" = "$sum" ] || error "the files script's words exited $status"
files words "$sum"
[ "$status" = 0 ] && [ "$out" = "$(cat "$list")" ] || error "its words $sum exited $status"
files words 0
[ "$status" = 1 ] || error "its words with another hash exited $status"
# ws's ports, in a sysfs of the stand-in's, root's: the uplink, a port with carrier, one without and
# with an address, one that is down, and a wireless one. A member of the bridge br0 has carrier.
net=$ws/root/sys/class/net
vars="$ws/root"/etc/ws/host_vars.yml
for each in enp4s0:1 enp5s0:1 enp6s0:0 enp7s0 wlan0:1 enp8s0:1; do
    sudo mkdir -p "$net/${each%:*}/device"
    echo 1 | sudo tee "$net/${each%:*}/type" >/dev/null
    [ "$each" = "${each%:*}" ] || echo "${each#*:}" | sudo tee "$net/${each%:*}/carrier" >/dev/null
done
sudo mkdir "$net/wlan0/wireless" "$net/br0"
sudo ln -s ../br0 "$net/enp8s0/master"
echo enp4s0 >"$ws/uplink"
mkdir -p "$ws/addr"
echo 198.51.100.7/24 >"$ws/addr/enp6s0"
files link
[ "$status" = 0 ] && [ "$out" = "$(printf '%s\n' 'enp5s0 1 -' 'enp6s0 0 198.51.100.7/24' \
    'enp7s0 - -')" ] || error "the files script's link exited $status"
files link enp4s0
[ "$status" = 1 ] && [ ! -e "$vars" ] || error "its link of the uplink exited $status"
files link enp5s0
printf '%s\n' --- 'network_direct_link_interface: "enp5s0"' | sudo cmp -s - "$vars" &&
    [ "$status" = 0 ] && [ "$out" = "wizard-remote-files.sh: wrote $vars" ] ||
    error "its link enp5s0 exited $status"
files link enp5s0
[ "$status" = 0 ] && [ -z "$out" ] || error "its link enp5s0 again exited $status"
remote status
printf '%s\n' "$out" | grep -qx 'cable -' || error "status named a cable before the roles' run"
remote run --only network,wireguard
files link
printf '%s\n' "$out" | grep -qx 'enp5s0 1 10\.77\.0\.1/30' &&
    [ "$(tail -n 1 "$ws/server.log")" = '--only network,wireguard' ] ||
    error "ws server apply --only network,wireguard gave enp5s0 no 10.77.0.1/30"
remote status
printf '%s\n' "$out" | grep -qx 'cable enp5s0' || error "status named no cable enp5s0"
remote outbound
printf '%s\n' --- 'network_direct_link_interface: "enp5s0"' 'security_outbound: false' |
    sudo cmp -s - "$vars" && [ "$status" = 0 ] && [ "$out" = "wizard-remote.sh: wrote $vars" ] ||
    error "the remote script's outbound exited $status"
remote outbound
[ "$status" = 0 ] && [ -z "$out" ] || error "its outbound again exited $status"
remote status
printf '%s\n' "$out" | grep -qx 'outbound off' || error "status read no outbound off"
printf '%s\n' --- 'security_outbound: true' 'security_outbound: Off  # by hand' |
    sudo tee "$vars" >/dev/null
remote outbound
[ "$status" = 0 ] && [ -z "$out" ] || error "its outbound with Off by hand exited $status"
remote status
printf '%s\n' "$out" | grep -qx 'outbound off' || error "status read Off by hand as on"
echo 'security_outbound: true' | sudo tee -a "$vars" >/dev/null
sudo mv "$vars" "$vars.real" && sudo ln -s host_vars.yml.real "$vars"
remote status
printf '%s\n' "$out" | grep -qx 'outbound on' || error "status read true after Off as off"
remote outbound
printf '%s\n' "$out" | grep -qF "$vars is a link to host_vars.yml.real" && [ "$status" = 1 ] ||
    error "its outbound through a link exited $status"
echo 'security_outbound: false' | sudo tee -a "$vars.real" >/dev/null
remote outbound
[ "$status" = 0 ] && [ -z "$out" ] || error "its outbound through a link to the line exited $status"
sudo rm -rf "$stamps/bin" "$ws/root/root/.config" "$ws/root/sys" "$ws/addr" "${vars%/*}" \
    "$ws/root/etc/netplan"
rm -f "$ws/apply.log" "$ws/hook.args" "$ws/uplink"
exit "$fail"
