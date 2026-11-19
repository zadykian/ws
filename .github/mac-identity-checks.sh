# shellcheck shell=bash
# The mac job's checks of the key that mise bootstrap's /etc/ws/ssh_config gives ws. 56-keys.sh
# sources them before git's, so that they run with its functions and the job's values.yml. They end
# with /etc/ws/ssh_config as the job's runs wrote it.
#
# shellcheck disable=SC2154 # tmp, restores and the functions are 56-keys.sh's
ws=/etc/ws/ssh_config
cp "$ws" "$tmp/ws"

# keyed LINE... checks that ssh -G gives ws LINE..., a line each, of its settings for the key.
# Those are identitiesonly, identityagent and identityfile, a path from the home by ~/.
keyed() {
    got=$(ssh -T -G ws | awk '$1 ~ /^identit(iesonly|yagent|yfile)$/' |
        sed "s|^\([a-z]*\) $HOME/|\1 ~/|")
    [ "$got" = "$(printf '%s\n' "$@")" ] || error "ssh -G ws gives '$got', not '$*'"
}

# changes SED... edits the job's /etc/ws/ssh_config with sed's arguments SED... into the file that
# mise bootstrap must write, and runs it.
changes() {
    sed "$@" "$tmp/ws" >"$tmp/want"
    boots
    cmp -s "$ws" "$tmp/want" ||
        error "mise bootstrap wrote another /etc/ws/ssh_config: $(diff -u "$tmp/want" "$ws")"
}

# agented PUB SOCKET: changes, with the file key's lines replaced by those of the agent at SOCKET,
# which holds the key of PUB. Then ssh -G must give ws that agent and PUB.
agented() {
    changes -e "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $1|" \
        -e '/^    UseKeychain yes$/d' -e "s|^    AddKeysToAgent yes\$|    IdentityAgent \"$2\"|"
    keyed 'identitiesonly yes' "identityagent $2" "identityfile $1"
}

# job: with the job's values.yml, mise bootstrap must give /etc/ws/ssh_config the job's lines again.
job() {
    with
    changes -n p
}

# The job's values.yml names the file key ~/.ssh/id_ed25519, which ws must take by its four lines, in
# their order.
keyed 'identitiesonly yes' 'identityfile ~/.ssh/id_ed25519'
four=$(printf '    %s\n' 'IdentityFile ~/.ssh/id_ed25519' 'IdentitiesOnly yes' 'UseKeychain yes' \
    'AddKeysToAgent yes')
[ "$(sed -n '/^    IdentityFile /,/^    AddKeysToAgent /p' "$ws")" = "$four" ] ||
    error "/etc/ws/ssh_config gives the file key other lines than its four, in their order"
# WS_IDENTITY_PUB_KEY must give ws the file beside the .pub in place of the file key. WS_IDENTITY_AGENT must
# give it the agent at its socket, by a path with a blank here.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/key"
with "WS_IDENTITY_PUB_KEY: $tmp/key.pub"
changes "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $tmp/key|"
keyed 'identitiesonly yes' "identityfile $tmp/key"
socket="$tmp/ws agent.sock"
with "WS_IDENTITY_PUB_KEY: $tmp/key.pub" "WS_IDENTITY_AGENT: $socket"
agented "$tmp/key.pub" "$socket"
# A .pub that is missing, holds a private key or holds no key ssh-keygen reads must stop mise
# bootstrap. So must a certificate, and a line that is no key, whose type the stop must not name.
with "WS_IDENTITY_PUB_KEY: $tmp/none.pub"
refuses "$tmp/none.pub, values.yml's WS_IDENTITY_PUB_KEY, is missing"
cp "$tmp/key" "$tmp/mixed.pub"
with "WS_IDENTITY_PUB_KEY: $tmp/mixed.pub"
refuses "$tmp/mixed.pub holds no public key alone"
echo 'ssh-ed25519 AAAA ws' >"$tmp/bad.pub"
with "WS_IDENTITY_PUB_KEY: $tmp/bad.pub"
refuses "ssh-keygen cannot read $tmp/bad.pub"
# So must one the user cannot read, though root can, as on the runner: the facts read as the user.
cp "$tmp/key.pub" "$tmp/unread.pub" && chmod 000 "$tmp/unread.pub"
if ! cat "$tmp/unread.pub" >/dev/null 2>&1; then
    with "WS_IDENTITY_PUB_KEY: $tmp/unread.pub"
    refuses "cannot read $tmp/unread.pub"
fi
ssh-keygen -q -s "$tmp/key" -I ws -n ws "$tmp/key.pub"
with "WS_IDENTITY_PUB_KEY: $tmp/key-cert.pub"
refuses "$tmp/key-cert.pub is a certificate"
rm "$tmp/key-cert.pub"
for line in 'ws-token AAAA' 'ssh-ws stand-in!'; do
    echo "$line" >"$tmp/token.pub"
    with "WS_IDENTITY_PUB_KEY: $tmp/token.pub"
    refuses "$tmp/token.pub holds no public key alone"
done
# Without them, /etc/ws/ssh_config must be the job's again.
job

# A stand-in for Secretive: ssh-agent at its socket under the runner's HOME, holding a P-256 key.
# The runner must lack Secretive's container. As the case exits, on a signal too, the agent, the
# container and ws's copies go.
secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent
data=$secretive/Data
if [ -e "$secretive" ]; then
    error "the runner has $secretive, so the cases of Secretive's key go"
    exit 1
