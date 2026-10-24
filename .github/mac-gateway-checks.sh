# shellcheck shell=sh
# The mac job's checks of mac/setup.sh's Gateway step, which needs no key. mac-setup-checks.sh
# sources them before git's, so that they run with its helpers and the job's mac.conf. The job made
# ~/Applications/JetBrains Gateway.app, an empty directory, which spares its runs the cask: the
# cases take it away, and put it back as they end.
#
# shellcheck disable=SC2154 # tmp, nl and the helpers are mac-setup-checks.sh's
app='JetBrains Gateway.app'
if [ -e "/Applications/$app" ] || ! rmdir ~/Applications/"$app"; then
    error "the runner has a $app of its own, or lacks the job's empty one, so Gateway's cases go"
    exit 1
fi

# A stand-in brew goes in $tmp/bin, first on PATH. It passes the three calls of setup.sh's tools to
# the runner's brew, and logs any other. Of those, it takes the cask's list, which succeeds where
# $tmp/cask is, and its install alone. The install makes $tmp/cask and an empty app in
# ~/Applications. It fails with $tmp/cask.fail, and makes no app with $tmp/cask.none.
brew=$(command -v brew)
: >"$tmp/cask.log"
cat >"$tmp/bin/brew" <<EOF
#!/bin/sh
case \$* in
'list --formula wireguard-tools' | '--prefix wireguard-tools' | '--prefix wireguard-go')
    exec '$brew' "\$@"
    ;;
esac
echo "\$*" >>'$tmp/cask.log'
case \$* in
'list --cask jetbrains-gateway') [ -f '$tmp/cask' ] ;;
'install --cask jetbrains-gateway')
    [ ! -f '$tmp/cask.fail' ] || { echo 'Error: the stand-in fails' >&2 && exit 1; }
    : >'$tmp/cask'
    [ -f '$tmp/cask.none' ] || mkdir ~/Applications/'$app'
    ;;
*) echo "brew: the stand-in takes no \$*" >&2 && exit 2 ;;
esac
EOF
chmod 755 "$tmp/bin/brew"
if [ "$(command -v brew)" != "$tmp/bin/brew" ]; then
    error "brew is $(command -v brew), not the stand-in, so Gateway's cases go"
    exit 1
fi

# brewed CALL... checks that the stand-in brew logged CALL..., a line each, since the last check.
brewed() {
    [ "$(cat "$tmp/cask.log")" = "$(printf '%s\n' "$@")" ] ||
        error "the stand-in brew logged '$(cat "$tmp/cask.log")', not '$*'"
    : >"$tmp/cask.log"
}

# Without the app, --check must name the cask's install and run none, a run must install it once,
# and a second run and --check print nothing.
with
list='list --cask jetbrains-gateway'
install='install --cask jetbrains-gateway'
prints "setup.sh: would run brew $install" --check
brewed "$list"
prints 'setup.sh: installing jetbrains-gateway'
brewed "$list" "$install"
prints ''
prints '' --check
brewed

# An app in /Applications must spare the install.
rmdir ~/Applications/"$app"
sudo mkdir -p "/Applications/$app"
prints ''
prints '' --check
brewed
sudo rmdir "/Applications/$app"
mkdir ~/Applications/"$app"
rm "$tmp/bin/brew"
