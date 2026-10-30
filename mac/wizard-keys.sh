# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's Mac key stage: the key that ssh offers ws, which setup-identity.sh's ws_identity
# finds from mac.conf, made where missing. wizard.sh sources it, and runs the stage after Answers
# and before gh on the Mac, which adds the key to GitHub. Without WS_IDENTITY, the keys are the
# YubiKey's, which the stage finds or makes as setup.sh does, and ws's agent loads. It calls the
# functions of lib.sh, setup-lib.sh and wizard-ask.sh, and reads wizard.sh's lock and WS_CONF,
# mac.conf's path.

# The YubiKey's step and the agent's are setup.sh's, sourced from dir, mac/'s path, as wizard.sh
# sets it.
# shellcheck source=setup-yubikey.sh
. "${dir:?wizard.sh sets dir}/setup-yubikey.sh"
# shellcheck source=setup-agent.sh
. "$dir/setup-agent.sh"

# keys_done succeeds where the key that ssh offers ws is there: in a file beside the .pub whose
# fingerprint it shares, or listed by the agent that holds it. Where it is Secretive's, git, which
# has a name, needs a key of its own there too, unless mac.conf has a line for WS_GIT_SIGNING_KEY.
# Without WS_IDENTITY, ws's agent must list git's key of the YubiKey's too.
# shellcheck disable=SC2154 # yubikey_dir is setup-yubikey.sh's
keys_done() {
    (ws_conf "$WS_CONF" && ws_identity && keys_there && ! keys_signing &&
        { [ -n "$WS_IDENTITY" ] || keys_there "$yubikey_dir/$WS_GIT_KEY.pub"; }) >/dev/null 2>&1
}

# keys_there [PUB] succeeds where the key that ws_identity set, or PUB in its agent, is there: the
# agent lists the .pub's type and base64, or the file beside the .pub is its private key.
# shellcheck disable=SC2154 # ws_identity sets ws_identity_pub and ws_identity_agent
keys_there() {
    keys_pub=$(keys_home "${1:-$ws_identity_pub}")
    if [ -z "$ws_identity_agent" ]; then
        keys_same "${keys_pub%.pub}"
        return
    fi
    [ -f "$keys_pub" ] || return 1
    keys_line=$(awk 'NF { print $1, $2; exit }' "$keys_pub")
    SSH_AUTH_SOCK=$(keys_home "$ws_identity_agent") ssh-add -L 2>/dev/null |
        awk '{ print $1, $2 }' | grep -qxF -e "$keys_line"
}

# keys_same KEY succeeds where KEY and KEY.pub are there and share a fingerprint. ssh-keygen reads
# KEY by a link in the lock, with no .pub beside it: by KEY's name, it would read KEY.pub instead.
# On macOS, it can't reread KEY from /dev/stdin, whose opens share one offset.
# shellcheck disable=SC2154 # lock is wizard.sh's
keys_same() {
    [ -f "$1" ] && [ -f "$1.pub" ] && ln -sf "$1" "$lock/key" &&
        keys_fp=$(ssh-keygen -l -f "$lock/key" 2>/dev/null | awk '{ print $2 }') &&
        rm -f "$lock/key" && [ -n "$keys_fp" ] &&
        [ "$keys_fp" = "$(ssh-keygen -l -f "$1.pub" 2>/dev/null | awk '{ print $2 }')" ]
}

