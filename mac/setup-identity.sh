# shellcheck shell=sh
# The functions of mac/setup.sh that set the key that ssh offers ws, from mac.conf's WS_IDENTITY and
# WS_IDENTITY_AGENT, which setup-conf.sh's ws_conf reads. setup-lib.sh sources it after
# setup-conf.sh, so that a script that sources setup-lib.sh has them too.

# ws_identity [FILE] sets ws_identity_pub to the .pub of the key that ssh offers ws, and
# ws_identity_agent to its agent's socket, or to nothing for a key in a file. Both are paths as
# config.d/ws names them. They are mac.conf's WS_IDENTITY and WS_IDENTITY_AGENT, where set. Else,
# where Secretive's socket is there, the key is its one in PublicKeys. git's signing key, which
# signs alone, does not count, nor a certificate, -cert.pub, which Secretive kept there before 4.0,
# nor a name ssh_config would misread, as Secretive's own are hex. Else it is the file key,
# ~/.ssh/id_ed25519, as before. A key in PublicKeys takes Secretive's agent. It stops where
# Secretive holds several such keys. It stops too where a .pub but the file key's is missing or
# unreadable, or holds other than one public key, as a private key or a certificate does. A key
# that ssh-keygen can't read stops it, as does a type ws's OpenSSH 10.2 lacks (ssh -Q key there),
# as an ML-DSA key of Secretive's. It warns where FILE, the config.d/ws it would replace, gave ws
# the file key, as Secretive's takes its place.
# shellcheck disable=SC2153,SC2154 # WS_IDENTITY_AGENT is ws_conf's and ws_secretive setup-conf.sh's
ws_identity() {
    ws_identity_pub=$WS_IDENTITY
    ws_identity_agent=$WS_IDENTITY_AGENT
    if [ -z "$ws_identity_pub" ] && [ -S "$ws_secretive/socket.ssh" ]; then
        ws_skip=$WS_GIT_SIGNING_KEY
        [ "${ws_skip#\~/}" = "$ws_skip" ] || ws_skip=$HOME/${ws_skip#\~/}
        ws_keys=
        for ws_each in "$ws_secretive"/PublicKeys/*.pub; do
            case $ws_each in "$ws_skip" | *-cert.pub) continue ;; esac
            [ -f "$ws_each" ] || continue
            case ${ws_each##*/} in *[!A-Za-z0-9._@+-]*) continue ;; esac
            ws_keys="$ws_keys${ws_keys:+, }$ws_each"
            ws_identity_pub=\~/${ws_each#"$HOME"/}
        done
        case $ws_keys in
        *', '*) ws_die "Secretive's agent holds several keys, $ws_keys: name the one for ws in" \
            "mac.conf's WS_IDENTITY" ;;
        esac
    fi
    ws_identity_pub=${ws_identity_pub:-\~/.ssh/id_ed25519.pub}
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
    [ -n "$WS_IDENTITY" ] || ! grep -qsx '    IdentityFile ~/.ssh/id_ed25519' "${1-/dev/null}" ||
        ws_say "warning: Secretive's key replaces ~/.ssh/id_ed25519 as ws's: keep the file key" \
            "with WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for an installed ws, or put the" \
            "key of $ws_identity_pub on GitHub before ws's next install"
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
