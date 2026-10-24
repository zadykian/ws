# shellcheck shell=sh
# The functions of mac/setup.sh that set the key that ssh offers ws, from mac.conf's WS_IDENTITY and
# WS_IDENTITY_AGENT, which setup-conf.sh's ws_conf reads. setup-lib.sh sources it after
# setup-conf.sh, so that a script that sources setup-lib.sh has them too.

# ws_identity sets ws_identity_pub to the .pub of the key that ssh offers ws, and ws_identity_agent
# to its agent's socket, or to nothing for a key in a file. Both are paths as config.d/ws names
# them: mac.conf's WS_IDENTITY and WS_IDENTITY_AGENT. Without WS_IDENTITY, the key is the file key,
# ~/.ssh/id_ed25519, as before.
# shellcheck disable=SC2153 # WS_IDENTITY_AGENT is ws_conf's
ws_identity() {
    ws_identity_pub=${WS_IDENTITY:-\~/.ssh/id_ed25519.pub}
    ws_identity_agent=$WS_IDENTITY_AGENT
}

# ws_identity_lines prints config.d/ws's lines for the key that ws_identity set, which ssh offers ws
# alone: with an agent, the agent's key that matches the .pub. Else the file beside the .pub, whose
# passphrase the Keychain keeps, and which the agent takes once ssh has read it.
ws_identity_lines() {
    if [ -n "$ws_identity_agent" ]; then
        printf '    %s\n' "IdentityFile $ws_identity_pub" 'IdentitiesOnly yes' \
            "IdentityAgent \"$ws_identity_agent\""
    else
        printf '    %s\n' "IdentityFile ${ws_identity_pub%.pub}" 'IdentitiesOnly yes' \
            'UseKeychain yes' 'AddKeysToAgent yes'
    fi
}
