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

# Without git's key, --check must name its slot and the two commands, and call neither. A run with
# no terminal must stop, naming them too.
with
sh "$standin" drop 30000001 83
make="ykman piv keys generate -a ED25519 --pin-policy NEVER --touch-policy NEVER 83 git.pem, then\
 ykman piv certificates generate -s CN=git -d 3650 83 git.pem"
: >"$yubikey/log"
prints "setup.sh: would make git's key in slot 83 of YubiKey 30000001: $make, which ask the PIN,\
 and the management key unless the PIN protects it${nl}setup.sh: would write $keys/git.pub once\
 the key is made" --check
! grep -q ' generate ' "$yubikey/log" || error "setup.sh --check ran ykman's generate"
stops "setup.sh: no terminal for ykman to make git's key, as it asks the PIN, and the management\
 key unless the PIN protects it: run setup.sh at a terminal, or make the key with $make"

# At a terminal of script's, with the management key and the PIN typed, a run must make the key in
# slot 83, its certificate for 3650 days as the stand-in's log shows, and warn that git's .pub held
# another key. macOS's script writes its stdin's end to the terminal ahead of the lines it still
# holds, so the end comes 5 s after them.
mgm=00112233445566778899aabbccddeeff0011223344556677
{
    printf '%s\n' "$mgm" "$mgm" 24681357
    sleep 5
} | script -q /dev/null sh mac/setup.sh >"$tmp/tty" 2>&1 || error 'setup.sh at a terminal failed'
tr -d '\r' <"$tmp/tty"
if ! grep -q "^setup\.sh: making git's key in slot 83 of YubiKey 30000001: " "$tmp/tty" ||
    ! grep -qF "setup.sh: warning: $keys/git.pub held another key, of another YubiKey or a" \
        "$tmp/tty" || ! same git 83 || [ "$(cat "$yubikey/30000001/83/cn")" != git ]; then
    error "setup.sh at a terminal made no key for git in slot 83, its .pub, or no warning"
fi
grep -qF ' piv certificates generate -s CN=git -d 3650 83 ' "$yubikey/log" ||
    error "setup.sh made git's certificate without -d 3650"
prints ''
prints '' --check

# A key with a PIN or touch policy, or of another type than Ed25519, must stop a run and --check,
# naming the commands that free its slot. Made again with none, its new .pub draws the warning.
again="ykman piv keys delete 83 and ykman piv certificates delete 83 free the slot, and the next\
 run makes the key again"
for policy in 'ONCE NEVER' 'NEVER CACHED'; do
    # shellcheck disable=SC2086 # the two policies, split bare
    sh "$standin" key 30000001 83 git $policy
    for arg in '' --check; do
        stops "setup.sh: git's key, in slot 83, asks for the PIN or a touch at each use, with PIN\
 policy ${policy% *} and touch policy ${policy#* }: $again" $arg
    done
done
sh "$standin" key 30000001 83 git NEVER NEVER ECCP256
for arg in '' --check; do
    stops "setup.sh: git's key, in slot 83, is ECCP256, not ED25519: $again" $arg
done
sh "$standin" key 30000001 83 git NEVER NEVER
prints "setup.sh: warning: $keys/git.pub held another key, of another YubiKey or a reset PIV\
 applet: GitHub and ws need the new one${nl}setup.sh: wrote $keys/git.pub"
same git 83 || error "setup.sh wrote another key than slot 83's to keys/git.pub"
# Two slots of one name must stop a run.
sh "$standin" key 30000001 85 git NEVER NEVER
stops "setup.sh: slots 83 and 85 each hold the certificate CN=git: ykman piv certificates delete"
sh "$standin" drop 30000001 85

# No YubiKey, or two, must stop a run and --check. A YubiKey from the factory draws a warning of its
# defaults, and --check names the slots of both keys it would make. Of firmware 5.4.3, it must stop.
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
setup --check || error "setup.sh --check with a YubiKey from the factory failed: $out"
case $out in
"$warning${nl}setup.sh: would make ws's key in slot 82 of YubiKey 30000002: "*"${nl}setup.sh: would\
 make git's key in slot 83 of YubiKey 30000002: "*) ;;
*) error "setup.sh --check with a YubiKey from the factory gave no warning, or other slots: $out" ;;
esac
echo 5.4.3 >"$yubikey/30000002/version"
setup && error 'setup.sh went on with a YubiKey of firmware 5.4.3'
[ "$out" = "$warning${nl}setup.sh: YubiKey 30000002's firmware, 5.4.3, makes no Ed25519 key, which\
 came in 5.7: plug in a YubiKey of firmware 5.7 or later" ] ||
    error "setup.sh stopped on firmware 5.4.3 with '$out'"
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
