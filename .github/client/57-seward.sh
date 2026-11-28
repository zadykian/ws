#!/bin/bash
# seward, which mise bootstrap (28-bootstrap.sh) copied into /usr/local/bin: it must be mise's,
# root's and 0755, at its pin, which --version prints. Its agent must run that copy as mise's user
# service, answer on its socket, and restart where a new pin changes the plist. The runner is a
# virtual machine, so ws-auth and ws-sign must be the user's own files in the file store, and the
# agent must list both.

set -eo pipefail
# shellcheck source=.github/client-mise.sh
. .github/client-mise.sh

fail=0
# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

from=$(cd / && mise -C /opt/ws/macos which seward) || error "mise has no seward for /opt/ws"
cmp -s /usr/local/bin/seward "$from" || error "/usr/local/bin/seward is not mise's $from"
owner=$(stat -f '%Su:%Sg %Lp' /usr/local/bin/seward)
[ "$owner" = "root:wheel 755" ] || error "/usr/local/bin/seward is $owner, not root:wheel 755"
got=$(/usr/local/bin/seward --version) || true
[ "$got" = 0.1.0 ] || error "seward --version prints '$got', not '0.1.0'"
[ "$fail" = 0 ] || exit 1

label=dev.mise.seward
plist=$HOME/Library/LaunchAgents/$label.plist
sock=$HOME/.ssh/seward.sock

# pid prints the agent's pid, which launchd gives once it runs.
pid() { launchctl print "gui/$(id -u)/$label" 2>/dev/null | awk '$1 == "pid" { print $3; exit }'; }

# runs_copy PID succeeds where the process maps /usr/local/bin/seward's own file, by inode. lsof
# keeps the path of a replaced file, so only the inode names the build.
runs_copy() {
    lsof -a -p "$1" -d txt -F i | grep -qx "i$(stat -f %i /usr/local/bin/seward)"
}

# serves waits up to 30 s for the agent's socket to answer ssh-add -L with 0 or 1. Exit 2 says that
# no agent answered, and 1 that it holds no key.
serves() {
    local status
    for _ in $(seq 1 30); do
        status=0
        SSH_AUTH_SOCK=$sock ssh-add -L >/dev/null 2>&1 || status=$?
        [ "$status" -le 1 ] && return 0
        sleep 1
    done
    return 1
}

# up waits up to 30 s for the agent's pid, and prints it.
up() {
    local now
    for _ in $(seq 1 30); do
        now=$(pid) || true
        [ -z "$now" ] || {
            echo "$now"
            return 0
        }
        sleep 1
    done
    return 1
}

first=$(up) || {
    echo "::error::launchd gave $label no pid in 30 s"
    exit 1
}
runs_copy "$first" || error "the agent's pid $first does not run /usr/local/bin/seward"
serves || error "ssh-add -L on $sock exits 2: no agent answers"

# A new pin changes WS_SEWARD_VERSION in the plist: mise restarts the agent in the run that sees it.
plutil -replace EnvironmentVariables.WS_SEWARD_VERSION -string 0.0.0 "$plist" || {
    echo "::error::plutil could not change WS_SEWARD_VERSION in $plist"
    exit 1
}
client_bootstrap || {
    echo "::error::mise bootstrap with the plist's version changed failed"
    exit 1
}
second=$(up) || {
    echo "::error::launchd gave $label no pid after the bootstrap"
    exit 1
}
[ "$second" != "$first" ] || error "the agent kept pid $first where its plist changed"
version=$(plutil -extract EnvironmentVariables.WS_SEWARD_VERSION raw "$plist") || {
    echo "::error::plutil could not read WS_SEWARD_VERSION in $plist"
    exit 1
}
[ "$version" = 0.1.0 ] || error "the plist's WS_SEWARD_VERSION is '$version', not '0.1.0'"
runs_copy "$second" || error "the restarted agent's pid $second does not run /usr/local/bin/seward"
serves || error "ssh-add -L on $sock exits 2 after the restart"

# keys checks each key. seward list must name it with the store file, which the fact .VM chose on
# this runner. The user must own its files, at 600 and 644, and the agent's socket must list it.
keys() {
    local name file mode body list agent
    list=$(/usr/local/bin/seward list) || error "seward list exits $?"
    agent=$(SSH_AUTH_SOCK=$sock ssh-add -L) || error "ssh-add -L on $sock exits $?"
    for name in ws-auth ws-sign; do
        printf '%s\n' "$list" | awk -F'\t' -v n="$name" '$1 == n && $4 == "file" { found = 1 } END { exit !found }' ||
            error "seward list names no $name with the store file: $list"
        for file in "$HOME/.ssh/$name" "$HOME/.ssh/$name.pub"; do
            [ "$(stat -f %Su "$file")" = "$(id -un)" ] || error "$file is not the user's"
            mode=600
            [ "$file" = "$HOME/.ssh/$name" ] || mode=644
            [ "$(stat -f %Lp "$file")" = "$mode" ] || error "$file is $(stat -f %Lp "$file"), not $mode"
        done
        # An empty body would match every line, so the .pub must hold one.
        body=$(awk '{ print $2 }' "$HOME/.ssh/$name.pub") || body=
        if [ -z "$body" ]; then
            error "$HOME/.ssh/$name.pub holds no key, or cannot be read"
        else
            printf '%s\n' "$agent" | grep -qF "$body" || error "the agent on $sock does not list $name"
        fi
    done
}
keys

client_bootstrap plan --detailed-exitcode || error "the plan after the restart exited $?, not 0"
exit "$fail"
