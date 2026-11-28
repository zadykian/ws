#!/bin/bash
# ws macos ssh keygen, list and remove, which hand the call to seward (57-seward.sh): a scratch key
# made with --store file must show in list with that store. remove must refuse with no terminal and
# no --yes, and with --yes delete both files. A bad name must exit 2.

set -eo pipefail

fail=0
# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

name=ws-case-59
key=$HOME/.ssh/$name
# The case's own key goes at its end, a failure included.
trap 'ws macos ssh remove "$name" --yes >/dev/null 2>&1 || true' EXIT

ws macos ssh keygen "$name" --store file || error "ws macos ssh keygen $name --store file exited $?"
[ -f "$key" ] && [ -f "$key.pub" ] || error "keygen left no $key and $key.pub"
tab=$(printf '\t')
ws macos ssh list | grep -q "^$name$tab.*${tab}file\$" || error "list shows no $name with the store file"

status=0
ws macos ssh remove "$name" </dev/null >/dev/null 2>&1 || status=$?
[ "$status" = 2 ] || error "remove with no terminal and no --yes exited $status, not 2"
[ -f "$key" ] && [ -f "$key.pub" ] || error "a refused remove deleted a file of $name"

ws macos ssh remove "$name" --yes || error "remove --yes exited $?"
[ ! -e "$key" ] && [ ! -e "$key.pub" ] || error "remove --yes left a file of $name"

status=0
ws macos ssh keygen ../bad </dev/null >/dev/null 2>&1 || status=$?
[ "$status" = 2 ] || error "keygen with a bad name exited $status, not 2"
exit "$fail"
