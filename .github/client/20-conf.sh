#!/bin/bash
# values.yml, without ws's WireGuard key. It names seward's two keys, which the bootstrap's units
# make, so that setup's key step plans nothing. Nothing else reads them.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh

mkdir -p ~/.ssh
printf '%s\n' 'WS_LAN: 127.0.0.0/8' 'WS_LAN_ADDRESS: 127.0.0.1' 'WS_REMOTE: localhost' \
    'WS_REMOTE_PORT: 22022' 'WS_IDENTITY_PUB_KEY: ~/.ssh/ws-auth.pub' \
    'WS_GIT_SIGNING_PUB_KEY: ~/.ssh/ws-sign.pub' |
    conf_write /etc/ws/values.yml
# A config of the user's own, which the Include must go above.
if [ ! -e ~/.ssh/config ]; then
    printf 'Host *\n    ServerAliveInterval 60\n' >~/.ssh/config
    chmod 600 ~/.ssh/config
fi
cp ~/.ssh/config "$RUNNER_TEMP/ssh-config-before"