# keys_home PATH prints PATH with a ~/ first made the home's path.
keys_home() {
    case $1 in
    \~/*) echo "$HOME/${1#\~/}" ;;
    *) echo "$1" ;;
    esac
}

# keys_signing succeeds where ws's key is Secretive's, and git, which has a name, lacks a key of its
# own there, as mac.conf has no line for WS_GIT_SIGNING_KEY. An empty one keeps setup.sh's key.
# shellcheck disable=SC2154 # ws_secretive is setup-conf.sh's
keys_signing() {
    [ "$(keys_home "$ws_identity_agent")" = "$ws_secretive/socket.ssh" ] &&
        [ -n "$WS_GIT_NAME" ] && ask_lacks WS_GIT_SIGNING_KEY
}

# keys_do makes the key that keys_done lacks, or says why it can't. Without WS_IDENTITY, it runs
# setup.sh's YubiKey and agent steps, which find or make the YubiKey's keys and have ws's agent load
# them, or stop. A WS_IDENTITY outside Secretive's PublicKeys, with no agent, is a key in a file,
# which keys_file makes. Else ws_identity names its agent, which must list it. git's key in
# Secretive follows ws's.
keys_do() {
    ws_conf "$WS_CONF"
    if [ -z "$WS_IDENTITY" ]; then
        yubikey_tools "Mac key"
        yubikey_keys "Mac key"
        agent
        return
    fi
    keys_pub=$(keys_home "$WS_IDENTITY")
    if [ -n "$keys_pub" ] && [ -z "$WS_IDENTITY_AGENT" ] &&
        [ "${keys_pub#"$ws_secretive"/PublicKeys/}" = "$keys_pub" ]; then
        keys_file "${keys_pub%.pub}"
        return
    fi
    ws_identity
    keys_pub=$(keys_home "$ws_identity_pub")
    keys_there || ws_die "Mac key: the agent at $(keys_home "$ws_identity_agent") lists no key" \
        "of $keys_pub: open Secretive, where it is that agent, or start the agent, then run the" \
        "wizard again"
    ! keys_signing || keys_git
}

# keys_file KEY makes KEY, a key in a file, where neither it nor KEY.pub is there. Else it stops
# where one is missing, or the two differ, saying how to mend that, and runs ws_identity's checks.
keys_file() {
    if [ ! -e "$1" ] && [ ! -e "$1.pub" ]; then
        keys_make "$1"
        return
    fi
    [ -e "$1" ] || ws_die "Mac key: $1.pub is there without its private key, $1: remove it, or" \
        "name the agent that holds the key in mac.conf's WS_IDENTITY_AGENT"
    keys_same "$1" || ws_die "Mac key: $1.pub is not $1's public key: ssh-keygen -y -f $1" \
        ">$1.pub writes it"
    ws_identity
}

# keys_make KEY makes the ed25519 key KEY, whose passphrase ssh-keygen asks at the terminal. It
# gives the key to ssh's agent, and the passphrase to the Keychain, so that ssh asks no more.
# Without the agent, ssh asks at the first login instead, and the Keychain keeps it then.
keys_make() {
    if ws_checking; then
        ws_say "Mac key: would make $1 with ssh-keygen -t ed25519 -a 100, then run ssh-add" \
            "--apple-use-keychain $1"
        return 0
    fi
    ws_say "Mac key: making $1: ssh-keygen asks its passphrase twice, then ssh-add once"
    ssh-keygen -q -t ed25519 -a 100 -f "$1" >&2 || ws_die "Mac key: ssh-keygen cannot make $1"
    ssh-add --apple-use-keychain "$1" >&2 || ws_say "Mac key: warning: ssh-add cannot give $1 to" \
        "the agent, so ssh asks its passphrase at the first login, and the Keychain keeps it then"
}

# keys_git has the user make git's signing key in Secretive's app, beside ws's, at Notify, so that
# git signs with no prompt while the Mac is unlocked (#109). It writes its .pub's path into
# mac.conf as WS_GIT_SIGNING_KEY.
keys_git() {
    if ws_checking; then
        ws_say "Mac key: would open Secretive, and wait for git's signing key made there"
        return 0
    fi
    open -a Secretive || ws_say "Mac key: warning: open cannot find Secretive: open it by hand"
    keys_skip=$keys_pub
    keys_pick "a second P-256 key, for git's signing, at Notify" \
        "An empty WS_GIT_SIGNING_KEY= in mac.conf keeps setup.sh's file key for git at the next run"
    keys_signer=\~/${keys_got#"$HOME"/}
    ask_write "$(ask_line WS_GIT_SIGNING_KEY "$keys_signer")
" "Mac key"
}

# keys_pick WHAT [ELSE] sets keys_got to the .pub in PublicKeys other than keys_skip's. Where none
# is there, it says to make WHAT in Secretive's app, adds ELSE, and waits up to 15 minutes for one.
# It stops on a key other than P-256, as an ML-DSA one, which ws's OpenSSH lacks.
keys_pick() {
    if ! keys_other; then
        ws_say "Mac key: in Secretive, make $1; the wizard waits up to 15 minutes for it${2:+. $2}"
        keys_left=450
        until keys_other; do
            [ "$((keys_left -= 1))" -ge 0 ] || ws_die "Mac key: Secretive made no key in 15" \
                "minutes: make $1, then run the wizard again${2:+. $2}"
            sleep 2
        done
    fi
    [ "$(awk 'NF { print $1; exit }' "$keys_got")" = ecdsa-sha2-nistp256 ] || ws_die "Mac key:" \
        "$keys_got is not the P-256 key asked for: remove it in Secretive, make $1, then run" \
        "the wizard again"
}

# keys_other succeeds where PublicKeys holds a .pub other than keys_skip's, as Secretive writes one
# for each key, and sets keys_got to it. It stops where several are there.
keys_other() {
    keys_got=
    for keys_each in "$ws_secretive"/PublicKeys/*.pub; do
        case $keys_each in "$keys_skip" | *-cert.pub) continue ;; esac
        [ -f "$keys_each" ] || continue
        [ -z "$keys_got" ] || ws_die "Mac key: Secretive holds several keys, $keys_got and" \
            "$keys_each: name ws's in mac.conf's WS_IDENTITY, and git's in WS_GIT_SIGNING_KEY"
        keys_got=$keys_each
    done
    [ -n "$keys_got" ]
}

# keys_fail says what to do where the Mac's key is still missing after keys_do.
keys_fail() {
    echo "ssh has no usable key for ws, as above; mend that, then run the wizard again"
}
