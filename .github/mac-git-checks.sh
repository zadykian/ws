# shellcheck shell=sh
# The mac job's checks of mac/setup.sh's git step, which mac-setup-checks.sh sources last, so that
# they run with its functions and the job's mac.conf. The runner must lack the files the step
# writes, which go as the script exits, on a signal too.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
key=$HOME/.ssh/id_ed25519_signing
files=$HOME/.config/ws

# git_made COMMAND... runs COMMAND with the files that git's cases make as its last arguments.
git_made() {
    "$@" "$key" "$key.pub" "$files/gitconfig" "$files/allowed_signers"
}

found=$(git_made ls -d 2>/dev/null) || true
if [ -n "$found" ]; then
    error "the runner has what git's cases would make: $(echo "$found" | tr '\n' ' ')"
    exit 1
fi
trap 'git_made rm -f; [ ! -f "$tmp/job.conf" ] || cp "$tmp/job.conf" "$conf"; rm -rf "$tmp"' EXIT

# With WS_GIT_NAME and WS_GIT_EMAIL, --check must say that it would make the signing key, show the
# files' diffs, and make none. A run must make the key, an ed25519 key with no passphrase, write the
# files and say so, and a second run and --check print nothing.
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com
setup --check || error "setup.sh --check failed: $out"
case $out in
"setup.sh: would make git's signing key, $key$nl"*"$nl+++ $files/gitconfig$nl"*) ;;
*) error "setup.sh --check showed no key or gitconfig: $out" ;;
esac
for each in "$key" "$files/gitconfig" "$files/allowed_signers"; do
    [ ! -e "$each" ] || error "setup.sh --check made $each"
done
prints "setup.sh: made git's signing key, $key${nl}setup.sh: wrote $files/allowed_signers${nl}\
setup.sh: wrote $files/gitconfig"
prints ''
prints '' --check
ssh-keygen -y -P '' -f "$key" | grep -q '^ssh-ed25519 ' ||
    error "$key is not an ed25519 key without a passphrase"
# A WS_GIT_SIGNING_KEY that is missing must stop setup.sh, as must a .pub that holds more than a
# public key's line, as a private key does. So must the default key's .pub alone, which a new key
# would replace, and its private key alone.
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com 'WS_GIT_SIGNING_KEY=~/.ssh/ws.pub'
stops "setup.sh: $HOME/.ssh/ws.pub, mac.conf's WS_GIT_SIGNING_KEY, is missing"
cp "$key" "$tmp/mixed.pub"
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com "WS_GIT_SIGNING_KEY=$tmp/mixed.pub"
stops "setup.sh: $tmp/mixed.pub holds no public key alone"
with 'WS_GIT_NAME="ws stand-in"' WS_GIT_EMAIL=ws@example.com
mv "$key" "$tmp/signing"
stops "setup.sh: $key.pub is there without its private key"
mv "$tmp/signing" "$key"
mv "$key.pub" "$tmp/signing.pub"
stops "setup.sh: $key.pub is missing beside its key"
mv "$tmp/signing.pub" "$key.pub"
