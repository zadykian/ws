#!/bin/bash
# values.yml's keys. Their checks run mise bootstrap.

set -eo pipefail
# shellcheck source=.github/mac-mise.sh
. .github/mac-mise.sh

sh .github/mac-conf-checks.sh
