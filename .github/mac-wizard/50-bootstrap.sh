#!/bin/sh
# bootstrap.sh on the Mac that 40-again.sh leaves, by sh from the clone's file on stdin, as
# curl | sh runs it, with standin.sh's PATH, which lacks brew. It must find the runner's Command
# Line Tools and brew, use the clone as it is, and point its core.hooksPath at its .githooks once.
# The wizard it runs prints nothing, so a second run and --check must print nothing at all. Where
# WS_CONF names no file, the wizard under bootstrap.sh's --check must show the lines it would write
# there, from the environment's answers. Writing none shows that --check reached it. In a home
# without a clone, --check must list the clone and wizard.sh's stages, and make nothing.

set -u

clone=$HOME/repository/ws
bootstrap=$clone/mac/bootstrap.sh
path=$(cat "$RUNNER_TEMP/path")
fail=0

# error MESSAGE... prints an ::error:: line, and fails the script at its end.
error() {
    echo "::error::$*"
    fail=1
}

# boot WANT ARG...: bootstrap.sh by stdin, with ARGs, must exit 0 and print WANT alone. It must
# leave the clone's HEAD and files as they were.
boot() {
    want=$1
    shift
    out=$(PATH=$path sh -s -- "$@" <"$bootstrap" 2>&1)
    status=$?
    printf '%s\n' "$out"
    [ "$status" = 0 ] || error "bootstrap.sh $* exited $status"
    [ "$out" = "$want" ] || error "bootstrap.sh $* printed other than '$want'"
    [ "$(git -C "$clone" rev-parse HEAD) $(git -C "$clone" status --porcelain)" = "$kept" ] ||
        error "bootstrap.sh $* changed the clone"
}

# The checkout sets no core.hooksPath, and none is set here, so that the first run sets it.
git -C "$clone" config --unset core.hooksPath
kept="$(git -C "$clone" rev-parse HEAD) $(git -C "$clone" status --porcelain)"
out=$(sh -s -- --bogus <"$bootstrap" 2>&1)
status=$?
[ "$status" = 1 ] && [ "$out" = "bootstrap.sh: unknown argument '--bogus'; usage: bootstrap.sh \
[--check] [--update]" ] || error "bootstrap.sh --bogus exited $status, printing '$out'"
boot "bootstrap.sh: Clone: would set core.hooksPath to $clone/.githooks" --check
boot "bootstrap.sh: Clone: set core.hooksPath to $clone/.githooks"
[ "$(git -C "$clone" config core.hooksPath)" = "$clone/.githooks" ] ||
    error 'bootstrap.sh set no core.hooksPath'
boot ''
boot '' --check

# The wizard, given bootstrap.sh's arguments, shows what it would write to a mac.conf not there.
conf=$RUNNER_TEMP/boot.conf
out=$(WS_CONF=$conf WS_TARGET=lan WS_LAN='' WS_REMOTE=localhost WS_REMOTE_PORT=22022 \
    WS_GIT_NAME='' WS_TUNNEL_NETWORKS='' WS_RESOLVERS='' WS_HOOK='' PATH=$path \
    sh -s -- --check <"$bootstrap" 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | grep -qx '+WS_REMOTE=localhost' && [ "$status" = 0 ] ||
    error "bootstrap.sh --check exited $status, with no line the wizard would write to $conf"
[ ! -e "$conf" ] || error "bootstrap.sh --check let the wizard write $conf"
rm -f "$conf"

# The wizard's stages, as its main runs them, by name, in quotes where they hold a blank.
home=$(mktemp -d "$RUNNER_TEMP/home.XXXXXX")
want=$(
    echo "bootstrap.sh: Clone: would clone https://github.com/zadykian/ws into $home/repository/ws"
    sed -n -e 's/^    stage "\(.*\)" [a-z_]*$/\1/p' -e 's/^    stage \([^ "]*\) [a-z_]*$/\1/p' \
        "$clone/mac/wizard.sh" | sed 's/.*/wizard.sh: &: after Clone/'
)
out=$(HOME=$home PATH=$path sh -s -- --check <"$bootstrap" 2>&1)
status=$?
printf '%s\n' "$out"
[ "$status" = 0 ] && [ "$out" = "$want" ] ||
    error "bootstrap.sh --check without a clone exited $status, printing other than '$want'"
[ -z "$(ls -A "$home")" ] || error "bootstrap.sh --check made $(ls -A "$home") without a clone"
rm -rf "$home"
exit "$fail"
