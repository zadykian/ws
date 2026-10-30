# shellcheck shell=sh
# The mac job's checks of the key that mac/setup.sh's config.d/ws gives ws, which
# mac-setup-checks.sh sources before Gateway's, so that they run with its functions and the job's
# mac.conf. They end with config.d/ws as the job's runs wrote it.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
ws=$HOME/.ssh/config.d/ws
cp "$ws" "$tmp/ws"
warned=

# keyed LINE... checks that ssh -G gives ws and ws-ssh LINE..., a line each, of its settings for the
# key. Those are identitiesonly, identityagent and identityfile, a path from the home by ~/.
keyed() {
    for host in ws ws-ssh; do
        got=$(ssh -G "$host" | awk '$1 ~ /^identit(iesonly|yagent|yfile)$/' |
            sed "s|^\([a-z]*\) $HOME/|\1 ~/|")
        [ "$got" = "$(printf '%s\n' "$@")" ] || error "ssh -G $host gives '$got', not '$*'"
    done
}

# changes SED... edits the job's config.d/ws with sed's arguments SED... into the file a run must
# write. --check must print the diff to it alone, and a second run and --check print nothing. A
# warning in warned, where set, must come first in --check and the run, and warned then empties.
changes() {
    sed "$@" "$tmp/ws" >"$tmp/want"
    prints "$warned$(diff -u -L "$ws" -L "$ws" "$ws" "$tmp/want")" --check
    prints "${warned}setup.sh: wrote $ws"
    warned=
    prints ''
    prints '' --check
    cmp -s "$ws" "$tmp/want" || error "setup.sh wrote another config.d/ws than --check showed"
}

# agented PUB SOCKET: changes, with the file key's lines replaced by those of the agent at SOCKET,
# which holds the key of PUB. Then ssh -G must give ws and ws-ssh that agent and PUB.
agented() {
    changes -e "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $1|" \
        -e '/^    UseKeychain yes$/d' -e "s|^    AddKeysToAgent yes\$|    IdentityAgent \"$2\"|"
    keyed 'identitiesonly yes' "identityagent $2" "identityfile $1"
}

# job: with the job's mac.conf, a run must give config.d/ws the job's lines again.
job() {
    with
    prints "setup.sh: wrote $ws"
    cmp -s "$ws" "$tmp/ws" || error "setup.sh left another config.d/ws than the job's"
}

# The job's runs had no WS_IDENTITY, so ws must take the file key, by the four lines it had before
# WS_IDENTITY, in their order.
keyed 'identitiesonly yes' 'identityfile ~/.ssh/id_ed25519'
four=$(printf '    %s\n' 'IdentityFile ~/.ssh/id_ed25519' 'IdentitiesOnly yes' 'UseKeychain yes' \
    'AddKeysToAgent yes')
[ "$(sed -n '/^    IdentityFile /,/^    AddKeysToAgent /p' "$ws")" = "$four" ] ||
    error "config.d/ws gives the file key other lines than its four, in their order"
# WS_IDENTITY must give ws the file beside the .pub in place of the file key. WS_IDENTITY_AGENT must
# give it the agent at its socket, by a path with a blank here.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/key"
with "WS_IDENTITY=$tmp/key.pub"
changes "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $tmp/key|"
keyed 'identitiesonly yes' "identityfile $tmp/key"
socket="$tmp/ws agent.sock"
with "WS_IDENTITY=$tmp/key.pub" "WS_IDENTITY_AGENT=\"$socket\""
agented "$tmp/key.pub" "$socket"
# A .pub that is missing, holds a private key or holds no key ssh-keygen reads must stop setup.sh.
# So must a certificate, and a line that is no key, which setup.sh must not print as a key's type.
with "WS_IDENTITY=$tmp/none.pub"
stops "setup.sh: $tmp/none.pub, mac.conf's WS_IDENTITY, is missing"
cp "$tmp/key" "$tmp/mixed.pub"
with "WS_IDENTITY=$tmp/mixed.pub"
stops "setup.sh: $tmp/mixed.pub holds no public key alone"
echo 'ssh-ed25519 AAAA ws' >"$tmp/bad.pub"
with "WS_IDENTITY=$tmp/bad.pub"
stops "setup.sh: ssh-keygen cannot read $tmp/bad.pub"
# So must one it cannot read, where the user cannot, as on the runner: root reads it.
cp "$tmp/key.pub" "$tmp/unread.pub" && chmod 000 "$tmp/unread.pub"
if ! cat "$tmp/unread.pub" >/dev/null 2>&1; then
    with "WS_IDENTITY=$tmp/unread.pub"
    stops "setup.sh: cannot read $tmp/unread.pub"
fi
ssh-keygen -q -s "$tmp/key" -I ws "$tmp/key.pub"
with "WS_IDENTITY=$tmp/key-cert.pub"
stops "setup.sh: $tmp/key-cert.pub is a certificate"
rm "$tmp/key-cert.pub"
for line in 'ws-token AAAA' 'ssh-ws stand-in!'; do
    echo "$line" >"$tmp/token.pub"
    with "WS_IDENTITY=$tmp/token.pub"
    stops "setup.sh: $tmp/token.pub holds no public key alone"
