#!/bin/bash
# seward, which mise bootstrap (28-bootstrap.sh) copied into /usr/local/bin: it must be mise's,
# root's and 0755, at its pin, which --version prints.

set -eo pipefail

fail=0
# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

from=$(cd / && mise -C /opt/ws/macos which seward) || error "mise has no seward for /opt/ws"
cmp -s /usr/local/bin/seward "$from" || error "/usr/local/bin/seward is not mise's $from"
owner=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/seward)
[ "$owner" = "root:wheel 755" ] || error "/usr/local/bin/seward is $owner, not root:wheel 755"
got=$(/usr/local/bin/seward --version) || true
[ "$got" = 0.1.0 ] || error "seward --version prints '$got', not '0.1.0'"
exit "$fail"
