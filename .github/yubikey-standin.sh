#!/bin/sh
# A stand-in YubiKey 5C Nano for the mac and wizard jobs, on GitHub's macOS runner, whose keys a
# SoftHSM2 token holds. Run by sh, it sets the stand-in up, or changes its YubiKeys for a case:
#
#   sh .github/yubikey-standin.sh                       the job's YubiKey, with ws's and git's keys
#   sh .github/yubikey-standin.sh new SERIAL            a YubiKey from the factory, plugged in
#   sh .github/yubikey-standin.sh access SERIAL         its PIN, PUK and management key changed
#   sh .github/yubikey-standin.sh key SERIAL SLOT NAME PIN TOUCH   a key, as ykman makes one
#   sh .github/yubikey-standin.sh drop SERIAL SLOT      a slot's key and certificate gone
#   sh .github/yubikey-standin.sh plug|unplug SERIAL    a YubiKey in or out
#
# The setup installs Homebrew's softhsm and yubico-piv-tool, and replaces yubico-piv-tool's
# lib/libykcs11.dylib with a link to SoftHSM2's module. So an agent that loads libykcs11 from
# Homebrew loads the stand-in's tokens, with no setting of its own. SoftHSM2's config, at its
# default for Homebrew, $(brew --prefix)/etc/softhsm/softhsm2.conf, or SOFTHSM2_CONF where set,
# keeps the tokens in the stand-in's. The job's YubiKey, 30000001, has firmware 5.7.1, no default
# PIN, PUK or management key, and the keys ws and git in slots 82 and 83, of policy NEVER NEVER.
#
# Its state is in S, $RUNNER_TEMP/yubikey:
#
# - SERIAL/ is a YubiKey: version holds its firmware, in is there while it is plugged in, and access
#   once its PIN, PUK and management key are no longer the defaults;
# - SERIAL/SLOT/ is a slot, 82 to 95: key.pem holds its private key, kept for the token, pub.der its
#   public key, policy its PIN and touch policies, as NEVER NEVER, and cn its certificate's CN. A
#   slot with cn and no pub.der holds a certificate alone;
# - tokens/ is SoftHSM2's, with a token for each YubiKey plugged in, labelled YubiKey PIV #SERIAL
#   as libykcs11 labels its token, with the user PIN VERIFY_NONE. Each key has libykcs11's ID and
#   label, 05 and Retired Key 1 for slot 82;
# - env names SoftHSM2's softhsm2-util and an openssl that makes Ed25519 keys, which macOS's lacks.

set -eu

S=${RUNNER_TEMP:?the stand-in YubiKey lives in RUNNER_TEMP}/yubikey

# load sets softhsm and openssl to the programs that env names.
load() {
    softhsm=$(sed -n 's/^softhsm=//p' "$S/env")
    openssl=$(sed -n 's/^openssl=//p' "$S/env")
}

# token SERIAL makes the YubiKey's token again from its slots' keys while it is plugged in, and
# deletes it while it is out, so that the PKCS#11 module lists its keys only while it is in.
token() {
    label="YubiKey PIV #$1"
    "$softhsm" --delete-token --token "$label" >/dev/null 2>&1 || true
    [ -f "$S/$1/in" ] || return 0
    "$softhsm" --init-token --free --label "$label" --so-pin 12345678 --pin VERIFY_NONE >/dev/null
    for dir in "$S/$1"/[89]?; do
        [ -f "$dir/key.pem" ] || continue
        slot=${dir##*/}
        n=$(($(printf '%d' "0x$slot") - 129))
        "$softhsm" --import "$dir/key.pem" --token "$label" --label "Retired Key $n" \
            --id "$(printf '%02x' $((n + 4)))" --pin VERIFY_NONE >/dev/null
    done
}

# keygen SERIAL SLOT PIN TOUCH makes an Ed25519 key in the slot, with its policies.
keygen() {
    dir=$S/$1/$2
    mkdir -p "$dir"
    (umask 077 && "$openssl" genpkey -algorithm ed25519 -out "$dir/key.pem")
    "$openssl" pkey -in "$dir/key.pem" -pubout -outform DER -out "$dir/pub.der"
    echo "$3 $4" >"$dir/policy"
}

# setup installs SoftHSM2 and yubico-piv-tool, points libykcs11 and SoftHSM2's config at the
# stand-in's, and makes the job's YubiKey.
setup() {
    for formula in softhsm yubico-piv-tool; do
        brew list --formula "$formula" >/dev/null 2>&1 || brew install "$formula"
    done
    prefix=$(brew --prefix)
    module=$(brew --prefix softhsm)/lib/softhsm/libsofthsm2.so
    [ -f "$module" ] || { echo "::error::softhsm has no $module" && exit 1; }
    for openssl in "$prefix"/opt/openssl*/bin/openssl "$(command -v openssl)"; do
        ! "$openssl" genpkey -algorithm ed25519 >/dev/null 2>&1 || break
    done
    "$openssl" genpkey -algorithm ed25519 >/dev/null 2>&1 ||
        { echo "::error::no openssl of Homebrew's makes an Ed25519 key" && exit 1; }
    mkdir -p "$S/tokens"
    printf 'softhsm=%s\nopenssl=%s\n' "$(brew --prefix softhsm)/bin/softhsm2-util" "$openssl" \
        >"$S/env"
    conf=${SOFTHSM2_CONF:-$prefix/etc/softhsm/softhsm2.conf}
    mkdir -p "${conf%/*}"
    printf '%s\n' "directories.tokendir = $S/tokens" 'objectstore.backend = file' \
        'log.level = ERROR' >"$conf"
    ln -sfn "$module" "$(brew --prefix yubico-piv-tool)/lib/libykcs11.dylib"
    load
    change new 30000001
    change access 30000001
    change key 30000001 82 ws NEVER NEVER
    change key 30000001 83 git NEVER NEVER
    "$softhsm" --show-slots | grep -q 'YubiKey PIV #30000001' ||
        { echo "::error::SoftHSM2 lists no token of the stand-in's, as $conf names" && exit 1; }
}

# change NAME SERIAL ARG... runs a case's command NAME on the stand-in's YubiKey SERIAL.
change() {
    cmd=$1
    serial=$2
    shift 2
    case $cmd in
    new)
        mkdir -p "$S/$serial"
        echo 5.7.1 >"$S/$serial/version"
        touch "$S/$serial/in"
        ;;
    access) touch "$S/$serial/access" ;;
    key)
        keygen "$serial" "$1" "$3" "$4"
        rm -f "$S/$serial/$1/cn"
        [ "$2" = - ] || echo "$2" >"$S/$serial/$1/cn"
        ;;
    drop) rm -rf "${S:?}/$serial/$1" ;;
    plug) touch "$S/$serial/in" ;;
    unplug) rm -f "$S/$serial/in" ;;
    *) echo "yubikey-standin.sh: no command $cmd" >&2 && exit 2 ;;
    esac
    token "$serial"
}

if [ $# = 0 ]; then
    setup
else
    load
    change "$@"
fi
