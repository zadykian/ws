#!/bin/sh
# ws's programs that the wizard's remote script runs, as stand-ins on GitHub's macOS runner, each by
# the name it is called by. standin.sh links the names in WS_STANDIN/bin, which ws.sh puts first on
# PATH, and claude where the remote script looks for it. Their state is in WS_STANDIN, which ws's
# sshd sets: units/NAME.on is there while the unit NAME runs, wg0 holds wg0's key, sb turns Secure
# Boot on, nct6687 builds the fans' driver, and gh and claude log each in.

set -u

s=$WS_STANDIN
case ${0##*/} in
systemctl)
    case $1 in
    list-units)
        for on in "$s"/units/*.on; do
            [ ! -e "$on" ] || echo "$(basename "$on" .on).service loaded active running stand-in"
        done
        ;;
    *) exit 2 ;;
    esac
    ;;
wg) cat "$s/wg0" 2>/dev/null ;;
gh) [ "${1-} ${2-}" = 'auth status' ] && [ -f "$s/gh" ] ;;
claude) [ -f "$s/claude" ] ;;
mokutil) if [ -f "$s/sb" ]; then echo 'SecureBoot enabled'; else echo 'SecureBoot disabled'; fi ;;
modinfo) [ -f "$s/nct6687" ] ;;
*) exit 2 ;;
esac
