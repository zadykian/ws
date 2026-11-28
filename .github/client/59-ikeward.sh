#!/bin/bash
# ikeward's role that mise bootstrap (28-bootstrap.sh) set up: strongSwan is Homebrew's, and
# /usr/local/sbin/ikeward must be mise's. It and its directory must be root's and 0755. A stand-in
# in its place, a copy that differs, must give way to mise's at the next bootstrap, whose plan then
# changes nothing.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

fail=0
# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

bin=/usr/local/sbin/ikeward
# copy checks that the binary is mise's, and that it and its directory are root's and 0755.
copy() {
    from=$(cd / && mise -C /opt/ws/macos which ikeward) || {
        error "mise has no ikeward for /opt/ws"
        return
    }
    cmp -s "$bin" "$from" || error "$bin is not mise's $from"
    for path in "$bin" "$(dirname "$bin")"; do
        owner=$(stat -f '%Su:%Sg %Lp' "$path") || {
            error "$path cannot be read"
            continue
        }
        [ "$owner" = "root:wheel 755" ] || error "$path is $owner, not root:wheel 755"
    done
}
[ -d /opt/homebrew/opt/strongswan ] || error "Homebrew's strongswan is not installed"
copy
printf '#!/bin/sh\n' | sudo tee "$bin" >/dev/null
client_bootstrap || error "mise bootstrap over the stand-in of ikeward failed"
client_bootstrap plan --detailed-exitcode || error "the plan after the stand-in gave way exited $?, not 0"
copy
exit "$fail"
