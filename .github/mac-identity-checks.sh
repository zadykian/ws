# shellcheck shell=sh
# The mac job's checks of the key that ws mac apply's config.d/ws gives ws, which
# mac-setup-checks.sh sources before Gateway's, so that they run with its functions and the job's
# mac.conf. They end with config.d/ws as the job's runs wrote it.
#
# shellcheck disable=SC2154 # tmp, conf and the functions are mac-setup-checks.sh's
ws=$HOME/.ssh/config.d/ws
cp "$ws" "$tmp/ws"
warned=

# keyed LINE... checks that ssh -G gives ws LINE..., a line each, of its settings for the key.
# Those are identitiesonly, identityagent and identityfile, a path from the home by ~/.
keyed() {
    got=$(ssh -G ws | awk '$1 ~ /^identit(iesonly|yagent|yfile)$/' |
        sed "s|^\([a-z]*\) $HOME/|\1 ~/|")
    [ "$got" = "$(printf '%s\n' "$@")" ] || error "ssh -G ws gives '$got', not '$*'"
}

# changes SED... edits the job's config.d/ws with sed's arguments SED... into the file a run must
# write. --check must print the diff to it alone, and a second run and --check print nothing. A
# warning in warned, where set, must come first in --check and the run, and warned then empties.
changes() {
    sed "$@" "$tmp/ws" >"$tmp/want"
    prints "$warned$(diff -u -L "$ws" -L "$ws" "$ws" "$tmp/want")" --check
    prints "${warned}ws mac apply: wrote $ws"
    warned=
    prints ''
    prints '' --check
    cmp -s "$ws" "$tmp/want" || error "ws mac apply wrote another config.d/ws than --check showed"
}

# agented PUB SOCKET: changes, with the file key's lines replaced by those of the agent at SOCKET,
# which holds the key of PUB. Then ssh -G must give ws that agent and PUB.
agented() {
    changes -e "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $1|" \
        -e '/^    UseKeychain yes$/d' -e "s|^    AddKeysToAgent yes\$|    IdentityAgent \"$2\"|"
    keyed 'identitiesonly yes' "identityagent $2" "identityfile $1"
}