fi
# secretive_restore stops the stand-in agent, and removes Secretive's container and ws's copies.
secretive_restore() {
    [ -z "${secretive_agent-}" ] || kill "$secretive_agent" || true
    secretive_agent=
    chmod -R u+rwx "$secretive" 2>/dev/null || true
    rm -rf "$secretive"
    sudo rm -rf /etc/ws/secretive
}
restores="$restores secretive_restore"
# pubname PUB prints the name that Secretive gives the .pub, and ws's copy keeps: its MD5 in hex.
pubname() {
    ssh-keygen -E md5 -l -f "$1" | sed -e 's/^[0-9]* MD5://' -e 's/ .*//' -e 's/://g' -e 's/$/.pub/'
}
mkdir -p "$data/PublicKeys"
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/p256"
# Before Secretive's agent has its socket, ws must keep the file key.
job
secretive_agent=$(ssh-agent -s -a "$data/socket.ssh" |
    sed -n 's/^SSH_AGENT_PID=\([0-9]*\);.*/\1/p')
if [ -z "$secretive_agent" ]; then
    error "no ssh-agent at Secretive's socket"
    exit 1
fi
# So must it while the agent has no key.
job
if ! SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$tmp/p256"; then
    error "ssh-agent at Secretive's socket took no key"
    exit 1
fi
rm "$tmp/p256"
# An empty WS_IDENTITY_PUB_KEY must stop mise bootstrap, though the agent holds one key: ws takes no
# key that values.yml leaves out. The stop must give the line that names the key, by ws's copy of
# its .pub, with its comment and fingerprint. PublicKeys is closed, as macOS 27 denies ws the files
# of Secretive's container: ws must ask the agent. With the line, ws must take the agent and the
# key, and the keys' unit must write the copy where ws lacks it.
chmod 000 "$data/PublicKeys"
name=$(pubname "$tmp/p256.pub")
copy=/etc/ws/secretive/$name
# unnamed KEY USE PUB... prints the stop where values.yml names no KEY, the key for USE. It gives
# the line that names the key of each .pub of PUB..., in their order, by the comment ws.
unnamed() {
    printf "values.yml names no %s, %s: add one of these lines to values.yml, " "$1" "$2"
    unnamed_key=$1 unnamed_sep=
    shift 2
    for pub in "$@"; do
        printf '%s%s: /etc/ws/secretive/%s for ws (%s)' "$unnamed_sep" "$unnamed_key" \
            "$(pubname "$pub")" "$(ssh-keygen -l -f "$pub" | awk '{ print $2 }')"
        unnamed_sep='; '
    done
}
with 'WS_IDENTITY_PUB_KEY: ""'
refuses "$(unnamed WS_IDENTITY_PUB_KEY "ws's login key" "$tmp/p256.pub")"
sec=\~/${data#"$HOME/"}
with "WS_IDENTITY_PUB_KEY: $copy"
agented "$copy" "$sec/socket.ssh"
cmp -s "$copy" "$tmp/p256.pub" || error "$copy is not the .pub of the agent's key"
# Where the copy goes, the plan must show a change and write nothing, and a run must write it back.
sudo rm "$copy"
status=0
mac_bootstrap plan --detailed-exitcode >/dev/null 2>&1 || status=$?
[ "$status" = 2 ] || error "the plan without $copy exited $status, not 2"
[ ! -e "$copy" ] || error "mise bootstrap's plan wrote $copy"
boots
cmp -s "$copy" "$tmp/p256.pub" || error "mise bootstrap did not write $copy back"
# A .pub in the container that values.yml names, as a Mac before macOS 27 wrote, must become ws's copy.
with "WS_IDENTITY_PUB_KEY: $sec/PublicKeys/$name"
boots
cmp -s "$ws" "$tmp/want" || error "mise bootstrap took the container's .pub other than as ws's copy"
chmod 755 "$data/PublicKeys"
# The job's file key, named beside the agent, must give /etc/ws/ssh_config the job's lines again.
job
# With more keys in the agent, the stop must give a line for each P-256 key, in the agent's order.
# The ed25519 key, which Secretive never makes, must get none.
ssh-keygen -q -t ecdsa -b 256 -N '' -C ws -f "$tmp/second"
SSH_AUTH_SOCK=$data/socket.ssh ssh-add -q "$tmp/key" "$tmp/second" || error "the agent took no more keys"
with 'WS_IDENTITY_PUB_KEY: ""'
refuses "$(unnamed WS_IDENTITY_PUB_KEY "ws's login key" "$tmp/p256.pub" "$tmp/second.pub")"
cp "$tmp/key.pub" "$data/PublicKeys/other.pub"
# Whatever the agent holds, WS_IDENTITY_PUB_KEY names ws's key: a .pub of its own, or one of
# Secretive's, which takes its agent unless WS_IDENTITY_AGENT names another.
with "WS_IDENTITY_PUB_KEY: $tmp/key.pub"
changes "s|^    IdentityFile ~/.ssh/id_ed25519\$|    IdentityFile $tmp/key|"
with "WS_IDENTITY_PUB_KEY: $sec/PublicKeys/other.pub"
agented "$sec/PublicKeys/other.pub" "$sec/socket.ssh"
with "WS_IDENTITY_PUB_KEY: $sec/PublicKeys/other.pub" "WS_IDENTITY_AGENT: $socket"
agented "$sec/PublicKeys/other.pub" "$socket"
# A key of a type that ws's OpenSSH lacks, as Secretive's ML-DSA, must stop mise bootstrap.
rm "$data/PublicKeys/other.pub"
echo 'ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws' >"$data/PublicKeys/ws.pub"
with "WS_IDENTITY_PUB_KEY: $sec/PublicKeys/ws.pub"
refuses "$data/PublicKeys/ws.pub holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks"
# Without Secretive's agent, /etc/ws/ssh_config must be the job's again.
secretive_restore
job
