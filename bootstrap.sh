#!/bin/sh
# Sets up this machine from github.com/zadykian/ws. Run it as root on a fresh Ubuntu Server 26.04:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh
#
# or, for a dry run that shows what the playbook would change:
#
#   curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh |
#       sh -s -- --check --diff
#
# It installs ansible-core and git with apt, clones the repository into /root/repository/ws, or
# leaves a clone there as it is, points git's core.hooksPath at its .githooks, installs the
# collections of its requirements.yml, and runs its site.yml. Its arguments go to
# ansible-playbook. A dry run takes these first steps too, as the playbook cannot run without them.
#
# Everything is in functions that the last line calls, so that a download cut short runs nothing.

set -eu

repository=https://github.com/zadykian/ws
dir=/root/repository/ws

# say prints a message of bootstrap.sh's.
say() {
    printf 'bootstrap.sh: %s\n' "$*" >&2
}

# die prints a message of bootstrap.sh's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# system checks that it runs as root on Ubuntu 26.04, the system the playbook sets up.
system() {
    [ "$(id -u)" = 0 ] || die "run it as root"
    # shellcheck source=/dev/null
    release=$(. /etc/os-release && echo "$ID $VERSION_ID") || release=
    [ "$release" = "ubuntu 26.04" ] || die "it sets up Ubuntu 26.04, not ${release:-this system}"
}

# packages installs ansible-core and git. Without recommends: ansible-core's bring Ubuntu's
# ansible package, whose collections would stand in for any requirements.yml lacks. Without them,
# python3-apt and ca-certificates are named: Ansible's apt module installs python3-apt itself
# only in a real run, not a dry one, and git clones over HTTPS.
#
# On a new machine, apt's daily timers run soon after the first boot and hold apt's locks for a
# while. apt-get update has no option to wait for its lock, so it tries again for 10 minutes;
# apt-get install waits as long for dpkg's.
packages() {
    say "installing ansible-core and git"
    export DEBIAN_FRONTEND=noninteractive
    tries=1
    until apt-get update; do
        [ "$tries" -lt 60 ] || die "apt-get update failed"
        tries=$((tries + 1))
        say "apt-get update failed, trying again in 10 seconds"
        sleep 10
    done
    apt-get -o DPkg::Lock::Timeout=600 install -y --no-install-recommends \
        ansible-core ca-certificates git python3-apt || die "cannot install ansible-core and git"
}

# clone clones the repository into dir, or leaves the clone there as it is, so that a run applies
# what the clone holds, and enables the hooks in its .githooks.
clone() {
    if [ -e "$dir/.git" ]; then
        say "using the clone in $dir as it is"
    elif [ -e "$dir" ]; then
        die "$dir is there, but is not a clone"
    else
        mkdir -p "${dir%/*}"
        git clone "$repository" "$dir" || die "cannot clone $repository"
    fi
    git -C "$dir" config core.hooksPath .githooks
}

main() {
    system
    packages
    clone
    cd "$dir"
    ansible-galaxy collection install -r requirements.yml || die "cannot install the collections"
    say "running site.yml"
    ansible-playbook site.yml "$@"
}

# Nothing it runs reads stdin, which under curl | sh is the rest of this script. The braces make a
# line cut short a syntax error, rather than main run without its arguments.
{ main "$@" </dev/null; }
