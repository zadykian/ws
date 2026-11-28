#!/bin/bash
# seward, which mise bootstrap (28-bootstrap.sh) copied into /usr/local/bin: it must be mise's,
# root's and 0755, at its pin, which --version prints. Its agent must run that copy as mise's user
# service, answer on its socket, and restart where a new pin changes the plist.

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
# no agent answered, and 1 that it holds no key, as no key exists yet.
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
plutil -replace EnvironmentVariables.WS_SEWARD_VERSION -string 0.0.0 "$plist"
client_bootstrap || {
    echo "::error::mise bootstrap with the plist's version changed failed"
    exit 1
}
second=$(up) || {
    echo "::error::launchd gave $label no pid after the bootstrap"
    exit 1
}
[ "$second" != "$first" ] || error "the agent kept pid $first where its plist changed"
version=$(plutil -extract EnvironmentVariables.WS_SEWARD_VERSION raw "$plist")
[ "$version" = 0.1.0 ] || error "the plist's WS_SEWARD_VERSION is '$version', not '0.1.0'"
runs_copy "$second" || error "the restarted agent's pid $second does not run /usr/local/bin/seward"
serves || error "ssh-add -L on $sock exits 2 after the restart"
client_bootstrap plan --detailed-exitcode || error "the plan after the restart exited $?, not 0"
exit "$fail"