# job: with the job's mac.conf, a run must give config.d/ws the job's lines again.
job() {
    with
    prints "ws mac apply: wrote $ws"
    cmp -s "$ws" "$tmp/ws" || error "ws mac apply left another config.d/ws than the job's"
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
# A .pub that is missing, holds a private key or holds no key ssh-keygen reads must stop ws mac apply.
# So must a certificate, and a line that is no key, which ws mac apply must not print as a key's type.
with "WS_IDENTITY=$tmp/none.pub"
stops "ws mac apply: $tmp/none.pub, mac.conf's WS_IDENTITY, is missing"
cp "$tmp/key" "$tmp/mixed.pub"
with "WS_IDENTITY=$tmp/mixed.pub"
stops "ws mac apply: $tmp/mixed.pub holds no public key alone"
echo 'ssh-ed25519 AAAA ws' >"$tmp/bad.pub"
with "WS_IDENTITY=$tmp/bad.pub"
stops "ws mac apply: ssh-keygen cannot read $tmp/bad.pub"
# So must one it cannot read, where the user cannot, as on the runner: root reads it.
cp "$tmp/key.pub" "$tmp/unread.pub" && chmod 000 "$tmp/unread.pub"
if ! cat "$tmp/unread.pub" >/dev/null 2>&1; then
    with "WS_IDENTITY=$tmp/unread.pub"
    stops "ws mac apply: cannot read $tmp/unread.pub"
fi
ssh-keygen -q -s "$tmp/key" -I ws "$tmp/key.pub"
with "WS_IDENTITY=$tmp/key-cert.pub"
stops "ws mac apply: $tmp/key-cert.pub is a certificate"
rm "$tmp/key-cert.pub"
for line in 'ws-token AAAA' 'ssh-ws stand-in!'; do
    echo "$line" >"$tmp/token.pub"
    with "WS_IDENTITY=$tmp/token.pub"
    stops "ws mac apply: $tmp/token.pub holds no public key alone"
done
# Without them, config.d/ws must be the job's again.
job

# A stand-in for Secretive: ssh-agent at its socket under the runner's HOME, holding a P-256 key.
# The runner must lack Secretive's container, and the file key. As the script exits, on a signal
# too, the agent, the container and ws's copies go.
secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$secretive/Data
found=$(ls -d "$secretive" ~/.ssh/id_ed25519 2>/dev/null) || true
if [ -n "$found" ]; then
    error "the runner has $(echo "$found" | tr '\n' ' '), so the cases of Secretive's key go"
    exit 1
fi
# secretive_restore stops the stand-in agent, and removes Secretive's container and ws's copies.
secretive_restore() {
    [ -z "${secretive_agent-}" ] || kill "$secretive_agent" || true
    secretive_agent=
    chmod -R u+rwx "$secretive" 2>/dev/null || true
    rm -rf "$secretive" "$HOME/.config/ws/secretive"
}
# pubname PUB prints the name that Secretive gives the .pub, and ws's copy keeps: its MD5 in hex.
pubname() {
    ssh-keygen -E md5 -l -f "$1" | sed -e 's/^[0-9]* MD5://' -e 's/ .*//' -e 's/://g' -e 's/$/.pub/'
}
trap 'secretive_restore; [ ! -f "$tmp/job.conf" ] || cp "$tmp/job.conf" "$conf"; rm -rf "$tmp"' EXIT
mkdir -p "$data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/p256"
# Before Secretive's agent has its socket, ws must keep the file key.
with
prints ''
secretive_agent=$(ssh-agent -s -a "$data/socket.ssh" |
    sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$secretive_agent" ]; then
    error "no ssh-agent at Secretive's socket"
    exit 1
fi
# So must it while the agent has no key.
prints ''
if ! SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$tmp/p256"; then
    error "ssh-agent at Secretive's socket took no key"
    exit 1
fi
rm "$tmp/p256"
# Without WS_IDENTITY, ws must take Secretive's agent and its one key, by ws's copy of its .pub.
# PublicKeys is closed, as macOS 27 denies ws the files of Secretive's container: ws must ask the
# agent. ws mac apply must warn as it replaces the file key, and write the copy where it is missing,
# as a run shows and --check says. A second key must stop ws mac apply, --check too, which must name
# both.
chmod 000 "$data/PublicKeys"
name=$(pubname "$tmp/p256.pub")
copy=$HOME/.config/ws/secretive/$name
cop=\~/.config/ws/secretive/$name
mkdir -p "${copy%/*}"
cp "$tmp/p256.pub" "$copy"
sec=\~/${data#"$HOME/"}
warned="ws mac apply: warning: Secretive's key replaces ~/.ssh/id_ed25519 as ws's: keep the file key"
warned="$warned with WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for an installed ws, or put the"
warned="$warned key of $cop on GitHub before ws's next install$nl"
agented "$cop" "$sec/socket.ssh"
rm "$copy"
prints "ws mac apply: would write $copy, ws's copy of a key of Secretive's" --check
[ ! -e "$copy" ] || error "ws mac apply --check wrote $copy"
prints "ws mac apply: wrote $copy"
cmp -s "$copy" "$tmp/p256.pub" || error "$copy is not the .pub of the agent's key"
prints ''
# A .pub in the container that mac.conf names, as a Mac before macOS 27 wrote, must become ws's copy.
with "WS_IDENTITY=$sec/PublicKeys/$name"
prints ''
with
chmod 755 "$data/PublicKeys"
# As the warning says, the file key's .pub in WS_IDENTITY keeps the file key, though the runner
# lacks it: config.d/ws must be the job's again (sed -n p changes nothing).
with 'WS_IDENTITY=~/.ssh/id_ed25519.pub'
changes -n p
with
SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$tmp/key" || error "the agent took no second key"
several="ws mac apply: Secretive's agent holds several keys, $copy,"
several="$several $HOME/.config/ws/secretive/$(pubname "$tmp/key.pub"):"
several="$several name the one for ws in mac.conf's WS_IDENTITY"
stops "$several"
stops "$several" --check
cp "$tmp/key.pub" "$data/PublicKeys/other.pub"
# WS_IDENTITY wins over them: a .pub of its own, or one of Secretive's, which takes its agent unless
# WS_IDENTITY_AGENT names another.
with "WS_IDENTITY=$tmp/key.pub"
changes "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $tmp/key|"
with "WS_IDENTITY=$sec/PublicKeys/other.pub"
agented "$sec/PublicKeys/other.pub" "$sec/socket.ssh"
with "WS_IDENTITY=$sec/PublicKeys/other.pub" "WS_IDENTITY_AGENT=\"$socket\""
agented "$sec/PublicKeys/other.pub" "$socket"
# A key of a type that ws's OpenSSH lacks, as Secretive's ML-DSA, must stop ws mac apply, --check too.
rm "$data/PublicKeys/other.pub"
echo 'ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws' >"$data/PublicKeys/ws.pub"
with "WS_IDENTITY=$sec/PublicKeys/ws.pub"
mldsa="ws mac apply: $data/PublicKeys/ws.pub holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks"
stops "$mldsa"
stops "$mldsa" --check
# Without Secretive's agent, config.d/ws must be the job's again.
secretive_restore
job
