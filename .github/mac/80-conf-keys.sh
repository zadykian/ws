#!/bin/bash
# values.yml's keys. Their checks run mise bootstrap, so the build is served for them.

set -eo pipefail
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

mac_serve
trap 'kill "$mac_server"' EXIT
sh .github/mac-conf-checks.sh
