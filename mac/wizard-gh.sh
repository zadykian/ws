# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's stage gh on the Mac: GitHub's CLI, from Homebrew where missing, logged in to
# github.com with the scopes that add the Mac's keys. Its login's keys must hold the Mac's key for
# ws, which Ubuntu's installer imports for ws's user. setup.sh, which runs after, adds git's signing
# key with that login. wizard.sh sources it after wizard-keys.sh, and runs the stage after the Mac
# key. It calls the functions of lib.sh, setup-lib.sh, wizard-keys.sh and setup-git.sh, and reads
# wizard.sh's lock and WS_CONF, mac.conf's path.

# gh's install is setup-git.sh's git_gh, sourced from dir, mac/'s path, as wizard.sh sets it.
# shellcheck source=setup-git.sh disable=SC2154 # dir is wizard.sh's
. "$dir/setup-git.sh"

# The scopes of gh's login that gh ssh-key add needs, for a key that logs in and for one that signs.
gh_scopes=write:public_key,write:ssh_signing_key

# gh_after prints Tools, bootstrap.sh's stage, where PATH has neither gh nor brew to install it.
gh_after() {
    command -v gh >/dev/null || command -v brew >/dev/null || echo Tools
}

# gh_done succeeds where gh, logged in to github.com, lists the Mac's key for ws among its login's,
# and can list its signing keys, to which setup.sh adds git's.
gh_done() {
    command -v gh >/dev/null && gh auth status -h github.com >/dev/null 2>&1 && gh_key &&
        gh api --paginate user/keys --jq '.[].key' 2>/dev/null | grep -qxF -e "$gh_line" &&
        gh api --paginate user/ssh_signing_keys --jq '.[].key' >/dev/null 2>&1
}

# gh_key sets gh_pub to the .pub of the Mac's key for ws, as ws_identity finds it from mac.conf, and
# gh_line to its type and base64. It fails where the .pub is missing.
# shellcheck disable=SC2154 # ws_identity sets ws_identity_pub
gh_key() {
    gh_pub=$( (ws_conf "$WS_CONF" && ws_identity && keys_home "$ws_identity_pub") 2>/dev/null) &&
        [ -f "$gh_pub" ] && gh_line=$(awk 'NF { print $1, $2; exit }' "$gh_pub") &&
        [ -n "$gh_line" ]
}

# gh_do installs gh where missing, and logs it in where it has no login. It adds the Mac's key for
# ws to its login's keys where they lack it, titled with the Mac's name. --check says what it would
# run, and reads no keys once it would log in.
gh_do() {
    git_gh "gh on the Mac"
    gh_title=$(hostname -s)
    gh_where=
    if [ ! -x "$gh" ] || ! "$gh" auth status -h github.com >/dev/null 2>&1; then
        gh_web login -p https -w
        gh_where=", where the login lacks the key"
    fi
    if ! gh_key; then
        ws_checking || return 1
        ws_say "gh on the Mac: would run gh ssh-key add for the Mac's key, once it is made"
        return 0
    fi
    gh_add="gh ssh-key add $gh_pub --title $gh_title"
    if [ -z "$gh_where" ] || ! ws_checking; then
        gh_listed && return 0
    fi
    if ws_checking; then
        ws_say "gh on the Mac: would run $gh_add$gh_where"
        return 0
    fi
    "$gh" ssh-key add "$gh_pub" --title "$gh_title" >/dev/null ||
        ws_die "gh on the Mac: $gh_add failed"
    ws_say "gh on the Mac: ran $gh_add"
}

# gh_listed succeeds where the keys of gh's login hold the Mac's key. Where the login lacks a scope,
# as a 404 from its keys or its signing keys shows, it refreshes the scopes first. --check then
# reads no more, and goes by the keys it read. gh's other errors stop it.
gh_listed() {
    gh_signs=1
    gh_api user/ssh_signing_keys || gh_signs=
    if ! gh_api user/keys || [ -z "$gh_signs" ]; then
        gh_web refresh
        gh_where=", where the login lacks the key"
        ws_checking || gh_api user/keys ||
            ws_die "gh on the Mac: gh api user/keys failed after gh auth refresh"
    fi
    printf '%s\n' "$gh_keys" | grep -qxF -e "$gh_line"
}

# gh_api PATH sets gh_keys to the keys that gh's login lists at PATH, as user/keys. It fails where
# the login lacks their scope, as a 404 shows, and stops on gh's other errors.
# shellcheck disable=SC2154 # lock is wizard.sh's
gh_api() {
    gh_keys=$("$gh" api --paginate "$1" --jq '.[].key' 2>"$lock/gh.err") && return 0
    grep -q 'HTTP 404' "$lock/gh.err" || ws_die "gh on the Mac: gh api $1 failed:" \
        "$(tail -n 1 "$lock/gh.err")"
    return 1
}

# gh_web ACTION ARG... runs gh auth ACTION for github.com with ARGs and the scopes. Its device flow
# shows a code to enter in the browser, at github.com/login/device. --check says so. With no
# terminal, it stops, naming the command.
gh_web() {
    gh_run="gh auth $* -h github.com -s $gh_scopes"
    if ws_checking; then
        ws_say "gh on the Mac: would run $gh_run"
        return 0
    fi
    [ -t 0 ] || ws_die "gh on the Mac: gh lacks a login to github.com with the scopes of the" \
        "Mac's keys, and no terminal is there for $gh_run: run it, then the wizard again"
    "$gh" auth "$@" -h github.com -s "$gh_scopes" >&2 || ws_die "gh on the Mac: $gh_run failed"
}

# gh_fail says what to do where gh's login still lacks the Mac's key.
gh_fail() {
    echo "gh's login lacks the Mac's key for ws, as above; run the wizard again"
}
