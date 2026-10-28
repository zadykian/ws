# shellcheck shell=sh
# The YubiKey's step of mac/setup.sh: the keys that mac.conf's WS_SSH_KEY and WS_GIT_KEY name,
# found on the YubiKey's PIV applet, made where missing, and their .pubs in ~/.config/ws/keys.
# ykman, which reads and makes them, yubico-piv-tool, whose libykcs11 the agent loads, and
# terminal-notifier, for the agent's notices, come from Homebrew. setup.sh sources it after lib.sh
# and setup-lib.sh, whose functions it calls, and runs yubikey before ssh's hosts. The wizard can
# call yubikey_tools, yubikey_serials and yubikey_keys, each with its stage's name for messages.

# The directory of the keys' .pubs, NAME.pub for each, which ssh and git name.
yubikey_dir=$HOME/.config/ws/keys
# The retired slots, where a key is made, in ykman's order.
yubikey_retired='82 83 84 85 86 87 88 89 8A 8B 8C 8D 8E 8F 90 91 92 93 94 95'

# yubikey runs the step: the tools, then the keys.
yubikey() {
    yubikey_tools
    yubikey_keys
}

# yubikey_tools [STAGE] installs Homebrew's ykman where PATH has none, and Homebrew's
# yubico-piv-tool where Homebrew lacks it, as the agent loads its libykcs11 from Homebrew's Cellar.
# Then yubikey_notifier. It sets yubikey_ykman to ykman's path, which --check leaves empty before
# the install. Its messages start with STAGE, where the wizard's stage gives one.
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
    yubikey_notifier
    yubikey_ykman=$(command -v ykman) || yubikey_ykman=
    ws_checking || [ -n "$yubikey_ykman" ] ||
        ws_die "${yubikey_at}brew install ykman left no ykman on PATH"
}

# yubikey_notifier installs Homebrew's terminal-notifier where Homebrew lacks it, as
# ws-agent-confirm runs it from launchd's PATH or /opt/homebrew/bin. A failed install warns alone:
# it costs the agent's notices, not its signatures, and the next run tries again.
yubikey_notifier() {
    "$brew" list --formula terminal-notifier >/dev/null 2>&1 && return 0
    ws_checking && { ws_say "${yubikey_at}would run brew install terminal-notifier" && return 0; }
    ws_say "${yubikey_at}installing terminal-notifier"
    "$brew" install terminal-notifier >&2 || ws_say "${yubikey_at}warning: brew install" \
        "terminal-notifier failed, so ws's agent posts no notice: the next run tries again"
}

# yubikey_serials prints the serials of the YubiKeys in, a line each, and nothing where none is in.
# It leaves out a YubiKey that hides its serial, as ykman list --serials does, and prints nothing
# under --check before yubikey_tools installs ykman.
yubikey_serials() {
    [ -n "$yubikey_ykman" ] || return 0
    "$yubikey_ykman" list --serials
}

# yubikey_keys [STAGE] finds the keys of WS_SSH_KEY and WS_GIT_KEY on the one YubiKey in, makes
# those missing, checks them, and writes their .pubs. It stops unless one YubiKey is in. Under
# --check before yubikey_tools installs ykman, it says what it would do once it has. Its messages
# start with STAGE, where the wizard's stage gives one.
# shellcheck disable=SC2120 # the wizard's stage passes its name
yubikey_keys() {
    yubikey_at=${1:+$1: }
    if [ -z "$yubikey_ykman" ]; then
        ws_say "${yubikey_at}would find or make the keys $WS_SSH_KEY and $WS_GIT_KEY on the" \
            "YubiKey once ykman is installed"
        return 0
    fi
    yubikey_one
    yubikey_defaults
    [ -d "${yubikey_dir%/*}" ] || ws_dir "${yubikey_dir%/*}" 755
    ws_dir "$yubikey_dir" 755
    yubikey_taken=
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
# makes it in the first free retired slot where none is. It stops where several slots hold that
# certificate. It checks the key, and writes its .pub.
yubikey_key() {
    yubikey_slot=$(yubikey_slots | awk -v want="CN=$1" '
        { subject = $0; sub(/^[^ ]+ [^ ]+ /, "", subject) }
        subject == want { print $1 }')
    case $yubikey_slot in
    *[!0-9A-F]*)
        ws_die "${yubikey_at}slots $(yubikey_join "$yubikey_slot") each hold the" \
            "certificate CN=$1: ykman piv certificates delete SLOT drops the name from all but one"
        ;;
    esac
    if [ -z "$yubikey_slot" ]; then
        yubikey_free "$1"
        yubikey_make "$1" || return 0
        yubikey_read
    fi
    yubikey_check "$1"
    yubikey_pub "$1"
}

# yubikey_free NAME sets yubikey_slot to the first retired slot that holds neither a key nor a
# certificate, and that no key yubikey_make would make under --check took. It stops where none is.
yubikey_free() {
    yubikey_used=" $(yubikey_slots | awk '{ printf "%s ", $1 }')$yubikey_taken "
    for yubikey_slot in $yubikey_retired; do
        case $yubikey_used in *" $yubikey_slot "*) continue ;; esac
        yubikey_taken="$yubikey_taken $yubikey_slot "
        return 0
    done
    ws_die "${yubikey_at}YubiKey $yubikey_serial has no free slot of 82 to 95 for $1's key:" \
        "ykman piv keys delete SLOT and ykman piv certificates delete SLOT free one"
}

