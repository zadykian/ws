# shellcheck shell=sh
# The mac job's checks of the key that mac/setup.sh's config.d/ws gives ws, which
# mac-setup-checks.sh sources before Gateway's, so that they run with its helpers and the job's
# mac.conf. They end with config.d/ws as the job's runs wrote it.
#
# shellcheck disable=SC2154 # tmp and the helpers are mac-setup-checks.sh's
ws=$HOME/.ssh/config.d/ws
cp "$ws" "$tmp/ws"

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
# write. --check must print the diff to it alone, and a second run and --check print nothing.
changes() {
    sed "$@" "$tmp/ws" >"$tmp/want"
    prints "$(diff -u -L "$ws" -L "$ws" "$ws" "$tmp/want")" --check
    prints "setup.sh: wrote $ws"
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
# Without them, config.d/ws must be the job's again.
job
