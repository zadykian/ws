#!/bin/sh
# Runs the mac job's cases, NN-*.sh beside it, in order, each in a group of the log, from the
# repository's root. Each takes the runner as the one before left it, so the first that fails
# stops the rest. The cases are bash, with pipefail, which their scripts set.

set -u

for case in "$(dirname "$0")"/[0-9][0-9]-*.sh; do
    echo "::group::${case##*/}"
    bash "$case"
    status=$?
    echo "::endgroup::"
    if [ "$status" != 0 ]; then
        echo "::error::${case##*/} failed, with status $status"
        exit 1
    fi
done
