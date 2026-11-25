#!/bin/bash
# Without ws's key, mise bootstrap (28-bootstrap.sh) made the Mac's key and installed no daemon.
# /etc/ws/tunnel.pub, readable by all, must hold the Mac's public key, which the stand-in ws below
# takes as its peer. ws macos tunnel key must print it again and keep the key. The key
# must be root's alone. ssh -G must give ws the tunnel's end, from mise bootstrap's
# /etc/ws/ssh_config.

set -eo pipefail

fail=0
wg=$(brew --prefix wireguard-tools)/bin/wg
mac=$(sudo sh -c '"$1" pubkey <"$2"' sh "$wg" /etc/ws/tunnel/private.key) ||
    {
        echo "::error::cannot read the Mac's public key"
        exit 1
    }
key=$(cat /etc/ws/tunnel.pub)
if [[ ! $key =~ ^[A-Za-z0-9+/]{43}=$ ]] || [ "$key" != "$mac" ]; then
    echo "::error::/etc/ws/tunnel.pub holds '$key' as the Mac's public key, not '$mac'"
    fail=1
fi
again=$(sudo /usr/local/bin/ws macos tunnel key) || {
    echo '::error::ws macos tunnel key failed with the key there'
    fail=1
}
[ "$again" = "$mac" ] || {
    echo "::error::ws macos tunnel key printed '$again', not '$mac'"
    fail=1
}
printf '%s\n' "$mac" >"$RUNNER_TEMP/mac.pub"
for want in "/etc/ws/tunnel/private.key root:wheel 600" "/etc/ws/tunnel.pub root:wheel 644" \
    "/etc/ws/tunnel root:wheel 755"; do
    path=${want%% *}
    got=$(sudo stat -f '%N %Su:%Sg %Lp' "$path")
    [ "$got" = "$want" ] || {
        echo "::error::$got, not $want"
        fail=1
    }
done
if launchctl print "system/$LABEL" >/dev/null 2>&1 || [ -e "/Library/LaunchDaemons/$LABEL.plist" ]; then
    echo "::error::mise bootstrap installed the daemon without ws's key"
    fail=1
fi
[ "$(ssh -T -G ws | awk '$1 == "hostname" { print $2 }')" = 10.99.0.1 ] ||
    {
        echo "::error::without ws's key, ssh -G ws does not give the tunnel's end"
        fail=1
    }
exit "$fail"
