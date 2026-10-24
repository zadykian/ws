#!/bin/sh
# Runs the wizard job's cases, NN-*.sh beside it, in order, each in a group of the log. Each takes
# the runner as the one before left it, so the first that fails stops the rest.

set -u

for case in "$(dirname "$0")"/[0-9][0-9]-*.sh; do
    echo "::group::${case##*/}"
    sh "$case"
    status=$?
    echo "::endgroup::"
    if [ "$status" != 0 ]; then
        echo "::error::${case##*/} failed, with status $status"
        exit 1
    fi
done