# yubikey_make NAME makes NAME's key in yubikey_slot: an Ed25519 key, used with no PIN and no touch,
# as ws's agent and git ask neither, then its certificate CN=NAME, which names it, valid for 3650
# days. ssh and git use the key alone and ignore the expiry, but other readers of the PIV
# certificate, such as macOS's smart-card pairing or Yubico Authenticator, show an expired one, and
# 10 years outlast the key. ykman asks the management key at each, and the PIN for the certificate,
# at the terminal: the PIN alone where it protects the management key. With none, it stops, naming
# both commands. Ed25519 needs firmware 5.7 or later. --check asks nothing, says what it would run,
# and fails, as the key is not there.
# shellcheck disable=SC2154 # ws_prog is lib.sh's, and ws_tmp, lib.sh's, sets ws_tmpfile
yubikey_make() {
    yubikey_version=$(printf '%s\n' "$yubikey_piv" |
        awk '$1 == "PIV" && $2 == "version:" { print $3 }')
    case $yubikey_version in
    [0-4].* | 5.[0-6].*)
        ws_die "${yubikey_at}YubiKey $yubikey_serial's firmware, $yubikey_version, makes no" \
            "Ed25519 key, which came in 5.7: plug in a YubiKey of firmware 5.7 or later"
        ;;
    esac
    yubikey_run="ykman piv keys generate -a ED25519 --pin-policy NEVER --touch-policy NEVER"
    yubikey_run="$yubikey_run $yubikey_slot $1.pem, then ykman piv certificates generate -s"
    yubikey_run="$yubikey_run CN=$1 -d 3650 $yubikey_slot $1.pem"
    if ws_checking; then
        ws_say "${yubikey_at}would make $1's key in slot $yubikey_slot of YubiKey" \
            "$yubikey_serial: $yubikey_run, which ask the PIN, and the management key unless the" \
            "PIN protects it"
        ws_say "${yubikey_at}would write $yubikey_dir/$1.pub once the key is made"
        return 1
    fi
    [ -t 0 ] || ws_die "${yubikey_at}no terminal for ykman to make $1's key, as it asks the PIN," \
        "and the management key unless the PIN protects it: run $ws_prog at a terminal, or make" \
        "the key with $yubikey_run"
    ws_say "${yubikey_at}making $1's key in slot $yubikey_slot of YubiKey $yubikey_serial: ykman" \
        "asks the management key, then it and the PIN again, or the PIN alone where it protects" \
        "the management key"
    ws_tmp
    yubikey_pem=$ws_tmpfile
    if ! "$yubikey_ykman" --device "$yubikey_serial" piv keys generate -a ED25519 \
        --pin-policy NEVER --touch-policy NEVER "$yubikey_slot" "$yubikey_pem" >&2; then
        rm -f "$yubikey_pem"
        ws_die "${yubikey_at}ykman piv keys generate failed for $1's key"
    fi
    if ! "$yubikey_ykman" --device "$yubikey_serial" piv certificates generate -s "CN=$1" \
        -d 3650 "$yubikey_slot" "$yubikey_pem" >&2; then
        rm -f "$yubikey_pem"
        ws_die "${yubikey_at}ykman piv certificates generate failed, which leaves a key with no" \
            "name in slot $yubikey_slot: ykman --device $yubikey_serial piv certificates generate" \
            "-s CN=$1 -d 3650 $yubikey_slot names it, or ykman piv keys delete $yubikey_slot" \
            "frees the slot"
    fi
    rm -f "$yubikey_pem"
}

# yubikey_check NAME stops where NAME's key, in yubikey_slot, is missing beside its certificate, is
# not Ed25519, or has a PIN or touch policy, as ykman piv keys info shows: ws's agent loads it with
# no PIN, and Root's login asks nothing (mac/wizard-host.sh). The Mac's OpenSSH loads Ed25519 keys
# through PKCS#11, and ws makes and tests no other.
yubikey_check() {
    yubikey_again="ykman piv keys delete $yubikey_slot and ykman piv certificates delete"
    yubikey_again="$yubikey_again $yubikey_slot free the slot, and the next run makes the key again"
    yubikey_type=$(yubikey_slots | awk -v slot="$yubikey_slot" '$1 == slot { print $2 }')
    case $yubikey_type in
    ED25519) ;;
    EMPTY)
        ws_die "${yubikey_at}slot $yubikey_slot holds the certificate CN=$1 without its key:" \
            "ykman piv certificates delete $yubikey_slot frees the name, and the next run makes" \
            "the key"
        ;;
    *)
        ws_die "${yubikey_at}$1's key, in slot $yubikey_slot, is $yubikey_type, not ED25519:" \
            "$yubikey_again"
        ;;
    esac
    yubikey_info=$("$yubikey_ykman" --device "$yubikey_serial" piv keys info "$yubikey_slot") ||
        ws_die "${yubikey_at}ykman piv keys info $yubikey_slot failed"
    yubikey_policy=$(printf '%s\n' "$yubikey_info" | awk -F ': +' '
        $1 == "PIN required for use" { pin = $2 }
        $1 == "Touch required for use" { touch = $2 }
        END { print pin " " touch }')
    [ "$yubikey_policy" = 'NEVER NEVER' ] || ws_die "${yubikey_at}$1's key, in slot" \
        "$yubikey_slot, asks for the PIN or a touch at each use, with PIN policy" \
        "${yubikey_policy% *} and touch policy ${yubikey_policy#* }: $yubikey_again"
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
