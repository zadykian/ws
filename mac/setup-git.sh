# shellcheck shell=sh
# The git step of mac/setup.sh, which setup-extras.sh sources, as wizard-gh.sh does for git_gh:
# git's identity and a signing key of the Mac's own. They go in ~/.config/ws/gitconfig, which
# ~/.gitconfig includes at its top, as the shell role sets them up on ws. gh, from Homebrew where
# missing, takes git's pushes to GitHub, and registers the key there, as the devtools role does on
# ws. It calls the functions of lib.sh and setup-lib.sh.

# The directory of mac.conf, where the step writes its files.
git_dir=$HOME/.config/ws
# git's XDG file, which git reads before ~/.gitconfig.
git_xdg=${XDG_CONFIG_HOME:-$HOME/.config}/git/config

# git_mac sets git up from mac.conf's WS_GIT_NAME, WS_GIT_EMAIL, WS_GIT_KEY and
# WS_GIT_SIGNING_KEY, and leaves it alone without the first two.
git_mac() {
    [ -n "$WS_GIT_NAME" ] || return 0
    git_include_ok
    git_key
    git_gh
    git_files
    git_include
    git_register
}

# git_key sets git_pub to the signing key's .pub, WS_GIT_SIGNING_KEY with its ~/ the home, and
# git_public to the key's type and base64. Without WS_GIT_SIGNING_KEY, the key is the YubiKey's
# WS_GIT_KEY, whose .pub setup.sh writes in ~/.config/ws/keys from the YubiKey before this runs,
# so --check may lack it. It makes no key, and stops where the .pub is missing or unreadable,
# holds more than a public key's line, as a private key does, or ssh-keygen can't read it.
# shellcheck disable=SC2153 # WS_GIT_KEY is ws_conf's
git_key() {
    git_pub=${WS_GIT_SIGNING_KEY:-$git_dir/keys/$WS_GIT_KEY.pub}
    case $git_pub in
    \~/*) git_pub=$HOME/${git_pub#\~/} ;;
    esac
    git_public=
    if [ ! -f "$git_pub" ]; then
        [ -n "$WS_GIT_SIGNING_KEY" ] || ! ws_checking || return 0
        [ -n "$WS_GIT_SIGNING_KEY" ] || ws_die "$git_pub, the .pub of the YubiKey's key" \
            "$WS_GIT_KEY, is missing"
        ws_die "$git_pub, mac.conf's WS_GIT_SIGNING_KEY, is missing: make the key first, as" \
            "docs/mac-git.md says"
    fi
    git_public=$(awk 'NF { n++; key = $1 " " $2 } END { if (n == 1) print key }' "$git_pub" \
        2>/dev/null) || ws_die "cannot read $git_pub: make it readable, or name another .pub in" \
        "mac.conf's WS_GIT_SIGNING_KEY"
    case $git_public in
    *' '*[!A-Za-z0-9+/=]*) git_public= ;;
    ssh-?*' '?* | ecdsa-sha2-?*' '?* | sk-?*' '?*) ;;
    *) git_public= ;;
    esac
    [ -n "$git_public" ] || ws_die "$git_pub holds no public key alone, as ssh-keygen -y prints one"
    ssh-keygen -l -f "$git_pub" >/dev/null 2>&1 ||
        ws_die "ssh-keygen cannot read $git_pub: git signs with a key the Mac's ssh-keygen knows," \
            "as ed25519 or Secretive's P-256"
}

# git_gh [STAGE] sets gh to the path of GitHub's CLI, which Homebrew installs where PATH has none.
# Under --check, it is the path Homebrew would give it. git runs gh in whatever directory it
# works in, so the path must be absolute, not one that a relative directory of PATH gives. Its
# messages name STAGE, where the wizard's stage gives one.
# shellcheck disable=SC2120 # wizard-gh.sh's stage passes its name
git_gh() {
    if ! gh=$(command -v gh); then
        brew=$(command -v brew) || ws_die "${1:+$1: }install Homebrew first: https://brew.sh"
        if ws_checking; then
            ws_say "${1:+$1: }would run brew install gh"
            gh=$("$brew" --prefix)/bin/gh
            return 0
        fi
        ws_say "${1:+$1: }installing gh"
        "$brew" install gh >&2 || ws_die "${1:+$1: }brew install gh failed"
        gh=$(command -v gh) || ws_die "${1:+$1: }brew install gh left no gh on PATH"
    fi
    case $gh in
    /*) ;;
    *) ws_die "${1:+$1: }PATH gives gh as $gh, which git would look for in each repository:" \
        "drop PATH's relative directories" ;;
    esac
}

# git_files writes ~/.config/ws/allowed_signers, which trusts the key for WS_GIT_EMAIL in git
# verify-commit, and ~/.config/ws/gitconfig. For the YubiKey's key, git signs through git_agent's
# ~/.config/ws/ws-ssh-keygen, which gives ssh-keygen ws's agent. So a terminal and an IDE sign
# alike, whatever agent the app that runs git has. Under --check, before the key is made, it writes
# no allowed signers.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
git_files() {
    git_program=
    case $git_pub in
    "$git_dir"/keys/*.pub) git_agent ;;
    esac
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
# the allowed signers file trusts for the email, through git_program where there is one. Then gh's
# credential command, as gh auth setup-git writes it, after an empty value that drops any set before
# it. The name and the paths go in quotes, which mac.conf's values cannot end.
git_conf() {
    cat <<EOF
# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf: its next run replaces changes made here.
# ~/.gitconfig includes it at its top, so that its own settings win where both set one.
[user]
    name = "$WS_GIT_NAME"
    email = $WS_GIT_EMAIL
    signingkey = "$git_pub"
[commit]
    gpgsign = true
[credential "https://github.com"]
    helper =
    helper = !$gh auth git-credential
[credential "https://gist.github.com"]
    helper =
    helper = !$gh auth git-credential
[gpg]
    format = ssh
[gpg "ssh"]
    allowedSignersFile = "$git_dir/allowed_signers"
EOF
    [ -z "$git_program" ] || echo "    program = \"$git_program\""
}

# git_include_ok sets git_included where ~/.gitconfig includes ~/.config/ws/gitconfig, first and
# once. It stops where git can't read the file, and where the include comes twice or sits below the
# file's own settings, which it then overrides. It stops too where the file is a link that lacks the
# include, rather than replace the link with a file. It writes nothing, so that the step stops
# before it acts, as setup-ssh.sh's ws_include_ok does for ssh's Include.
git_include_ok() {
    git_file=$HOME/.gitconfig
    git_included=
    [ -e "$git_file" ] || [ -L "$git_file" ] || return 0
    git_paths=$(git config --file "$git_file" --type path --get-all include.path) ||
        [ $? = 1 ] || ws_die "git config cannot read $git_file"
    if printf '%s\n' "$git_paths" | git_ours -q; then
        # The file's first setting must be the include, of ~/.config/ws/gitconfig.
        git_first=$(git config --file "$git_file" --name-only --list | head -n 1)
        [ "$git_first" != include.path ] || git_first=$(printf '%s\n' "$git_paths" | head -n 1)
        printf '%s\n' "$git_first" | git_ours -q ||
            ws_die "$git_file includes ~/.config/ws/gitconfig below its own settings, which" \
                "it then overrides: move the include to the top"
        git_count=$(printf '%s\n' "$git_paths" | git_ours -c)
        [ "$git_count" = 1 ] || ws_die "$git_file includes ~/.config/ws/gitconfig $git_count" \
            "times: keep the one at the top"
        git_included=1
        return 0
    fi
    [ ! -L "$git_file" ] || ws_die "$git_file is a link to $(readlink "$git_file"): put" \
        "'[include] path = ~/.config/ws/gitconfig' at the top of that file"
}

# git_include puts an include of ~/.config/ws/gitconfig at the top of ~/.gitconfig where
# git_include_ok found none, so that the file's own settings come after it and win. It makes the
# file where missing, and keeps the old one as ~/.gitconfig.TIME~, as setup-ssh.sh keeps
# ~/.ssh/config. It names the file's own settings of identity and signing, which win, and those of
# git's XDG file, which lose.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
git_include() {
    [ -z "$git_included" ] || return 0
    git_mode=644
    if [ -e "$git_file" ]; then
        git_mode=$(ws_mode "$git_file")
        if ! ws_checking; then
            git_backup=$git_file.$(date +%Y%m%d%H%M%S)~
            cp -p "$git_file" "$git_backup" || ws_die "cannot keep $git_file as $git_backup"
            ws_say "kept the old $git_file as $git_backup"
        fi
    fi
    ws_tmp
    git_top=$ws_tmpfile
    {
        printf '[include]\n    path = ~/.config/ws/gitconfig\n'
        [ ! -f "$git_file" ] || { echo && cat "$git_file"; }
    } >"$git_top" || ws_die "cannot write $git_top"
    ws_put "$git_file" "$git_mode" <"$git_top"
    rm -f "$git_top"
    git_list=$(git_own "$git_file")
    [ -z "$git_list" ] || ws_say "warning: $git_file's own settings of $git_list, below the" \
        "include, win over ~/.config/ws/gitconfig's"
    git_list=$(git_own "$git_xdg")
    [ -z "$git_list" ] || ws_say "warning: $git_xdg's settings of $git_list come before the" \
        "include in ~/.gitconfig, so ~/.config/ws/gitconfig's win over them"
}

# git_ours -q|-c greps stdin for the lines that name ~/.config/ws/gitconfig, as git config prints
# an include of ~/.gitconfig: by the home's path, or relative to the home.
git_ours() {
    grep "$1" -xF -e "$git_dir/gitconfig" -e .config/ws/gitconfig
}

# git_own FILE prints the settings of identity and signing that the git config FILE sets, once
# each, in its order, and nothing where FILE is missing.
git_own() {
    [ -f "$1" ] || return 0
    git config --file "$1" --name-only --list |
        grep -xE -e 'user\.(name|email|signingkey)|commit\.gpgsign' \
            -e 'gpg\.(format|ssh\.program|ssh\.allowedsignersfile)' |
        awk '!seen[$0]++ { printf "%s%s", n++ ? ", " : "", $0 }'
}

# git_listed succeeds where the signing keys of gh's login hold the key. It sets git_status to
# gh's exit status, and keeps gh's errors in git_err.
git_listed() {
    git_status=0
    git_keys=$("$gh" api --paginate user/ssh_signing_keys --jq '.[].key' 2>"$git_err") ||
        git_status=$?
    [ "$git_status" = 0 ] && [ -n "$git_public" ] &&
        printf '%s\n' "$git_keys" | grep -qxF -e "$git_public"
}

# git_register adds the key to the signing keys of gh's login where they lack it, titled with the
# Mac's name, so that GitHub verifies the commits it signs. Then it checks that they list it.
# Without a login or its write:ssh_signing_key scope, or on another error of gh's, it warns, at each
# run until one adds the key. Under --check, before gh or the key is there, it says what it would
# add once they are.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
git_register() {
    git_title=$(hostname -s)
    git_add="gh ssh-key add $git_pub --type signing --title $git_title"
    if [ ! -x "$gh" ]; then
        ws_say "would run $git_add once gh is installed, where it has a login that lacks the key"
        return 0
    fi
    ws_tmp
    git_err=$ws_tmpfile
    if ! git_listed && [ "$git_status" = 0 ]; then
        if [ -z "$git_public" ]; then
            ws_say "would run $git_add once the key is made"
        elif ws_checking; then
            ws_say "would run $git_add"
        else
            "$gh" ssh-key add "$git_pub" --type signing --title "$git_title" >/dev/null \
                2>"$git_err" || git_status=$?
            if [ "$git_status" = 0 ] && git_listed; then
                ws_say "ran $git_add"
            elif [ "$git_status" = 0 ]; then
                echo "gh api user/ssh_signing_keys lacks the key after $git_add" >"$git_err"
                git_status=1
            fi
        fi
    fi
    [ "$git_status" = 0 ] || git_warn
    rm -f "$git_err"
}

# git_warn warns that gh cannot register the key, and why, from git_status and git_err, in the
# devtools role's words. A 404 answers a login without the scope, and gh's last line of error loses
# its final period.
git_warn() {
    case $git_status:$(cat "$git_err") in
    4:*) git_why="gh has no login: gh auth login -h github.com -s write:ssh_signing_key logs in" ;;
    *'HTTP 404'*)
        git_why="gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h github.com"
        git_why="$git_why -s write:ssh_signing_key adds it"
        ;;
    *)
        git_why=$(tail -n 1 "$git_err")
        git_why=${git_why%.}
        git_why="gh failed: ${git_why:-exit $git_status}"
        ;;
    esac
    ws_say "warning: $git_why. Until a later run adds $git_pub, GitHub shows the commits it" \
        "signs as unverified"
}
