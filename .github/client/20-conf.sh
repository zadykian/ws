#!/bin/bash
# values.yml, without ws's WireGuard key. It names a file key, made here, as ws's login key, which
# the bootstrap's facts need. The runner must lack one, as client-identity-checks.sh checks the key's
# lines. Git signs with the same key, with no passphrase.

set -eo pipefail
# shellcheck source=.github/client-conf.sh
. .github/client-conf.sh

mkdir -p ~/.ssh
if [ -e ~/.ssh/id_ed25519 ]; then
    echo '::error::the runner has ~/.ssh/id_ed25519 already'
    exit 1
fi
ssh-keygen -q -t ed25519 -N '' -C ws -f ~/.ssh/id_ed25519 </dev/null
printf '%s\n' 'WS_LAN: 127.0.0.0/8' 'WS_LAN_ADDRESS: 127.0.0.1' 'WS_REMOTE: localhost' \
    'WS_REMOTE_PORT: 22022' 'WS_IDENTITY_PUB_KEY: ~/.ssh/id_ed25519.pub' \
    'WS_GIT_SIGNING_PUB_KEY: ~/.ssh/id_ed25519.pub' |
    conf_write /etc/ws/values.yml
# A config of the user's own, which the Include must go above.
if [ ! -e ~/.ssh/config ]; then
    printf 'Host *\n    ServerAliveInterval 60\n' >~/.ssh/config
    chmod 600 ~/.ssh/config
fi
cp ~/.ssh/config "$RUNNER_TEMP/ssh-config-before"
