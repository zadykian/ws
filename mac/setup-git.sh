# shellcheck shell=sh
# The git step of mac/setup.sh, which setup-extras.sh sources: a signing key of the Mac's own for
# git, as the shell role makes one on ws. It calls the functions of lib.sh and setup-lib.sh.

# git_mac sets git up from mac.conf's WS_GIT_NAME, WS_GIT_EMAIL and WS_GIT_SIGNING_KEY, and leaves
# it alone without them.
git_mac() {
    [ -n "$WS_GIT_NAME" ] || return 0
    git_key
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
