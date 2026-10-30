# shellcheck shell=sh
# The functions of mac/setup.sh that set the key that ssh offers ws, the YubiKey's key of mac.conf's
# WS_SSH_KEY, which setup-conf.sh's ws_conf reads. setup-lib.sh sources it after setup-conf.sh, so
# that a script that sources setup-lib.sh has them too.

# ws_identity [FILE] sets ws_identity_pub to the .pub of the key that ssh offers ws, the YubiKey's
# WS_SSH_KEY, and ws_identity_agent to the socket of ws's agent, which holds it. Both are paths as
# config.d/ws names them. setup.sh writes the .pub in ~/.config/ws/keys from the YubiKey before this
# runs, so it goes unchecked here, and --check may lack it. It warns where FILE, the config.d/ws it
# would replace, gave ws another key.
# shellcheck disable=SC2153,SC2154 # mac.conf's keys are ws_conf's, the socket setup-conf.sh's
ws_identity() {
    ws_identity_pub=\~/.config/ws/keys/$WS_SSH_KEY.pub
    ws_identity_agent=\~/${ws_agent_socket#"$HOME"/}
    ws_old=$(sed -n 's/^    IdentityFile //p' "${1-/dev/null}" 2>/dev/null) || ws_old=
    [ -z "$ws_old" ] || [ "$ws_old" = "$ws_identity_pub" ] || ws_say "warning: the YubiKey's key," \
        "$ws_identity_pub, replaces $ws_old as ws's. The wizard's gh stage puts it on GitHub, whose" \
        "keys a ws installed later imports. An installed ws takes it by hand:" \
        "$(ws_identity_step "$WS_USER")"
}

# ws_identity_step USER prints the step by hand that gives an installed ws the Mac's new key, named
# before it, for USER's logins, as ws imports the GitHub account's keys at its install alone.
ws_identity_step() {
    echo "add the key's line to $1's ~/.ssh/authorized_keys on ws, at ws's console or over ssh" \
        "by the old key, as docs/mac-yubikey.md shows, with no reinstall"
}

# ws_identity_lines prints config.d/ws's lines for the key that ws_identity set, which ssh offers ws
# alone: the key of ws's agent that matches the .pub.
ws_identity_lines() {
    printf '    %s\n' "IdentityFile $ws_identity_pub" 'IdentitiesOnly yes' \
        "IdentityAgent \"$ws_identity_agent\""
}
