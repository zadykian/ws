#!/bin/bash
# ~/.ssh/config has the Include first and keeps its own lines.

set -eo pipefail

head -n 1 ~/.ssh/config | grep -qx 'Include config.d/ws' ||
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
        echo '::error::setup.sh kept no copy of the old ~/.ssh/config'
        exit 1
    }
[ "$(stat -f %Lp ~/.ssh/config.d)" = 700 ] ||
    {
        echo '::error::~/.ssh/config.d is not 700'
        exit 1
    }
[ "$(stat -f %Lp ~/.ssh/config.d/ws)" = 600 ] ||
    {
        echo '::error::~/.ssh/config.d/ws is not 600'
        exit 1
    }
