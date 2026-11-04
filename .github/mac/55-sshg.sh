#!/bin/bash
# ssh -G gives ws the tunnel's end, and its user, alias and key.

set -eo pipefail

fail=0
# expect HOST KEY VALUE: ssh -G HOST must give KEY the value VALUE.
expect() {
    got=$(ssh -G "$1" | awk -v k="$2" '$1 == k { print $2; exit }')
    if [ "$got" != "$3" ]; then
        echo "::error::ssh -G $1 gives $2 '$got', not '$3'"
        fail=1
    fi
}
expect ws hostname 10.99.0.1
expect ws port 22
expect ws user root
expect ws hostkeyalias ws
expect ws serveraliveinterval 30
expect ws identitiesonly yes
exit "$fail"
