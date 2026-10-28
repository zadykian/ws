# shellcheck shell=sh
# The mac job's checks of mac/setup.sh's YubiKey step, which mac-setup-checks.sh sources before the
# cases of ws's key, so that they run with its functions and the job's mac.conf. The YubiKey is the
# stand-in's (.github/yubikey-standin.sh): the job's, 30000001, holds ws's key in slot 82 and git's
# in 83. The cases leave it so, with git's key made again.
#
# shellcheck disable=SC2154 # tmp, nl and the functions are mac-setup-checks.sh's
standin=.github/yubikey-standin.sh
yubikey=$RUNNER_TEMP/yubikey
keys=$HOME/.config/ws/keys
openssl=$(sed -n 's/^openssl=//p' "$yubikey/env")

# same NAME SLOT succeeds where keys/NAME.pub, mode 644, holds the key of the job's YubiKey in SLOT:
# the bytes that end its blob, as openssl reads them from the stand-in's public key. ssh-keygen must
# read it too.
same() {
    key=$("$openssl" pkey -pubin -inform DER -in "$yubikey/30000001/$2/pub.der" -noout -text |
        awk '/^pub:$/ { on = 1; next } on && /^ / { gsub(/[ :]/, ""); printf "%s", $0; next }
            { on = 0 }')
    blob=$(awk '{ print $2 }' "$keys/$1.pub" | base64 -d | tail -c "$((${#key} / 2))" |
        od -An -tx1 | tr -d ' \n')
    [ "$(stat -f %Lp "$keys/$1.pub")" = 644 ] && [ -n "$key" ] && [ "$blob" = "$key" ] &&
        [ "$(awk '{ print $3 }' "$keys/$1.pub")" = "$1@yubikey" ] &&
        ssh-keygen -l -f "$keys/$1.pub" >/dev/null
}

# The job's runs wrote both keys' .pubs.
if ! same ws 82 || ! same git 83 || [ "$(stat -f %Lp "$keys")" != 755 ]; then
    error "keys/ws.pub and keys/git.pub are not the stand-in's keys, 644, in a directory of 755"
fi

# Without git's key, a run and --check must stop.
with
sh "$standin" drop 30000001 83
for arg in '' --check; do
    stops "setup.sh: YubiKey 30000001 holds no key git, whose certificate is CN=git: make it as\
 docs/mac-yubikey.md says" $arg
done

# Made again, the key's new .pub draws the warning.
sh "$standin" key 30000001 83 git NEVER NEVER
prints "setup.sh: warning: $keys/git.pub held another key, of another YubiKey or a reset PIV\
 applet: GitHub and ws need the new one${nl}setup.sh: wrote $keys/git.pub"
same git 83 || error "setup.sh wrote another key than slot 83's to keys/git.pub"

# No YubiKey, or two, must stop a run and --check. A YubiKey from the factory draws a warning of its
# defaults, and stops a run at the first key it lacks.
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
setup && error 'setup.sh went on with a YubiKey from the factory'
case $out in
"$warning${nl}setup.sh: YubiKey 30000002 holds no key ws, whose certificate is CN=ws: make it as\
 docs/mac-yubikey.md says") ;;
*) error "setup.sh with a YubiKey from the factory gave no warning, or another stop: $out" ;;
esac
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
