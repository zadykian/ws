# shellcheck shell=sh
# The functions of mac/setup.sh that set the key that ssh offers ws, from mac.conf's WS_SSH_KEY,
# WS_IDENTITY and WS_IDENTITY_AGENT, which setup-conf.sh's ws_conf reads. setup-lib.sh sources it
# after setup-conf.sh, so that a script that sources setup-lib.sh has them too.

# ws_identity [FILE] sets ws_identity_pub to the .pub of the key that ssh offers ws, and
# ws_identity_agent to its agent's socket, or to nothing for a key in a file. Both are paths as
# config.d/ws names them. Without mac.conf's WS_IDENTITY, the key is the YubiKey's WS_SSH_KEY, in
# ws's agent. setup.sh writes its .pub in ~/.config/ws/keys from the YubiKey before this runs, so
# it goes unchecked here, and --check may lack it. ws_identity_old warns then where FILE, the
# config.d/ws it would replace, gave ws another key. Else the key is WS_IDENTITY, and its agent
# WS_IDENTITY_AGENT, or Secretive's for a .pub in its PublicKeys. It stops where that .pub, but the
# file key's, is missing or unreadable, or holds other than one public key, as a private key or a
# certificate does. A key that ssh-keygen can't read stops it, as does a type ws's OpenSSH 10.2
# lacks (ssh -Q key there), as an ML-DSA key of Secretive's.
# shellcheck disable=SC2153,SC2154 # mac.conf's keys are ws_conf's, the sockets setup-conf.sh's
ws_identity() {
    ws_identity_pub=$WS_IDENTITY
    ws_identity_agent=$WS_IDENTITY_AGENT
    if [ -z "$ws_identity_pub" ]; then
        ws_identity_pub=\~/.config/ws/keys/$WS_SSH_KEY.pub
        ws_identity_agent=\~/${ws_agent_socket#"$HOME"/}
        ws_identity_old "${1-}"
        return 0
    fi
    [ "$ws_identity_pub$ws_identity_agent" != \~/.ssh/id_ed25519.pub ] || return 0
    ws_file=$ws_identity_pub
    [ "${ws_file#\~/}" = "$ws_file" ] || ws_file=$HOME/${ws_file#\~/}
    case $ws_file in
    "$ws_secretive"/PublicKeys/*)
        [ -n "$ws_identity_agent" ] || ws_identity_agent=\~/${ws_secretive#"$HOME"/}/socket.ssh
        ;;
    esac
    [ -f "$ws_file" ] || ws_die "$ws_file, mac.conf's WS_IDENTITY, is missing: make the key first"
    ws_type=$(awk 'NF { n++; t = $1; k = $2 }
        END { if (n == 1 && k ~ "^[A-Za-z0-9+/]+=*$") print t }' "$ws_file" 2>/dev/null) ||
        ws_die "cannot read $ws_file: make it readable, or name another in mac.conf's WS_IDENTITY"
    case $ws_type in
    ssh-ed25519 | ecdsa-sha2-nistp256 | ecdsa-sha2-nistp384 | ecdsa-sha2-nistp521 | ssh-rsa | \
        sk-ssh-ed25519@openssh.com | sk-ecdsa-sha2-nistp256@openssh.com) ;;
    *-cert-v01@openssh.com) ws_die "$ws_file is a certificate: name its key's own .pub in" \
        "mac.conf's WS_IDENTITY" ;;
    ssh-?* | ecdsa-sha2-?* | sk-?*)
        ws_die "$ws_file holds a key of type $ws_type, which ws's OpenSSH lacks: name the .pub of" \
            "a P-256 or ed25519 key in mac.conf's WS_IDENTITY"
        ;;
    *) ws_die "$ws_file holds no public key alone, as ssh-keygen -y prints one" ;;
    esac
    ssh-keygen -l -f "$ws_file" >/dev/null 2>&1 || ws_die "ssh-keygen cannot read $ws_file: name" \
        "the .pub of a P-256 or ed25519 key in mac.conf's WS_IDENTITY"
}

# ws_identity_old FILE warns where FILE, the config.d/ws that setup.sh replaces, gave ws another key
# than ws_identity's, with no WS_IDENTITY, as Secretive's key that an older run's search found.
ws_identity_old() {
    [ -z "$WS_IDENTITY" ] || return 0
    ws_old=$(sed -n 's/^    IdentityFile //p' "${1:-/dev/null}" 2>/dev/null) || ws_old=
    [ -z "$ws_old" ] || [ "${ws_old%.pub}" = "${ws_identity_pub%.pub}" ] ||
        ws_say "warning: the key of $ws_identity_pub replaces $ws_old as ws's: add that line to" \
            "ws's authorized_keys, or keep the old key with WS_IDENTITY=$ws_old in mac.conf"
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
