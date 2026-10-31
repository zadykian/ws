#!/bin/bash
# The runner.

set -eo pipefail

sw_vers
uname -m
ssh -V
ls -ld / /usr /usr/local /usr/local/libexec 2>&1 || true
