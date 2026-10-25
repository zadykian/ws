# shellcheck shell=sh
# The wizard's Mac key stage: the key that ssh offers ws, which setup-identity.sh's ws_identity
# finds from mac.conf, made where missing. wizard.sh sources it, and runs the stage after Answers
# and before setup.sh, which names the key in ~/.ssh/config.d/ws. Without WS_IDENTITY, the key is
# Secretive's, in the Mac's Secure Enclave, unless the answer at the terminal is no: then ssh-keygen
# makes ~/.ssh/id_ed25519. It calls the functions of lib.sh, setup-lib.sh and wizard-ask.sh, and
# reads wizard.sh's lock and WS_CONF, mac.conf's path.

# keys_done succeeds where the key that ssh offers ws is there: in a file beside the .pub whose
# fingerprint it shares, or listed by the agent that holds it.
keys_done() {
    (ws_conf "$WS_CONF" && ws_identity && keys_there) >/dev/null 2>&1
}

# keys_there succeeds where the key that ws_identity set is there: the agent lists the key of the
# .pub, or the file beside the .pub is its private key.
# shellcheck disable=SC2154 # ws_identity sets ws_identity_pub and ws_identity_agent
keys_there() {
    keys_pub=$(keys_home "$ws_identity_pub")
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

# keys_do makes the key that keys_done lacks, or says why it can't. A WS_IDENTITY outside
# Secretive's PublicKeys, with no agent, is a key in a file, which keys_file makes. Without
# WS_IDENTITY, the key that ws_identity finds is there, or keys_choose makes one.
keys_do() {
    ws_conf "$WS_CONF"
    keys_pub=$(keys_home "$WS_IDENTITY")
    if [ -n "$keys_pub" ] && [ -z "$WS_IDENTITY_AGENT" ] &&
        [ "${keys_pub#"$ws_secretive"/PublicKeys/}" = "$keys_pub" ]; then
        keys_file "${keys_pub%.pub}"
        return
    fi
    ws_identity
    keys_pub=$(keys_home "$ws_identity_pub")
    if [ -n "$ws_identity_agent" ]; then
        keys_there || ws_die "Mac key: the agent at $(keys_home "$ws_identity_agent") lists no" \
            "key of $keys_pub: open Secretive, where it is that agent, or start the agent, then" \
            "run the wizard again"
    elif [ -e "$keys_pub" ] || [ -e "${keys_pub%.pub}" ]; then
        keys_file "${keys_pub%.pub}"
    else
        keys_choose
    fi
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

# keys_choose makes ws's key where it has none: in Secretive, where the answer at the terminal is
# other than no, else in keys_pub's file, by keys_make. --check asks nothing, and says what each
# answer would do. With no terminal, it stops, naming the key that answers.
keys_choose() {
    if ws_checking; then
        ws_say "Mac key: would ask whether Secretive keeps the Mac's key: yes installs its cask," \
            "no makes ${keys_pub%.pub} with ssh-keygen"
    else
        [ -t 0 ] || ws_die "Mac key: no terminal to ask whether Secretive keeps the Mac's key:" \
            "set WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for ssh-keygen's, or run the" \
            "wizard at a terminal"
        printf "Keep the Mac's ssh key in Secretive, in the Secure Enclave? [Y/n]: " >&2
        read -r keys_answer || ws_die "Mac key: the terminal ended before the answer"
        case $keys_answer in
        [Nn]*) keys_make "${keys_pub%.pub}" ;;
        *) keys_secretive ;;
        esac
    fi
}

# keys_secretive has the user make ws's key in Secretive's app, installed where missing, at its
# default protection, which asks for Touch ID at each use. It writes the key's .pub and the agent
# into mac.conf, as a second key there would stop ws_identity's default, and the agent must list it.
keys_secretive() {
    keys_agent=\~/${ws_secretive#"$HOME"/}/socket.ssh
    if ws_checking; then
        keys_app || ws_say "Mac key: would run brew install --cask secretive"
        ws_say "Mac key: would open Secretive, and wait for ws's key made there"
        return 0
    fi
    keys_app || brew install --cask secretive >&2 || ws_die "Mac key: brew install --cask" \
        "secretive failed; install Secretive, or answer no for ssh-keygen's key"
    open -a Secretive || ws_say "Mac key: warning: open cannot find Secretive: open it by hand"
    keys_skip=$(keys_home "$WS_GIT_SIGNING_KEY")
    keys_pick "a P-256 key for ws's login, at its default protection"
    keys_login=\~/${keys_got#"$HOME"/}
    ! ask_lacks WS_IDENTITY || ask_write "$(ask_line WS_IDENTITY "$keys_login")
$(ask_line WS_IDENTITY_AGENT "$keys_agent")
" "Mac key" || return 1
    ws_conf "$WS_CONF"
    ws_identity
    keys_there || ws_die "Mac key: Secretive's agent lists no key of $keys_pub: finish" \
        "Secretive's setup, then run the wizard again"
}

# keys_pick WHAT sets keys_got to the .pub in PublicKeys other than keys_skip's. Where none is
# there, it says to make WHAT in Secretive's app, and waits up to 15 minutes for one.
keys_pick() {
    if ! keys_other; then
        ws_say "Mac key: in Secretive, make $1; the wizard waits up to 15 minutes for it"
        keys_left=450
        until keys_other; do
            [ "$((keys_left -= 1))" -ge 0 ] || ws_die "Mac key: Secretive made no key in 15" \
                "minutes: make $1, then run the wizard again"
            sleep 2
        done
    fi
}

# keys_other succeeds where PublicKeys holds a .pub other than keys_skip's, as Secretive writes one
# for each key, and sets keys_got to it.
keys_other() {
    keys_got=
    for keys_each in "$ws_secretive"/PublicKeys/*.pub; do
        case $keys_each in "$keys_skip" | *-cert.pub) continue ;; esac
        [ -f "$keys_each" ] || continue
        keys_got=$keys_each
    done
    [ -n "$keys_got" ]
}

# keys_app succeeds where Secretive's app is in /Applications, where its cask puts it, or in
# ~/Applications.
keys_app() {
    [ -d /Applications/Secretive.app ] || [ -d "$HOME/Applications/Secretive.app" ]
}

# keys_fail says what to do where the Mac's key is still missing after keys_do.
keys_fail() {
    echo "ssh has no usable key for ws, as above; mend that, then run the wizard again"
}
