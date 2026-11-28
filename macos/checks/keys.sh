#!/bin/bash
# The Mac's check of seward's keys: trial 11. Run as the Mac's user in a terminal of the login
# session, where alone a key signs, as `keys.sh [REPO SHA]`, inside the clone that holds the commit.
# It makes no key and no commit. The page, macos/checks/README.md, says what to pass.
set -uo pipefail

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=macos/checks/lib.sh
. "$dir/lib.sh"

seward=${SEWARD_BIN:-/usr/local/bin/seward}
# keys are the two keys that setup makes, and sock is the agent's socket in ~/.ssh.
keys="ws-auth ws-sign"
sock=$HOME/.ssh/seward.sock

# store_is NAME STORE succeeds where seward list names STORE as the key's store, its fourth column.
store_is() {
    local got
    got=$("$seward" list 2>/dev/null | awk -F'\t' -v name="$1" '$1 == name { print $4 }')
    [ "$got" = "$2" ] || {
        echo "$1 is in the store '$got', not $2"
        return 1
    }
}

# agent_lists NAME succeeds where the agent on the socket lists the key of ~/.ssh/NAME.pub.
agent_lists() {
    local key listed
    key=$(awk '{ print $2 }' "$HOME/.ssh/$1.pub") && [ -n "$key" ] || {
        echo "$HOME/.ssh/$1.pub holds no key"
        return 1
    }
    listed=$(SSH_AUTH_SOCK=$sock ssh-add -L 2>&1) || {
        echo "ssh-add -L on $sock: $(printf '%s' "$listed" | tail -n 1)"
        return 1
    }
    grep -qF -e " $key" <<<"$listed" || {
        echo "the agent on $sock does not list the key of $1.pub"
        return 1
    }
}

# verified REPO SHA succeeds where GitHub reads the commit as verified.
verified() {
    local got
    got=$(gh api "repos/$1/commits/$2" --jq '.commit.verification | "\(.verified) \(.reason)"' 2>&1) || {
        echo "gh api: $(printf '%s' "$got" | tail -n 1)"
        return 1
    }
    case $got in
    "true "*) ;;
    *)
        echo "GitHub reads $2 as: $got"
        return 1
        ;;
    esac
}

check11() {
    local key
    check "11 seward list loads every key" "$seward" list
    for key in $keys; do
        check "11 $key is a key of the Secure Enclave" store_is "$key" enclave
        check "11 the agent lists the key of $key.pub" agent_lists "$key"
    done
    check "11 ssh logs in to ws by ws-auth with no prompt" ssh -o BatchMode=yes ws true
    if [ $# -gt 0 ]; then
        check "11 git verifies the commit $2" git verify-commit "$2"
        check "11 GitHub reads the commit $2 as verified" verified "$1" "$2"
    else
        waits "11 sign a commit by hand and push it, then give its repository and SHA: keys.sh REPO SHA"
    fi
}

main() {
    case $# in
    0 | 2) check11 "$@" ;;
    *)
        echo "usage: keys.sh [REPO SHA]" >&2
        exit 2
        ;;
    esac
    finish
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
