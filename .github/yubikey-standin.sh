#!/bin/sh
# A stand-in YubiKey 5C Nano for the mac and wizard jobs, on GitHub's macOS runner. Run by sh, it
# sets the stand-in up, or changes its YubiKeys for a case:
#
#   sh .github/yubikey-standin.sh                       the job's YubiKey, with ws's and git's keys
#   sh .github/yubikey-standin.sh new SERIAL            a YubiKey from the factory, plugged in
#   sh .github/yubikey-standin.sh access SERIAL         its PIN, PUK and management key changed
#   sh .github/yubikey-standin.sh key SERIAL SLOT NAME PIN TOUCH   a key, as ykman makes one
#   sh .github/yubikey-standin.sh drop SERIAL SLOT      a slot's key and certificate gone
#   sh .github/yubikey-standin.sh plug|unplug SERIAL    a YubiKey in or out
#
# The job's YubiKey, 30000001, has firmware 5.7.1, no default PIN, PUK or management key, and the
# keys ws and git in slots 82 and 83, of policy NEVER NEVER.
#
# Its state is in S, $RUNNER_TEMP/yubikey:
#
# - SERIAL/ is a YubiKey: version holds its firmware, in is there while it is plugged in, and access
#   once its PIN, PUK and management key are no longer the defaults;
# - SERIAL/SLOT/ is a slot, 82 to 95: key.pem holds its private key, pub.der its public key, policy
#   its PIN and touch policies, as NEVER NEVER, and cn its certificate's CN. A slot with cn and no
#   pub.der holds a certificate alone;
# - env names an openssl that makes Ed25519 keys, which macOS's lacks.

set -eu

S=${RUNNER_TEMP:?the stand-in YubiKey lives in RUNNER_TEMP}/yubikey

# load sets openssl to the program that env names.
load() {
    openssl=$(sed -n 's/^openssl=//p' "$S/env")
}

# keygen SERIAL SLOT PIN TOUCH makes an Ed25519 key in the slot, with its policies.
keygen() {
    dir=$S/$1/$2
    mkdir -p "$dir"
    (umask 077 && "$openssl" genpkey -algorithm ed25519 -out "$dir/key.pem")
    "$openssl" pkey -in "$dir/key.pem" -pubout -outform DER -out "$dir/pub.der"
    echo "$3 $4" >"$dir/policy"
}

# setup makes the job's YubiKey.
setup() {
    prefix=$(brew --prefix)
    for openssl in "$prefix"/opt/openssl*/bin/openssl "$(command -v openssl)"; do
        ! "$openssl" genpkey -algorithm ed25519 >/dev/null 2>&1 || break
    done
    "$openssl" genpkey -algorithm ed25519 >/dev/null 2>&1 ||
        { echo "::error::no openssl of Homebrew's makes an Ed25519 key" && exit 1; }
    mkdir -p "$S"
    printf 'openssl=%s\n' "$openssl" >"$S/env"
    load
    change new 30000001
    change access 30000001
    change key 30000001 82 ws NEVER NEVER
    change key 30000001 83 git NEVER NEVER
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
}

if [ $# = 0 ]; then
    setup
else
    load
    change "$@"
fi
