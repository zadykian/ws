# shellcheck shell=sh
# The YubiKey's step of mac/setup.sh: the one YubiKey in, whose PIV applet holds the keys that
# mac.conf's WS_SSH_KEY and WS_GIT_KEY name. ykman, which reads them, and yubico-piv-tool, whose
# libykcs11 the agent loads, come from Homebrew. setup.sh sources it after lib.sh and
# setup-lib.sh, whose functions it calls, and runs yubikey before ssh's hosts. The wizard can call
# yubikey_tools, yubikey_serials and yubikey_keys, each with its stage's name for their messages.

# yubikey runs the step: the tools, then the keys.
yubikey() {
    yubikey_tools
    yubikey_keys
}

# yubikey_tools [STAGE] installs Homebrew's ykman where PATH has none, and Homebrew's
# yubico-piv-tool where Homebrew lacks it, as the agent loads its libykcs11 from Homebrew's Cellar.
# It sets yubikey_ykman to ykman's path, which --check leaves empty before the install. Its
# messages start with STAGE, where the wizard's stage gives one.
# shellcheck disable=SC2120 # the wizard's stage passes its name
yubikey_tools() {
    yubikey_at=${1:+$1: }
    brew=$(command -v brew) || ws_die "${yubikey_at}install Homebrew first: https://brew.sh"
    yubikey_need=
    command -v ykman >/dev/null || yubikey_need=ykman
    "$brew" list --formula yubico-piv-tool >/dev/null 2>&1 ||
        yubikey_need="$yubikey_need${yubikey_need:+ }yubico-piv-tool"
    if [ -n "$yubikey_need" ] && ws_checking; then
        ws_say "${yubikey_at}would run brew install $yubikey_need"
    elif [ -n "$yubikey_need" ]; then
        ws_say "${yubikey_at}installing $yubikey_need"
        # shellcheck disable=SC2086 # the formulae's names, split bare
        "$brew" install $yubikey_need >&2 || ws_die "${yubikey_at}brew install $yubikey_need failed"
    fi
    yubikey_ykman=$(command -v ykman) || yubikey_ykman=
    ws_checking || [ -n "$yubikey_ykman" ] ||
        ws_die "${yubikey_at}brew install ykman left no ykman on PATH"
}

# yubikey_serials prints the serials of the YubiKeys in, a line each, and nothing where none is in.
# It leaves out a YubiKey that hides its serial, as ykman list --serials does, and prints nothing
# under --check before yubikey_tools installs ykman.
yubikey_serials() {
    [ -n "$yubikey_ykman" ] || return 0
    "$yubikey_ykman" list --serials
}

# yubikey_keys [STAGE] finds the one YubiKey in, which holds the keys of WS_SSH_KEY and WS_GIT_KEY,
# and warns where its PIV applet keeps a default. It stops unless one YubiKey is in. Under --check
# before yubikey_tools installs ykman, it says what it would do once it has. Its messages start
# with STAGE, where the wizard's stage gives one.
# shellcheck disable=SC2120 # the wizard's stage passes its name
yubikey_keys() {
    yubikey_at=${1:+$1: }
    if [ -z "$yubikey_ykman" ]; then
        ws_say "${yubikey_at}would find the keys $WS_SSH_KEY and $WS_GIT_KEY on the YubiKey" \
            "once ykman is installed"
        return 0
    fi
    yubikey_one
    yubikey_defaults
}

# yubikey_one sets yubikey_serial to the serial of the one YubiKey in, which each later call of
# ykman names by --device, so that one plugged in meanwhile changes nothing. It stops on none, and
# on several.
yubikey_one() {
    yubikey_in=$(yubikey_serials) || ws_die "${yubikey_at}ykman list --serials failed"
    case $yubikey_in in
    '') ws_die "${yubikey_at}no YubiKey is in: plug in the one with ws's keys, or a new one" ;;
    *[!0-9]*)
        ws_die "${yubikey_at}several YubiKeys are in, $(yubikey_join "$yubikey_in"):" \
            "leave in the one that holds ws's keys alone"
        ;;
    esac
    yubikey_serial=$yubikey_in
    yubikey_read
}

# yubikey_read sets yubikey_piv to what ykman piv info prints of the YubiKey's PIV applet, which
# ykman 5.9 prints as text alone.
yubikey_read() {
    yubikey_piv=$("$yubikey_ykman" --device "$yubikey_serial" piv info) ||
        ws_die "${yubikey_at}ykman --device $yubikey_serial piv info failed"
}

# yubikey_defaults warns where ykman piv info reports the PIV applet's default PIN, PUK or
# management key, with which any program on the Mac could replace or delete the keys. It changes
# none, as a PUK that setup.sh chose would be lost to the user.
yubikey_defaults() {
    yubikey_list=$(printf '%s\n' "$yubikey_piv" | sed -n \
        -e 's/^WARNING: Using default PIN!$/PIN/p' \
        -e 's/^WARNING: Using default PUK!$/PUK/p' \
        -e 's/^WARNING: Using default Management key!$/management key/p')
    yubikey_list=$(yubikey_join "$yubikey_list")
    [ -z "$yubikey_list" ] || ws_say "${yubikey_at}warning: YubiKey $yubikey_serial keeps the PIV" \
        "applet's default $yubikey_list: ykman piv access changes them, as docs/mac-yubikey.md says"
}

# yubikey_join LINES prints LINES joined by commas, and the last two by and.
yubikey_join() {
    printf '%s\n' "$1" | awk '{ item[NR] = $0 }
        END {
            for (i = 1; i <= NR; i++)
                out = out (i == 1 ? "" : i == NR ? " and " : ", ") item[i]
            print out
        }'
}
