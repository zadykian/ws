# git on the Mac

With `WS_GIT_NAME` and `WS_GIT_EMAIL` in `mac.conf`, `mac/setup.sh` sets up git on the Mac ([The
Mac](mac.md)), as the `shell` role does on ws ([Signing](shell.md#signing)). It writes
`~/.config/ws/gitconfig`: the author, and each commit signed with `WS_GIT_SIGNING_KEY`, which
`~/.config/ws/allowed_signers` trusts for the email in `git verify-commit`. Without the two keys,
the script leaves git alone.

Without `WS_GIT_SIGNING_KEY`, the key is `~/.ssh/id_ed25519_signing`, which the script makes once
with no passphrase, as git signs unattended, in a rebase too. Where its `.pub` is there without the
private key, as an agent's key leaves it, the script stops rather than replace it: name the `.pub`
in `WS_GIT_SIGNING_KEY`. A key that `WS_GIT_SIGNING_KEY` names must be there already, as the script
makes no other: `ssh-keygen -t ed25519 -N '' -f ~/.ssh/NAME` makes one. An empty
`WS_GIT_SIGNING_KEY` gives the script's own instead. A `.pub` that holds more than a public key's
line, as a private key does, stops the script too.

GitHub verifies a commit by the signing keys of its author's account, so add the key there once:
`gh ssh-key add ~/.ssh/id_ed25519_signing.pub --type signing`.
