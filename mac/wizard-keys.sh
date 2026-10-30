# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's Mac key stage: the key that ssh offers ws, which setup-identity.sh's ws_identity
# finds from mac.conf, made where missing. wizard.sh sources it, and runs the stage after Answers
# and before gh on the Mac, which adds the key to GitHub. Without WS_IDENTITY, the keys are the
# YubiKey's, which the stage finds or makes as setup.sh does, and ws's agent loads. git's key is the
# YubiKey's too. It calls the functions of lib.sh and setup-lib.sh, and reads wizard.sh's lock and
# WS_CONF, mac.conf's path.

# The YubiKey's step and the agent's are setup.sh's, sourced from dir, mac/'s path, as wizard.sh
# sets it.
# shellcheck source=setup-yubikey.sh
. "${dir:?wizard.sh sets dir}/setup-yubikey.sh"
# shellcheck source=setup-agent.sh
. "$dir/setup-agent.sh"

# keys_after prints Tools, bootstrap.sh's stage, where the keys are the YubiKey's, as mac.conf
# names no WS_IDENTITY, and PATH has no brew to install ykman.
keys_after() {
    command -v brew >/dev/null || (ws_conf "$WS_CONF" && [ -n "$WS_IDENTITY" ]) >/dev/null 2>&1 ||
        echo Tools
}

# keys_done succeeds where the key that ssh offers ws is there: in a file beside the .pub whose
# fingerprint it shares, or listed by the agent that holds it. Without WS_IDENTITY, ws's agent must
# list git's key of the YubiKey's too.
# shellcheck disable=SC2154 # yubikey_dir is setup-yubikey.sh's
keys_done() {
    (ws_conf "$WS_CONF" && ws_identity && keys_there &&
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

# keys_do makes the key that keys_done lacks, or says why it can't. Without WS_IDENTITY, it runs
# setup.sh's YubiKey and agent steps, which find or make the YubiKey's keys and have ws's agent load
# them, or stop. A WS_IDENTITY outside Secretive's PublicKeys, with no agent, is a key in a file,
# which keys_file makes. Else ws_identity names its agent, which must list it.
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

# keys_fail says what to do where the Mac's key is still missing after keys_do.
keys_fail() {
    echo "ssh has no usable key for ws, as above; mend that, then run the wizard again"
}
