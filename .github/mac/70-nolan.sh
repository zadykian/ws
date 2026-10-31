#!/bin/bash
# Without the LAN's keys, ws-ssh goes to WS_REMOTE.

set -eo pipefail

sed -i '' '/^WS_LAN/d' ~/.config/ws/mac.conf
sh mac/setup.sh || {
    echo '::error::setup.sh without the LAN failed'
    exit 1
}
fail=0
for pair in hostname=localhost port=22022 addressfamily=inet; do
    got=$(ssh -G ws-ssh | awk -v k="${pair%%=*}" '$1 == k { print $2; exit }')
    if [ "$got" != "${pair#*=}" ]; then
        echo "::error::ssh -G ws-ssh gives ${pair%%=*} '$got', not '${pair#*=}'"
        fail=1
    fi
done
exit "$fail"
