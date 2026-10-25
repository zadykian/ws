#!/bin/sh
# ws's programs that the wizard's remote script runs, as stand-ins on GitHub's macOS runner, each by
# the name it is called by. standin.sh links the names in WS_STANDIN/bin, which ws.sh puts first on
# PATH, claude where the remote script looks for it, and bootstrap.sh in the stand-in clone. Their
# state is in WS_STANDIN, which ws's sshd sets:
#
# - units/NAME is a unit's log, as the journal holds it, and units/NAME.on is there while it runs;
# - run.log holds systemd-run's arguments, and bootstrap.log bootstrap.sh's, a line each;
# - bootstrap.wait holds the seconds that bootstrap.sh runs, and bootstrap.status its status;
# - with bootstrap.cut, bootstrap.sh kills the unit's job, as a stop or a reboot would;
# - wg0 holds wg0's key, sb turns Secure Boot on, nct6687 builds the fans' driver, and gh and
#   claude log each in;
# - gh.keys holds the signing keys of gh's account, which bootstrap.sh --tags devtools adds ws's to,
#   and gh.nosign gives their list a 404, as a login without the write:ssh_signing_key scope gets.

set -u

s=$WS_STANDIN
case ${0##*/} in
systemd-run)
    echo "$*" >>"$s/run.log"
    while [ "${1#--}" != "$1" ]; do
        case $1 in --unit=*) unit=${1#--unit=} ;; esac
        shift
    done
    touch "$s/units/$unit.on"
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
bootstrap.sh)
    echo "$*" >>"$s/bootstrap.log"
    echo "bootstrap.sh's stand-in, with '$*'"
    sleep "$(cat "$s/bootstrap.wait" 2>/dev/null || echo 0)"
    if [ -f "$s/bootstrap.cut" ]; then
        kill -9 "$PPID"
        exit 1
    fi
    # The devtools role adds ws's signing key to the account, where gh's login has the scope.
    [ "$*" != '--tags devtools' ] || [ ! -f "$s/gh" ] || [ -f "$s/gh.nosign" ] ||
        cut -d ' ' -f 1-2 "$s/signing.pub" >>"$s/gh.keys"
    echo 'PLAY RECAP'
    exit "$(cat "$s/bootstrap.status" 2>/dev/null || echo 0)"
    ;;
wg) cat "$s/wg0" 2>/dev/null ;;
gh)
    # The remote script's calls, then the device flow's login and refresh, which the wizard runs in
    # ssh -t.
    case $* in
    'auth status -h github.com') [ -f "$s/gh" ] ;;
    'api --paginate user/ssh_signing_keys --jq .[].key')
        [ -f "$s/gh" ] || exit 4
        [ ! -f "$s/gh.nosign" ] || { echo 'gh: Not Found (HTTP 404)' >&2 && exit 1; }
        cat "$s/gh.keys"
        ;;
    'auth login -h github.com -p https -w -s write:ssh_signing_key')
        echo '! First copy your one-time code: 0000-0000'
        touch "$s/gh" && rm -f "$s/gh.nosign"
        ;;
    'auth refresh -h github.com -s write:ssh_signing_key') rm -f "$s/gh.nosign" ;;
    *) exit 2 ;;
    esac
    ;;
claude) [ -f "$s/claude" ] ;;
mokutil) if [ -f "$s/sb" ]; then echo 'SecureBoot enabled'; else echo 'SecureBoot disabled'; fi ;;
modinfo) [ -f "$s/nct6687" ] ;;
*) exit 2 ;;
esac
