# shellcheck shell=sh
# The mac job's checks of git's wrapper for ws's agent, which mac-git-checks.sh sources last, so that
# they run with its functions, its files and its stand-in gh. The stand-in takes the job's YubiKey
# out, and puts it back.
#
# shellcheck disable=SC2154 # tmp, pub and the functions are mac-git-checks.sh's

# unsigned WHAT ARG... makes a commit in a new repository, with git's options ARG..., which must
# fail on the lack of the key in ws's agent. WHAT names the case.
unsigned() {
    what=$1
    shift
    rm -rf "$tmp/repo"
    git init -q "$tmp/repo"
    if git -C "$tmp/repo" "$@" commit -q --allow-empty -m ws 2>"$tmp/out.err"; then
        error "git signed a commit $what"
    elif ! grep -q "Couldn't find key in agent" "$tmp/out.err"; then
        error "git failed $what on another error than the agent's: $(cat "$tmp/out.err")"
    fi
    [ -z "$(git -C "$tmp/repo" rev-list --all 2>/dev/null)" ] || error "git made a commit $what"
}

# With the YubiKey out, ws's agent lists no key, and git must make no commit. Back in, git must sign
# again.
sh .github/yubikey-standin.sh unplug 30000001 || error "the stand-in kept 30000001 in"
unsigned 'with the YubiKey out' -c "user.signingKey=$pub"
sh .github/yubikey-standin.sh plug 30000001 || error "the stand-in left 30000001 out"
(SSH_AUTH_SOCK=$tmp/none && export SSH_AUTH_SOCK && signed && exit "$fail") || fail=1
# Nor may git sign with a key in a file beside its .pub, as ssh-keygen would without -U, though
# SSH_AUTH_SOCK names an agent of the case's own that holds it.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/file" </dev/null
agent=$(ssh-agent -s -a "$tmp/file.sock" | sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$agent" ] || ! SSH_AUTH_SOCK=$tmp/file.sock ssh-add -q "$tmp/file"; then
    error "no ssh-agent at $tmp/file.sock took the key of $tmp/file"
fi
(SSH_AUTH_SOCK=$tmp/file.sock && export SSH_AUTH_SOCK &&
    unsigned 'with a key in a file' -c "user.signingKey=$tmp/file.pub" && exit "$fail") || fail=1
kill "$agent" || true
agent=
