#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote /etc/ws/ghostty, root's, as ws.ghostty whole. It ended
# the config that Ghostty's menu opens with the line that reads it. The runner has no config of
# Ghostty's of its own, so that config is Ghostty's default.

set -eo pipefail

config="$HOME/Library/Application Support/com.mitchellh.ghostty/config.ghostty"
fail=0
cmp -s /etc/ws/ghostty macos/roles/ghostty/files/ws.ghostty ||
    {
        echo '::error::/etc/ws/ghostty is not macos/roles/ghostty/files/ws.ghostty'
        fail=1
    }
owner=$(stat -f '%Su %Lp' /etc/ws/ghostty)
[ "$owner" = "root 644" ] ||
    {
        echo "::error::/etc/ws/ghostty is $owner, not root 644"
        fail=1
    }
[ "$(tail -n 1 "$config")" = 'config-file = ?/etc/ws/ghostty' ] ||
    {
        echo "::error::Ghostty's config does not end with the line that reads /etc/ws/ghostty"
        fail=1
    }
exit "$fail"
