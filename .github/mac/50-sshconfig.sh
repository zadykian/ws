#!/bin/bash
# mise bootstrap (28-bootstrap.sh) put the Include first in ~/.ssh/config, which keeps its own
# lines, and wrote /etc/ws/ssh_config, root's.

set -eo pipefail

head -n 1 ~/.ssh/config | grep -qx 'Include /etc/ws/ssh_config' ||
    {
        echo '::error::the Include is not first'
        cat ~/.ssh/config
        exit 1
    }
tail -n +3 ~/.ssh/config | cmp -s - "$RUNNER_TEMP/ssh-config-before" ||
    {
        echo "::error::~/.ssh/config lost its own lines"
        cat ~/.ssh/config
        exit 1
    }
ls ~/.ssh/config.*~ ||
    {
        echo '::error::mise bootstrap kept no copy of the old ~/.ssh/config'
        exit 1
    }
[ "$(stat -f '%Su:%Sg %Lp' /etc/ws/ssh_config)" = 'root:wheel 644' ] ||
    {
        echo '::error::/etc/ws/ssh_config is not root:wheel 644'
        exit 1
    }
