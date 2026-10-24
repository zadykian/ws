# shellcheck shell=sh
# The git step of mac/setup.sh, which setup-extras.sh sources: git's identity and a signing key of
# the Mac's own. They go in ~/.config/ws/gitconfig, as the shell role sets them up on ws. It calls
# the functions of lib.sh and setup-lib.sh.

# The directory of mac.conf, where the step writes its files.
git_dir=$HOME/.config/ws

# git_mac sets git up from mac.conf's WS_GIT_NAME, WS_GIT_EMAIL and WS_GIT_SIGNING_KEY, and leaves
# it alone without them.
git_mac() {
    [ -n "$WS_GIT_NAME" ] || return 0
    git_key
    git_files
}

# git_key sets git_pub to the signing key's .pub, WS_GIT_SIGNING_KEY with its ~/ the home, and
# git_public to the key's type and base64. Without WS_GIT_SIGNING_KEY, the key is
# ~/.ssh/id_ed25519_signing, which it makes once with no passphrase, as the shell role does on ws:
# git signs unattended, in a rebase too. It makes no other key, and replaces no .pub. It stops where
# the .pub is missing, holds more than a public key's line, as a private key does, or ssh-keygen
# can't read it.
git_key() {
    git_pub=${WS_GIT_SIGNING_KEY:-$HOME/.ssh/id_ed25519_signing.pub}
    case $git_pub in
    \~/*) git_pub=$HOME/${git_pub#\~/} ;;
    esac
    git_public=
    if [ -z "$WS_GIT_SIGNING_KEY" ] && [ ! -e "${git_pub%.pub}" ]; then
        [ ! -e "$git_pub" ] && [ ! -L "$git_pub" ] || ws_die "$git_pub is there without its" \
            "private key: name it in mac.conf's WS_GIT_SIGNING_KEY where an agent holds the key," \
            "or remove it"
        if ws_checking; then
            ws_say "would make git's signing key, ${git_pub%.pub}"
            return 0
        fi
        ssh-keygen -q -t ed25519 -N '' -f "${git_pub%.pub}" </dev/null >&2 ||
            ws_die "ssh-keygen cannot make ${git_pub%.pub}"
        ws_say "made git's signing key, ${git_pub%.pub}"
    fi
    if [ ! -f "$git_pub" ]; then
        [ -n "$WS_GIT_SIGNING_KEY" ] || ws_die "$git_pub is missing beside its key:" \
            "ssh-keygen -y -f ${git_pub%.pub} >$git_pub writes it"
        ws_die "$git_pub, mac.conf's WS_GIT_SIGNING_KEY, is missing: make the key first, as" \
            "docs/mac-git.md says"
    fi
    git_public=$(awk 'NF { n++; key = $1 " " $2 } END { if (n == 1) print key }' "$git_pub")
    case $git_public in
    *' '*[!A-Za-z0-9+/=]*) git_public= ;;
    ssh-?*' '?* | ecdsa-sha2-?*' '?* | sk-?*' '?*) ;;
    *) git_public= ;;
    esac
    [ -n "$git_public" ] || ws_die "$git_pub holds no public key alone, as ssh-keygen -y prints one"
    ssh-keygen -l -f "$git_pub" >/dev/null 2>&1 ||
        ws_die "ssh-keygen cannot read $git_pub: git signs with a key the Mac's ssh-keygen knows," \
            "as ed25519"
}

# git_files writes ~/.config/ws/allowed_signers, which trusts the key for WS_GIT_EMAIL in git
# verify-commit, and ~/.config/ws/gitconfig. Under --check, before the key is made, it writes no
# allowed signers.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
git_files() {
    if [ -n "$git_public" ]; then
        ws_put "$git_dir/allowed_signers" 644 <<EOF
# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf: its next run replaces changes made here.
$WS_GIT_EMAIL $git_public
EOF
    else
        ws_say "would write $git_dir/allowed_signers once the key is made"
    fi
    ws_tmp
    git_new=$ws_tmpfile
    git_conf >"$git_new"
    ws_put "$git_dir/gitconfig" 644 <"$git_new"
    rm -f "$git_new"
}

# git_conf prints ~/.config/ws/gitconfig: the identity, and each commit signed with the key, which
# the allowed signers file trusts for the email. The name and the paths go in quotes, which
# mac.conf's values cannot end.
git_conf() {
    cat <<EOF
# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf: its next run replaces changes made here.
# Include it at the top of ~/.gitconfig, so that that file's own settings win where both set one.
[user]
    name = "$WS_GIT_NAME"
    email = $WS_GIT_EMAIL
    signingkey = "$git_pub"
[commit]
    gpgsign = true
[gpg]
    format = ssh
[gpg "ssh"]
    allowedSignersFile = "$git_dir/allowed_signers"
EOF
}
