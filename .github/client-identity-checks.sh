# shellcheck shell=bash
# The client job's check of ws's login key in /etc/ws/ssh_config. The key is seward's
# ~/.ssh/ws-auth.pub, whatever values.yml names, and its agent serves it. The runner runs no sshd,
# so ssh -G is the proof. 56-keys.sh sources this file before git's.
#
# shellcheck disable=SC2154 # error is 56-keys.sh's
ws=/etc/ws/ssh_config

# ssh -G gives ws those three settings, a path from the home by ~/.
got=$(ssh -T -G ws | awk '$1 ~ /^identit(iesonly|yagent|yfile)$/' | sed "s|^\([a-z]*\) $HOME/|\1 ~/|")
want=$(printf '%s\n' 'identitiesonly yes' 'identityagent ~/.ssh/seward.sock' 'identityfile ~/.ssh/ws-auth.pub')
[ "$got" = "$want" ] || error "ssh -G ws gives '$got', not '$want'"
# The file holds them as three lines, in their order, with no file key's UseKeychain or AddKeysToAgent.
three=$(printf '    %s\n' 'IdentityFile ~/.ssh/ws-auth.pub' 'IdentitiesOnly yes' 'IdentityAgent "~/.ssh/seward.sock"')
[ "$(sed -n '/^    IdentityFile /,/^    IdentityAgent /p' "$ws")" = "$three" ] ||
    error "$ws gives the login key other lines than its three, in their order"
! grep -qE '^    (UseKeychain|AddKeysToAgent) ' "$ws" || error "$ws holds a file key's lines"
