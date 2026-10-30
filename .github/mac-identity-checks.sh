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

# The YubiKey's lines in the job's config.d/ws, as sed's patterns: its .pub, and ws's agent.
yk_file='^    IdentityFile ~/.config/ws/keys/ws.pub$'
yk_agent='^    IdentityAgent "~/.ssh/ws-agent.sock"$'

# filed KEY: changes, with the YubiKey's lines replaced by those of the file KEY, whose passphrase
# the Keychain keeps. Then ssh -G must give ws and ws-ssh KEY alone.
filed() {
    changes -e "s|$yk_file|    IdentityFile $1|" \
        -e "s|$yk_agent|    UseKeychain yes\\$nl    AddKeysToAgent yes|"
    keyed 'identitiesonly yes' "identityfile $1"
}

# yubikey OLD: changes back to the job's config.d/ws without WS_IDENTITY (sed -n p changes nothing).
# setup.sh must warn first that the YubiKey's key replaces OLD, which config.d/ws gave ws, and name
# the step that gives an installed ws the new key.
yubikey() {
    with
    warned="setup.sh: warning: the YubiKey's key, ~/.config/ws/keys/ws.pub, replaces $1 as ws's."
    warned="$warned The wizard's gh stage puts it on GitHub, whose keys a ws installed later"
    warned="$warned imports. An installed ws takes it by hand: add the key's line to root's"
    warned="$warned ~/.ssh/authorized_keys on ws, at ws's console or over ssh by the old key, as"
    warned="$warned docs/mac-yubikey.md shows, with no reinstall$nl"
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
# A config.d/ws of an older run, which gave ws the file key, must take the YubiKey's, with the
# warning.
sed -e "s|$yk_file|    IdentityFile ~/.ssh/id_ed25519|" \
    -e "s|$yk_agent|    UseKeychain yes\\$nl    AddKeysToAgent yes|" "$tmp/ws" >"$ws"
yubikey \~/.ssh/id_ed25519
# WS_IDENTITY must give ws the file beside the .pub in place of the YubiKey's key.
ssh-keygen -q -t ed25519 -N '' -C ws -f "$tmp/key"
with "WS_IDENTITY=$tmp/key.pub"
filed "$tmp/key"
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
# So must a key of a type that ws's OpenSSH lacks, as an ML-DSA key, --check too.
echo 'ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws' >"$tmp/mldsa.pub"
with "WS_IDENTITY=$tmp/mldsa.pub"
mldsa="setup.sh: $tmp/mldsa.pub holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks"
stops "$mldsa"
stops "$mldsa" --check
# The file key's .pub must give ws the file key, though the runner lacks it.
with 'WS_IDENTITY=~/.ssh/id_ed25519.pub'
filed \~/.ssh/id_ed25519
# Without WS_IDENTITY, ws must take the YubiKey's key again.
yubikey \~/.ssh/id_ed25519
