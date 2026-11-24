#!/bin/sh
# CI's run of ws ubuntu guest setup, as on a new machine, in the server job. It runs on the tree in
# /opt/ws in place, as the checkout is not a tag for setup to fetch. Run as `bootstrap.sh [ARG]...`,
# ARG setup's flags, --in-place and --plan. The ws that runs is /usr/local/bin/ws, which
# .tools/install-build put there.
set -eu
if [ ! -x /usr/local/bin/ws ]; then
    echo '::error::/usr/local/bin/ws is missing: .tools/install-build installs it.'
    exit 1
fi
exec /usr/local/bin/ws ubuntu guest setup "$@"
