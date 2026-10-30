# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's Mac key stage: the YubiKey's keys for ssh and git, which the stage finds or makes as
# setup.sh does, and ws's agent loads. wizard.sh sources it, and runs the stage after Answers and
# before gh on the Mac, which adds ssh's key to GitHub. It calls the functions of lib.sh and
# setup-lib.sh, and reads WS_CONF, mac.conf's path.

# The YubiKey's step and the agent's are setup.sh's, sourced from dir, mac/'s path, as wizard.sh
# sets it.
# shellcheck source=setup-yubikey.sh
. "${dir:?wizard.sh sets dir}/setup-yubikey.sh"
# shellcheck source=setup-agent.sh
. "$dir/setup-agent.sh"

# keys_after prints Tools, bootstrap.sh's stage, where PATH has no brew to install ykman.
keys_after() {
    command -v brew >/dev/null || echo Tools
}

# keys_done succeeds where ws's agent lists the YubiKey's keys for ssh and git, as it lists them only
# while the YubiKey is in. So a run with the YubiKey out waits for it here, rather than failing at
# Root's key.
keys_done() {
    (ws_conf "$WS_CONF" && keys_listed) >/dev/null 2>&1
}

# keys_home PATH prints PATH with a ~/ first made the home's path.
keys_home() {
    case $1 in
    \~/*) echo "$HOME/${1#\~/}" ;;
    *) echo "$1" ;;
    esac
}

# keys_do runs setup.sh's YubiKey and agent steps, once one YubiKey is in, which find or make the
# YubiKey's keys and have ws's agent load them, or stop.
keys_do() {
    ws_conf "$WS_CONF"
    yubikey_tools "Mac key"
    keys_plugged || return 0
    yubikey_keys "Mac key"
    keys_agent
}

# keys_plugged waits up to 5 minutes for a YubiKey where none is in, as the one with the keys may be
# out at a run's start. Where several are in, yubikey_keys stops after it. --check waits for none:
# it says what a run would do, and fails, and sets keys_wait, at which setup.sh's stage and Root's
# key's come after this one, as both need the YubiKey. Before ykman's install, it passes, as
# yubikey_keys then says what it would do.
# shellcheck disable=SC2154 # yubikey_tools sets yubikey_ykman
keys_plugged() {
    [ -n "$yubikey_ykman" ] && ! keys_in || return 0
    if ws_checking; then
        ws_say "Mac key: no YubiKey is in; a run would wait up to 5 minutes for one, then find" \
            "or make the keys $WS_SSH_KEY and $WS_GIT_KEY on it"
        # shellcheck disable=SC2034 # setup_after and root_after read it
        keys_wait=1
        return 1
    fi
    ws_say "Mac key: no YubiKey is in: plug in the one with ws's keys, or a new one; the wizard" \
        "waits up to 5 minutes for it"
    keys_left=150
    until keys_in; do
        [ "$((keys_left -= 1))" -ge 0 ] || ws_die "Mac key: no YubiKey came in 5 minutes: plug" \
            "in the one with ws's keys, or a new one, then run the wizard again"
        sleep 2
    done
}

# keys_in succeeds where a YubiKey is in, as ykman lists its serial. It stops where ykman fails.
keys_in() {
    keys_serials=$(yubikey_serials) || ws_die "Mac key: ykman list --serials failed"
    [ -n "$keys_serials" ]
}

# keys_agent has ws's agent hold the YubiKey's keys, where it lacks one: setup.sh's agent step
# installs the agent, or has launchd start it again. Then it waits up to 15 s for the agent to list
# both, as an agent that launchd doesn't run loads them within about 5 s of a change. --check says
# what a run would do.
keys_agent() {
    keys_listed && return 0
    if ws_checking; then
        ws_say "Mac key: would load the keys $WS_SSH_KEY and $WS_GIT_KEY into ws's agent, as" \
            "setup.sh does, and wait for it to list them"
        return 0
    fi
    agent
    keys_left=15
    until keys_listed; do
        [ "$((keys_left -= 1))" -ge 0 ] || return 1
        sleep 1
    done
}

# keys_listed succeeds where ws's agent lists the keys of the .pubs of WS_SSH_KEY and WS_GIT_KEY,
# which the YubiKey's step writes. It sets keys_list and keys_status to what ssh-add -L prints, its
# error too, and its status, and keys_lacks to the first .pub whose key the agent lacks.
# shellcheck disable=SC2154 # ws_agent_socket is setup-conf.sh's, yubikey_dir setup-yubikey.sh's
keys_listed() {
    keys_list=$(SSH_AUTH_SOCK=$ws_agent_socket ssh-add -L 2>&1)
    keys_status=$?
    for keys_lacks in "$yubikey_dir/$WS_SSH_KEY.pub" "$yubikey_dir/$WS_GIT_KEY.pub"; do
        [ "$keys_status" = 0 ] && keys_line=$(awk 'NF { print $1, $2; exit }' "$keys_lacks") &&
            printf '%s\n' "$keys_list" | awk '{ print $1, $2 }' | grep -qxF -e "$keys_line" ||
            return 1
    done 2>/dev/null
    keys_lacks=
}

# keys_fail says what to do where ws's agent still lacks one of the YubiKey's keys after keys_do. It
# gives ssh-add -L's own error, as with no agent at the socket, or names the .pub whose key the
# agent lacks.
keys_fail() {
    keys_listed || :
    keys_why=" lists no key of $keys_lacks"
    [ "$keys_status" = 0 ] || keys_why=": ssh-add -L: $(printf '%s\n' "$keys_list" | tail -n 1)"
    echo "ws's agent at $ws_agent_socket${keys_why%.}. launchd runs it from the login at the" \
        "Mac's screen, and its log, ~/Library/Logs/ws-agent.log, says why it lacks a key: mend" \
        "that, take the YubiKey out and in again, then run the wizard again"
}
