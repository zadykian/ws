#!/bin/sh
# The mac job's check of the addresses that ssh takes: ssh -G must give ws and ws-ssh the values
# that mac.conf and setup.sh's config.d/ws lead to. The job runs it after the second run of
# setup.sh, in the repository's root, with the job's mac.conf (.github/workflows/mac.yml). Each
# failure prints an ::error:: line, and any fails the script at its end.

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
expect ws-ssh hostname 127.0.0.1
expect ws-ssh port 22
expect ws-ssh user root
expect ws-ssh hostkeyalias ws
expect ws-ssh identitiesonly yes
exit "$fail"
