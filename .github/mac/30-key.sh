#!/bin/bash
# Without ws's key, setup.sh makes the Mac's key and installs no daemon.
# The key it prints, and the command that makes it ws's peer, must hold the Mac's public key,
# which the stand-in ws below takes as its peer. Root's daemon runs the three programs and
# sources the two files, so they must be root's alone. Stand-ins for the apps of Gateway and
# Ghostty, its font and cld spare this run and later ones their downloads (mac-standins.sh).

set -eo pipefail

sh .github/mac-standins.sh
sh mac/setup.sh | tee "$RUNNER_TEMP/first.out" ||
    {
        echo "::error::setup.sh without ws's key failed"
        exit 1
    }
fail=0
wg=$(brew --prefix wireguard-tools)/bin/wg
mac=$(sudo sh -c '"$1" pubkey <"$2"' sh "$wg" /etc/ws-tunnel/private.key) ||
    {
        echo "::error::cannot read the Mac's public key"
        exit 1
    }
key=$(sed -n "s/^The Mac's WireGuard public key is //p" "$RUNNER_TEMP/first.out")
if [[ ! $key =~ ^[A-Za-z0-9+/]{43}=$ ]] || [ "$key" != "$mac" ]; then
    echo "::error::setup.sh printed '$key' as the Mac's public key, not '$mac'"
    fail=1
fi
give="    ssh ws-ssh \"printf '%s\\n' '$mac' | tee /etc/wireguard/mac.pub >/dev/null\""
grep -qxF "$give" "$RUNNER_TEMP/first.out" ||
    {
        echo "::error::setup.sh printed no command that gives ws '$mac'"
        fail=1
    }
printf '%s\n' "$key" >"$RUNNER_TEMP/mac.pub"
printf '%s\n' "$mac" | cmp -s - ~/.config/ws/tunnel.pub ||
    {
        echo "::error::~/.config/ws/tunnel.pub holds no '$mac'"
        fail=1
    }
[ "$(sudo stat -f '%Su:%Sg %Lp' /etc/ws-tunnel/private.key)" = "root:wheel 600" ] ||
    {
        echo "::error::/etc/ws-tunnel/private.key is not root's, 600"
        fail=1
    }
for each in ws-tunnel:755 wg:755 wireguard-go:755 \
    tunnel-conf.sh:644 tunnel-paths.sh:644; do
    file=/usr/local/libexec/ws-tunnel/${each%:*}
    [ "$(stat -f '%Su:%Sg %Lp' "$file")" = "root:wheel ${each#*:}" ] ||
        {
            echo "::error::$file is not root's, ${each#*:}"
            fail=1
        }
done
if launchctl print "system/$LABEL" >/dev/null 2>&1; then
    echo "::error::setup.sh loaded the daemon without ws's key"
    fail=1
fi
[ "$(ssh -G ws | awk '$1 == "hostname" { print $2 }')" = 127.0.0.1 ] ||
    {
        echo "::error::without ws's key, ssh -G ws does not take the LAN's way"
        fail=1
    }
exit "$fail"