done
# Without them, config.d/ws must be the job's again.
job

# A stand-in for Secretive: ssh-agent at its socket under the runner's HOME, holding a P-256 key,
# whose .pub is in PublicKeys. The runner must lack Secretive's container, and the file key. As the
# script exits, on a signal too, the agent and the container go.
secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$secretive/Data
found=$(ls -d "$secretive" ~/.ssh/id_ed25519 2>/dev/null) || true
if [ -n "$found" ]; then
    error "the runner has $(echo "$found" | tr '\n' ' '), so the cases of Secretive's key go"
    exit 1
fi
# secretive_restore stops the stand-in agent, and removes Secretive's container.
secretive_restore() {
    [ -z "${secretive_agent-}" ] || kill "$secretive_agent" || true
    secretive_agent=
    rm -rf "$secretive"
}
trap 'secretive_restore; [ ! -f "$tmp/job.conf" ] || cp "$tmp/job.conf" "$conf"; rm -rf "$tmp"' EXIT
mkdir -p "$data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/p256"
cp "$tmp/p256.pub" "$data/PublicKeys/ws.pub"
# Before Secretive's agent has its socket, ws must keep the file key.
with
prints ''
secretive_agent=$(ssh-agent -s -a "$data/socket.ssh" |
    sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$secretive_agent" ] || ! SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$tmp/p256"; then
    error "no ssh-agent at Secretive's socket took the key"
    exit 1
fi
rm "$tmp/p256"
# So must it while PublicKeys has no key.
mv "$data/PublicKeys/ws.pub" "$tmp/ws.pub"
prints ''
mv "$tmp/ws.pub" "$data/PublicKeys/ws.pub"
# Without WS_IDENTITY, ws must take Secretive's agent and its one key, as a certificate is none, nor
# a name with a blank, nor a directory. setup.sh must warn as it replaces the file key. A second key
# must stop setup.sh, --check too, which must name both.
cp "$tmp/p256.pub" "$data/PublicKeys/ws-cert.pub"
cp "$tmp/p256.pub" "$data/PublicKeys/ws key.pub"
mkdir "$data/PublicKeys/ws-dir.pub"
sec=\~/${data#"$HOME/"}
warned="setup.sh: warning: Secretive's key replaces ~/.ssh/id_ed25519 as ws's: keep the file key"
warned="$warned with WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for an installed ws, or put the"
warned="$warned key of $sec/PublicKeys/ws.pub on GitHub before ws's next install$nl"
agented "$sec/PublicKeys/ws.pub" "$sec/socket.ssh"
# As the warning says, the file key's .pub in WS_IDENTITY keeps the file key, though the runner
# lacks it: config.d/ws must be the job's again (sed -n p changes nothing).
with 'WS_IDENTITY=~/.ssh/id_ed25519.pub'
changes -n p
with
cp "$tmp/key.pub" "$data/PublicKeys/other.pub"
several="setup.sh: Secretive's agent holds several keys, $data/PublicKeys/other.pub,"
several="$several $data/PublicKeys/ws.pub: name the one for ws in mac.conf's WS_IDENTITY"
stops "$several"
stops "$several" --check
# Where WS_GIT_SIGNING_KEY names one of the two, ws's key must be the other, as git's signing key
# does not count. ws_identity runs alone here, so that git's step does not.
got=$(WS_GIT_SIGNING_KEY=$sec/PublicKeys/other.pub sh -c '. mac/lib.sh && dir=mac &&
    . mac/setup-lib.sh && ws_identity && echo "$ws_identity_pub $ws_identity_agent"' 2>&1) || true
[ "$got" = "$sec/PublicKeys/ws.pub $sec/socket.ssh" ] ||
    error "ws_identity gives '$got', not ws.pub, where other.pub is WS_GIT_SIGNING_KEY"
# WS_IDENTITY wins over them: a .pub of its own, or one of Secretive's, which takes its agent unless
# WS_IDENTITY_AGENT names another.
with "WS_IDENTITY=$tmp/key.pub"
changes "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $tmp/key|"
with "WS_IDENTITY=$sec/PublicKeys/other.pub"
agented "$sec/PublicKeys/other.pub" "$sec/socket.ssh"
with "WS_IDENTITY=$sec/PublicKeys/other.pub" "WS_IDENTITY_AGENT=\"$socket\""
agented "$sec/PublicKeys/other.pub" "$socket"
# A key of a type that ws's OpenSSH lacks, as Secretive's ML-DSA, must stop setup.sh, --check too.
rm "$data/PublicKeys/other.pub"
echo 'ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws' >"$data/PublicKeys/ws.pub"
with
mldsa="setup.sh: $data/PublicKeys/ws.pub holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks"
stops "$mldsa"
stops "$mldsa" --check
# Without Secretive's agent, config.d/ws must be the job's again.
secretive_restore
job
