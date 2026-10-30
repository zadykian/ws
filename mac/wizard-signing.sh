# shellcheck shell=sh source-path=SCRIPTDIR
# The wizard's stage Signing key: ws's public signing key, which the remote script's status prints,
# among the signing keys of the Mac's gh login, so that GitHub shows ws's commits as verified. ws's
# own gh holds no token that may add it. wizard.sh sources it after wizard-gh.sh, and runs the stage
# after ws's run. It calls the functions of lib.sh, wizard-gh.sh and wizard-ws.sh, and reads
# wizard.sh's lock, and root_in of wizard-host.sh.

# signing_after prints ws's run, which makes the key, where that stage isn't done, then the stage
# gh on the Mac, whose login lists and adds the key.
signing_after() {
    if ! run_done; then
        echo "ws's run"
    elif ! gh_done; then
        echo "gh on the Mac"
    fi
}

# signing_done succeeds where root's git config on ws names no key file, or the Mac's gh lists the
# key among its login's signing keys. It sets signing_line to the key, as status prints it, and gh
# to the Mac's gh. It is not done without gh's login, which a check then names. It reads ws only
# where root logged in there in this run.
# shellcheck disable=SC2154 # root_in is wizard-host.sh's
signing_done() {
    [ "$root_in" = 1 ] && remote_read || return 1
    signing_line=$(remote_get signing)
    [ -n "$signing_line" ] || return 1
    [ "$signing_line" != none ] || return 0
    gh=$(command -v gh) || return 1
    "$gh" auth status -h github.com >/dev/null 2>&1 || return 1
    gh_api user/ssh_signing_keys || return 1
    printf '%s\n' "$gh_keys" | grep -qxF -e "$signing_line"
}

# signing_do adds the key to the Mac's gh login, titled with ws's hostname, from a file in the lock.
# --check says it would. gh_api stops on gh's errors other than a 404, which its login lacking the
# scope shows, as gh_do's stage mends first.
# shellcheck disable=SC2154 # lock is wizard.sh's
signing_do() {
    signing_title=$(remote_get hostname)
    signing_add="gh ssh-key add --type signing --title $signing_title"
    if ws_checking; then
        ws_say "Signing key: would run $signing_add for ws's signing key"
        return 0
    fi
    printf '%s\n' "$signing_line" >"$lock/signing.pub"
    "$gh" ssh-key add "$lock/signing.pub" --type signing --title "$signing_title" >/dev/null ||
        ws_die "Signing key: $signing_add failed"
    ws_say "Signing key: ran $signing_add for ws's signing key"
}

# signing_fail says what to do where the Mac's gh login still lacks ws's signing key.
signing_fail() {
    echo "the Mac's gh login still lacks ws's signing key, as above; mend what gh printed, then" \
        "run the wizard again"
}
