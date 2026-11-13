#!/bin/sh
# ws's programs that the wizard's remote script runs, as stand-ins on GitHub's macOS runner, each by
# the name the script calls it by. standin.sh links the names in WS_STANDIN/bin, which ws.sh puts
# first on PATH, claude where the remote script looks for it and ws at /usr/local/bin/ws. Their
# state is in WS_STANDIN, which ws's sshd sets:
#
# - sudo.open is there while sudo asks no password, and sudo.password holds the one that it asks
#   otherwise. sudo.log holds the command lines that sudo ran for ws.sh;
# - ws.version is the version that ws prints, and install.log holds a line for each curl, its URL.
#   curl writes the release's install.sh, which puts its WS_VERSION in ws.version;
# - units/NAME is a unit's log, as the journal holds it, and units/NAME.on is there while it runs;
# - run.log holds systemd-run's arguments, and server.log the arguments of ws server ubuntu apply
#   past its --json, which it requires, a line each. values.log holds those of ws server ubuntu
#   values set, a line each, each argument in brackets;
# - secrets.log holds the arguments of ws server ubuntu secrets, a line each, secrets.in its last
#   stdin, and keys what its keys prints;
# - creds/UNIT holds a unit's credentials, and seen/ the identity and the secrets file that
#   ws server ubuntu apply --secrets found, identity and secrets.age;
# - server.wait holds the seconds that ws server ubuntu apply runs. server.status holds its status,
#   and that of ws server ubuntu state --json, which prints a line with its version, HOME and
#   locale;
# - with server.cut, ws server ubuntu apply kills the unit's job, as a stop or a reboot would;
# - wg0 holds wg0's key, sb turns Secure Boot on, nct6687 builds the fans' driver, and claude logs
#   in;
# - ws.key is wg0's key, which ws server ubuntu apply --only wireguard makes, and utun names wg0,
#   root's;
# - uplink names the port of the default route. addr/PORT holds a port's IPv4 addresses, which ws
#   server ubuntu apply --only network,wireguard gives the port that host_vars names, with carrier.
#   That run writes the port in root's netplan file, as the network role does, or removes the file.

set -u

s=$WS_STANDIN

# wg0 sets up ws's wg0 as the wireguard role does. It sets its key, made once, whose public key
# goes in wg0, and the Mac's key in mac.pub as its peer. wg0 is a wireguard-go of the runner's,
# beside the Mac's, on UDP 51820, as mac.yml's stand-in ws. It has no address in the tunnel.
wg0() {
    bin=/opt/homebrew/bin
    [ -f "$s/ws.key" ] || (umask 077 && "$bin/wg" genkey >"$s/ws.key") || return 1
    "$bin/wg" pubkey <"$s/ws.key" >"$s/wg0" || return 1
    if [ ! -f "$s/utun" ] || ! "$bin/wg" show "$(cat "$s/utun")" >/dev/null 2>&1; then
        mkdir -p /var/run/wireguard
        WG_TUN_NAME_FILE=$s/utun "$bin/wireguard-go" utun || return 1
    fi
    "$bin/wg" set "$(cat "$s/utun")" private-key "$s/ws.key" listen-port 51820 \
        peer "$(cat "$s/root/etc/wireguard/mac.pub")" allowed-ips 10.99.0.2/32 || return 1
    ifconfig "$(cat "$s/utun")" up
}
# direct names the port that ws's host_vars names in the netplan file, as the network role does,
# or removes the file where host_vars names none. It gives the port ws's end of the cable,
# 10.77.0.1/30, once networkd sees carrier, which the cases' sysfs shows.
direct() {
    port=$(sed -n 's/^network_direct_link_interface: "\(.*\)"$/\1/p' \
        "$s/root"/etc/ws/host_vars.yml 2>/dev/null)
    netplan=$s/root/etc/netplan/60-direct-link.yaml
    rm -f "$netplan"
    [ -z "$port" ] || { mkdir -p "${netplan%/*}" && echo "    \"$port\":" >"$netplan"; } || return 1
    [ -n "$port" ] && [ "$(cat "$s/root/sys/class/net/$port/carrier" 2>/dev/null)" = 1 ] ||
        return 0
    mkdir -p "$s/addr"
    grep -qsx 10.77.0.1/30 "$s/addr/$port" || echo 10.77.0.1/30 >>"$s/addr/$port"
}
case ${0##*/} in
sudo)
    # The forms that the wizard sends. -n takes no password. -k -S -p '' takes it as the first
    # line of stdin, a byte at a time, so that the script behind it stays in the pipe. The command
    # runs as the runner's root, as real sudo does, which sets SUDO_USER.
    case $1 in
    -n)
        shift
        [ -f "$s/sudo.open" ] || { echo 'sudo: interactive authentication is required' >&2 && exit 1; }
        ;;
    -k)
        [ "$1 $2 $3 $4" = "-k -S -p " ] || exit 2
        shift 4
        IFS= read -r password
        [ "$password" = "$(cat "$s/sudo.password")" ] ||
            { echo 'sudo: Authentication failed, try again.' >&2 && exit 1; }
        ;;
    *) exit 2 ;;
    esac
    exec /usr/bin/sudo -n env PATH="$PATH" WS_STANDIN="$s" "$@"
    ;;
