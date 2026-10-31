#!/bin/bash
# mac.conf, without ws's key.

set -eo pipefail

mkdir -p ~/.config/ws ~/.ssh
printf '%s\n' 'WS_LAN=127.0.0.0/8' 'WS_LAN_ADDRESS=127.0.0.1' 'WS_REMOTE=localhost' \
    'WS_REMOTE_PORT=22022' >~/.config/ws/mac.conf
# A config of the user's own, which the Include must go above.
if [ ! -e ~/.ssh/config ]; then
    printf 'Host *\n    ServerAliveInterval 60\n' >~/.ssh/config
    chmod 600 ~/.ssh/config
fi
cp ~/.ssh/config "$RUNNER_TEMP/ssh-config-before"
