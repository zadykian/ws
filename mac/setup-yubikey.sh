# shellcheck shell=sh
# The YubiKey's step of mac/setup.sh: the keys that mac.conf's WS_SSH_KEY and WS_GIT_KEY name,
# found on the YubiKey's PIV applet, and their .pubs in ~/.config/ws/keys. ykman, which reads
# them, and yubico-piv-tool, whose libykcs11 the agent loads, come from Homebrew. setup.sh sources
# it after lib.sh and setup-lib.sh, whose functions it calls, and runs yubikey before ssh's hosts.
# The wizard can call yubikey_tools, yubikey_serials and yubikey_keys, each with its stage's name
# for their messages.

# The directory of the keys' .pubs, NAME.pub for each, which ssh and git name.
yubikey_dir=$HOME/.config/ws/keys

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

# yubikey_keys [STAGE] finds the keys of WS_SSH_KEY and WS_GIT_KEY on the one YubiKey in, and
# writes their .pubs. It stops unless one YubiKey is in. Under --check before yubikey_tools installs
# ykman, it says what it would do once it has. Its messages start with STAGE, where the wizard's
# stage gives one.
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
    [ -d "${yubikey_dir%/*}" ] || ws_dir "${yubikey_dir%/*}" 755
    ws_dir "$yubikey_dir" 755
    yubikey_key "$WS_SSH_KEY"
    yubikey_key "$WS_GIT_KEY"
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

# yubikey_slots prints a line for each slot that ykman piv info lists, as it lists those that hold a
# key or a certificate: SLOT KEY SUBJECT, as 82 ED25519 CN=ws. KEY is EMPTY for a certificate
# alone, and SUBJECT - for a key alone.
yubikey_slots() {
    printf '%s\n' "$yubikey_piv" | awk '
        function out() { if (slot != "") print slot, key, subject }
        /^Slot [0-9A-F][0-9A-F] \(/ { out(); slot = $2; key = "EMPTY"; subject = "-"; next }
        /^  Private key type: / { key = $4 }
        /^  Subject DN: / { subject = substr($0, index($0, ":") + 1); sub(/^ +/, "", subject) }
        END { out() }'
}

# yubikey_key NAME sets yubikey_slot to the slot of the key NAME, whose certificate is CN=NAME, and
# stops where none is. It writes the key's .pub.
yubikey_key() {
    yubikey_slot=$(yubikey_slots | awk -v want="CN=$1" '
        { subject = $0; sub(/^[^ ]+ [^ ]+ /, "", subject) }
        subject == want { print $1 }')
    [ -n "$yubikey_slot" ] || ws_die "${yubikey_at}YubiKey $yubikey_serial holds no key $1, whose" \
        "certificate is CN=$1: make it as docs/mac-yubikey.md says"
    yubikey_pub "$1"
}

# yubikey_pub NAME writes yubikey_dir/NAME.pub, the public key of NAME's key, in yubikey_slot. It
# warns where the file held another key, as of another YubiKey or a reset PIV applet, which GitHub
# and ws lack.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
yubikey_pub() {
    ws_tmp
    yubikey_der=$ws_tmpfile
    if ! "$yubikey_ykman" --device "$yubikey_serial" piv keys export -F DER "$yubikey_slot" \
        "$yubikey_der" >/dev/null || ! yubikey_line=$(yubikey_openssh "$yubikey_der"); then
        rm -f "$yubikey_der"
        ws_die "${yubikey_at}ykman piv keys export $yubikey_slot gave no Ed25519 key" \
            "for $1"
    fi
    rm -f "$yubikey_der"
    yubikey_file=$yubikey_dir/$1.pub
    yubikey_old=$(awk 'NF { print $1, $2; exit }' "$yubikey_file" 2>/dev/null) || yubikey_old=
    if [ -n "$yubikey_old" ] && [ "$yubikey_old" != "$yubikey_line" ]; then
        ws_say "${yubikey_at}warning: $yubikey_file held another key, of another YubiKey or a" \
            "reset PIV applet: GitHub and ws need the new one"
    fi
    ws_put "$yubikey_file" 644 <<EOF
$yubikey_line $1@yubikey
EOF
}

# yubikey_openssh DER prints the OpenSSH public key, its type and base64, of DER, a key's
# SubjectPublicKeyInfo: an Ed25519 key's 12 bytes of header and its 32. It fails on any other.
# ssh-keygen -i takes no Ed25519 key, in OpenSSH 10.3 too, so the key's blob is made here: each of
# its strings after its length in 4 bytes.
yubikey_openssh() {
    yubikey_form=$(wc -c <"$1" | tr -d ' '):$(od -An -tx1 -N12 "$1" | tr -d ' \n')
    case $yubikey_form in
    44:302a300506032b6570032100)
        printf 'ssh-ed25519 %s\n' "$({ printf '\000\000\000\013ssh-ed25519\000\000\000\040' &&
            tail -c 32 "$1"; } | base64 | tr -d '\n')"
        ;;
    *) return 1 ;;
    esac
}
