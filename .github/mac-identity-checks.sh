# shellcheck shell=sh
# The mac job's checks of the key that mac/setup.sh's config.d/ws gives ws, which
# mac-setup-checks.sh sources before Gateway's, so that they run with its functions and the job's
# mac.conf. They end with config.d/ws as the job's runs wrote it.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
ws=$HOME/.ssh/config.d/ws
cp "$ws" "$tmp/ws"
warned=
# The .pub of the key that the job's config.d/ws gives ws.
jobkey=\~/.config/ws/keys/ws.pub

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

# The YubiKey's lines in the job's config.d/ws, as sed's patterns: its .pub, and ws's agent.
yk_file='^    IdentityFile ~/.config/ws/keys/ws.pub$'
yk_agent='^    IdentityAgent "~/.ssh/ws-agent.sock"$'

# agented PUB SOCKET: changes, with the YubiKey's .pub and agent replaced by PUB and the agent at
# SOCKET. Then ssh -G must give ws and ws-ssh that agent and PUB.
agented() {
    changes -e "s|$yk_file|    IdentityFile $1|" -e "s|$yk_agent|    IdentityAgent \"$2\"|"
    keyed 'identitiesonly yes' "identityagent $2" "identityfile $1"
}

# filed KEY: changes, with the YubiKey's lines replaced by those of the file KEY, whose passphrase
# the Keychain keeps. Then ssh -G must give ws and ws-ssh KEY alone.
filed() {
    changes -e "s|$yk_file|    IdentityFile $1|" \
        -e "s|$yk_agent|    UseKeychain yes\\$nl    AddKeysToAgent yes|"
    keyed 'identitiesonly yes' "identityfile $1"
}

# job OLD: changes back to the job's config.d/ws without WS_IDENTITY (sed -n p changes nothing).
# setup.sh must warn first that the job's key replaces OLD, which config.d/ws gave ws.
job() {
    with
    warned="setup.sh: warning: the key of $jobkey replaces $1 as ws's: add that line to ws's"
    warned="$warned authorized_keys, or keep the old key with WS_IDENTITY=$1 in mac.conf$nl"
    changes -n p
}

# The job's runs had no WS_IDENTITY, so ws must take the YubiKey's key, ws, through ws's agent, by
# three lines in their order.
keyed 'identitiesonly yes' 'identityagent ~/.ssh/ws-agent.sock' \
    'identityfile ~/.config/ws/keys/ws.pub'
three=$(printf '    %s\n' 'IdentityFile ~/.config/ws/keys/ws.pub' 'IdentitiesOnly yes' \
    'IdentityAgent "~/.ssh/ws-agent.sock"')
[ "$(sed -n '/^    IdentityFile /,/^    IdentityAgent /p' "$ws")" = "$three" ] ||
    error "config.d/ws gives the YubiKey's key other lines than its three, in their order"
# WS_IDENTITY must give ws the file beside the .pub in place of the YubiKey's key. WS_IDENTITY_AGENT
# must give it the agent at its socket, by a path with a blank here.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/key"
with "WS_IDENTITY=$tmp/key.pub"
filed "$tmp/key"
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
job "$tmp/key.pub"

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
# Before Secretive's agent has its socket, ws must keep its key.
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
# Without WS_IDENTITY, Secretive's agent and its one key must leave ws its key too. The file key's
# .pub in WS_IDENTITY must give ws the file key, though the runner lacks it.
prints ''
sec=\~/${data#"$HOME/"}
with 'WS_IDENTITY=~/.ssh/id_ed25519.pub'
filed \~/.ssh/id_ed25519
cp "$tmp/key.pub" "$data/PublicKeys/other.pub"
# WS_IDENTITY names a .pub of its own, or one of Secretive's, which takes its agent unless
# WS_IDENTITY_AGENT names another.
with "WS_IDENTITY=$tmp/key.pub"
filed "$tmp/key"
with "WS_IDENTITY=$sec/PublicKeys/other.pub"
agented "$sec/PublicKeys/other.pub" "$sec/socket.ssh"
with "WS_IDENTITY=$sec/PublicKeys/other.pub" "WS_IDENTITY_AGENT=\"$socket\""
agented "$sec/PublicKeys/other.pub" "$socket"
# A key of a type that ws's OpenSSH lacks, as Secretive's ML-DSA, must stop setup.sh, --check too.
rm "$data/PublicKeys/other.pub"
echo 'ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws' >"$data/PublicKeys/ws.pub"
with "WS_IDENTITY=$sec/PublicKeys/ws.pub"
mldsa="setup.sh: $data/PublicKeys/ws.pub holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks"
stops "$mldsa"
stops "$mldsa" --check
# Without Secretive's agent, config.d/ws must be the job's again.
secretive_restore
job "$sec/PublicKeys/other.pub"
