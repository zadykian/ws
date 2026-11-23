#!/bin/bash
# values.yml's keys. Their checks run mise bootstrap.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

sh .github/client-conf-checks.sh
