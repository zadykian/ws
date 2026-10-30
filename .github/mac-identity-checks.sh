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

# yubikey OLD: changes back to the job's config.d/ws (sed -n p changes nothing).
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

# The job's runs must give ws the YubiKey's key, ws, through ws's agent, by three lines in their
# order.
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