systemd-run)
    echo "$*" >>"$s/run.log"
    cred=
    while [ "${1#-}" != "$1" ]; do
        case $1 in
        --unit=*) unit=${1#--unit=} ;;
        -p)
            shift
            case $1 in LoadCredentialEncrypted=*) cred=${1#LoadCredentialEncrypted=} ;; esac
            ;;
        esac
        shift
    done
    touch "$s/units/$unit.on"
    # A credential is copied from the credstore to a directory of the unit's before systemd-run
    # returns, as Type=exec has it. Only the unit's job sees that directory, as
    # CREDENTIALS_DIRECTORY.
    if [ -n "$cred" ]; then
        store="$s/root"/etc/credstore.encrypted
        mkdir -p "$s/creds/$unit" && cp "$store/$cred" "$s/creds/$unit/$cred" || exit 1
        CREDENTIALS_DIRECTORY=$s/creds/$unit
        export CREDENTIALS_DIRECTORY
    fi
    # The unit goes on where ssh drops, as systemd's does.
    (
        trap '' HUP
        "$@" >>"$s/units/$unit" 2>&1
        rm -f "$s/units/$unit.on"
    ) </dev/null >/dev/null 2>&1 &
    ;;
systemctl)
    case $1 in
    list-units)
        for on in "$s"/units/*.on; do
            [ ! -e "$on" ] || echo "$(basename "$on" .on).service loaded active running stand-in"
        done
        ;;
    show)
        eval "unit=\${$#}"
        if [ -e "$s/units/$unit.on" ]; then echo active; else echo inactive; fi
        ;;
    *) exit 2 ;;
    esac
    ;;
journalctl)
    # The lines of the unit's log after those that the cursor's file counts.
    while [ $# -gt 0 ]; do
        case $1 in
        -u) unit=$2 && shift ;;
        --cursor-file=*) cursor=${1#--cursor-file=} ;;
        esac
        shift
    done
    [ -f "$s/units/$unit" ] || exit 0
    at=$(cat "$cursor" 2>/dev/null || echo 0)
    lines=$(wc -l <"$s/units/$unit")
    [ "$lines" -gt "$at" ] || exit 0
    sed -n "$((at + 1)),${lines}p" "$s/units/$unit"
    echo "$lines" >"$cursor"
    ;;
curl)
    # The release's install.sh, written to the file that -o names: the URL is its last argument.
    while [ $# -gt 1 ]; do
        [ "$1" != -o ] || out=$2
        shift
    done
    echo "$1" >>"$s/install.log"
    # shellcheck disable=SC2016 # the sh that runs install.sh expands them
    printf '%s\n' '#!/bin/sh' 'echo "$WS_VERSION" >"$WS_STANDIN/ws.version"' >"$out"
    ;;
ws)
    case $1 in
    version) cat "$s/ws.version" 2>/dev/null ;;
    server)
        [ "${2-}" = ubuntu ] || exit 2
        shift
        if [ "$2" = secrets ]; then
            shift 2
            echo "$*" >>"$s/secrets.log"
            cat >"$s/secrets.in"
            [ "$1" != keys ] || cat "$s/keys"
            exit
        fi
        if [ "$2 ${3-}" = 'values set' ]; then
            shift 3
            printf '[%s]' "$@" >>"$s/values.log"
            echo >>"$s/values.log"
            exit 0
        fi
        if [ "$2 ${3-}" = 'state --json' ]; then
            echo "report of $(cat "$s/ws.version") with $HOME and $LC_ALL"
            exit "$(cat "$s/server.status" 2>/dev/null || echo 0)"
        fi
        [ "$2 ${3-}" = 'apply --json' ] || exit 2
        shift 3
        echo "$*" >>"$s/server.log"
        echo "ws server ubuntu apply's stand-in, with '$*'"
        echo '{"event":"done","step":"stand-in","index":0}'
        # With --secrets, the credential must be there, and the file: they are copied to seen/.
        case $* in
        *--secrets*)
            mkdir -p "$s/seen"
            for file; do :; done
            cp "$CREDENTIALS_DIRECTORY/ws-secrets" "$s/seen/identity" &&
                cp "$file" "$s/seen/secrets.age" || exit 1
            ;;
        esac
        sleep "$(cat "$s/server.wait" 2>/dev/null || echo 0)"
        if [ -f "$s/server.cut" ]; then
            kill -9 "$PPID"
            exit 1
        fi
        [ "$*" != '--only wireguard' ] || wg0 || exit 1
        [ "$*" != '--only network,wireguard' ] || direct
        echo 'PLAY RECAP'
        exit "$(cat "$s/server.status" 2>/dev/null || echo 0)"
        ;;
    *) exit 2 ;;
    esac
    ;;
wg)
    # wg0's key, and with show wg0 peers the peers of wg0's wireguard-go.
    [ "$*" = 'show wg0 peers' ] || exec cat "$s/wg0" 2>/dev/null
    /opt/homebrew/bin/wg show "$(cat "$s/utun")" peers
    ;;
claude)
    case $* in
    'auth status') [ -f "$s/claude" ] ;;
    'auth login') [ -t 0 ] && touch "$s/claude" ;;
    *) exit 2 ;;
    esac
    ;;
ip)
    # The default routes, by the port that uplink names, and the IPv4 addresses in addr/PORT.
    case $* in
    *'route show default table all')
        [ ! -f "$s/uplink" ] || echo "default via 192.0.2.1 dev $(cat "$s/uplink") metric 100"
        ;;
    "-o -4 address show dev ${6-}")
        [ ! -f "$s/addr/$6" ] || sed "s|.*|2: $6    inet & scope global $6|" "$s/addr/$6"
        ;;
    *) exit 2 ;;
    esac
    ;;
getent)
    # An account's line: root's home is the stand-in's, as another user's is below its home.
    case $2 in
    root) echo "root:x:0:0::$s/root/root:/bin/sh" ;;
    *) echo "$2:x:501:20::$s/root/home/$2:/bin/sh" ;;
    esac
    ;;
runuser)
    # runuser -u NAME -- COMMAND, by the runner's sudo, which needs no password.
    [ "$1" = -u ] && [ "$3" = -- ] || exit 2
    user=$2
    shift 3
    exec /usr/bin/sudo -n -u "$user" env PATH="$PATH" WS_STANDIN="$s" "$@"
    ;;
mokutil) if [ -f "$s/sb" ]; then echo 'SecureBoot enabled'; else echo 'SecureBoot disabled'; fi ;;
modinfo) [ -f "$s/nct6687" ] ;;
*) exit 2 ;;
esac
