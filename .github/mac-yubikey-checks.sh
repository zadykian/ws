# shellcheck shell=sh
# The mac job's checks of mac/setup.sh's YubiKey step, which mac-setup-checks.sh sources before the
# cases of ws's key, so that they run with its functions and the job's mac.conf. The YubiKey is the
# stand-in's (.github/yubikey-standin.sh): the job's, 30000001, holds ws's key in slot 82 and git's
# in 83. The cases leave it so.
#
# shellcheck disable=SC2154 # nl and the functions are mac-setup-checks.sh's
standin=.github/yubikey-standin.sh

# No YubiKey, or two, must stop a run and --check. A YubiKey from the factory draws a warning of its
# defaults at each run.
with
sh "$standin" unplug 30000001
for arg in '' --check; do
    stops "setup.sh: no YubiKey is in: plug in the one with ws's keys, or a new one" $arg
done
sh "$standin" plug 30000001
sh "$standin" new 30000002
stops 'setup.sh: several YubiKeys are in, 30000001 and 30000002: leave in the one' --check
sh "$standin" unplug 30000001
warning="setup.sh: warning: YubiKey 30000002 keeps the PIV applet's default PIN, PUK and management\
 key: ykman piv access changes them, as docs/mac-yubikey.md says"
prints "$warning"
prints "$warning" --check
sh "$standin" unplug 30000002
sh "$standin" plug 30000001
prints ''

# Without ykman and yubico-piv-tool, --check must name their install, and the keys it would find
# once ykman is installed. PATH lacks each directory that holds a ykman, the stand-in's too. A
# stand-in brew in $tmp/no-tools/bin lacks yubico-piv-tool, and passes other calls to the runner's.
mkdir -p "$tmp/no-tools/bin"
cat >"$tmp/no-tools/bin/brew" <<BREW
#!/bin/sh
[ "\$*" != 'list --formula yubico-piv-tool' ] || exit 1
exec '$(command -v brew)' "\$@"
BREW
chmod 755 "$tmp/no-tools/bin/brew"
path=$PATH
PATH=$tmp/no-tools/bin$(echo "$PATH" | tr : '\n' | while read -r each; do
    [ -x "$each/ykman" ] || printf ':%s' "$each"
done)
setup --check || error "setup.sh --check without ykman failed: $out"
PATH=$path
case $out in
"setup.sh: would run brew install ykman yubico-piv-tool${nl}setup.sh: would find "*" once ykman\
 is installed") ;;
*) error "setup.sh --check without ykman and yubico-piv-tool printed '$out'" ;;
esac
