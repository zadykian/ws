# git on the Mac

With `WS_GIT_NAME` and `WS_GIT_EMAIL` in `mac.conf`, `mac/setup.sh` sets up git on the Mac ([The
Mac](mac.md)), as the `shell` role does on ws ([Signing](shell.md#signing)). It writes
`~/.config/ws/gitconfig`: the author, and each commit signed with `WS_GIT_SIGNING_KEY`, which
`~/.config/ws/allowed_signers` trusts for the email in `git verify-commit`. Without the two keys,
the script leaves git alone, and emptying them leaves the files and the include as they are.

Without `WS_GIT_SIGNING_KEY`, the key is `~/.ssh/id_ed25519_signing`, which the script makes once
with no passphrase, as git signs unattended, in a rebase too. Where its `.pub` is there without the
private key, as an agent's key leaves it, the script stops rather than replace it: name the `.pub`
in `WS_GIT_SIGNING_KEY`. A key that `WS_GIT_SIGNING_KEY` names must be there already, as the script
makes no other: `ssh-keygen -t ed25519 -N '' -f ~/.ssh/NAME` makes one. An empty
`WS_GIT_SIGNING_KEY` gives the script's own instead. A `.pub` that holds more than a public key's
line, as a private key does, stops the script too.

Where Secretive holds the Mac's ssh key, give git a key of its own in Secretive's app, at Notify.
The login key asks for Touch ID at each use by default, so a rebase would ask at each commit. Name
its `.pub`, in `~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys`, in
`WS_GIT_SIGNING_KEY`. git then signs through `~/.config/ws/secretive-ssh-keygen`, which gives
`ssh-keygen` Secretive's agent, so that a terminal, a JetBrains IDE and other apps sign alike. The
key never leaves the Secure Enclave, and signs only while the Mac is unlocked.

`~/.gitconfig` includes `~/.config/ws/gitconfig` at its top, so that its own settings win where both
set one. Where the include is missing, the script adds it at the top, or makes the file, and keeps
the old file as `~/.gitconfig.TIME~`. It names the file's own settings of identity and signing,
which then win, such as a `user.email`. Where the include sits below the file's own settings, which
it then overrides, or comes twice, the script stops: keep one, at the top. Where `~/.gitconfig` is a
link that lacks the include, the script stops: put it at the top of the file the link points to.
Each stop comes before the script changes anything for git. git reads its XDG file,
`~/.config/git/config`, before `~/.gitconfig`, so `~/.config/ws/gitconfig` wins over the settings
there. The script names those of identity and signing as it adds the include.

GitHub verifies a commit by the signing keys of its author's account, so add the key there once:
`gh ssh-key add ~/.ssh/id_ed25519_signing.pub --type signing`.
