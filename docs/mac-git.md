# git on the Mac

With `WS_GIT_NAME` and `WS_GIT_EMAIL` in `mac.conf`, `mac/setup.sh` sets up git on the Mac ([The
Mac](mac.md)), as the `shell` role does on ws ([Signing](shell.md#signing)). It writes
`~/.config/ws/gitconfig`: the author, and each commit signed with the YubiKey's key, or
`WS_GIT_SIGNING_KEY`, which `~/.config/ws/allowed_signers` trusts for the email in
`git verify-commit`. It names gh's credential command too, as `gh auth setup-git` does, so that
pushes to GitHub take gh's login, and installs Homebrew's gh where `PATH` has none. Without the two
keys, the script leaves git alone, and emptying them leaves the files and the include as they are.

Without `WS_GIT_SIGNING_KEY`, git signs with the YubiKey's key of `WS_GIT_KEY`, whose `.pub` is
`~/.config/ws/keys/NAME.pub` ([The YubiKey](mac-yubikey.md)). It signs through
`~/.config/ws/ws-ssh-keygen`, which gives `ssh-keygen` ws's agent, at `~/.ssh/ws-agent.sock`. So a
terminal, a JetBrains IDE and other apps sign alike, whatever agent they have. The wrapper takes
the agent's keys alone (`ssh-keygen -U`). With the YubiKey out, the agent lists none, and a commit
fails on `Couldn't find key in agent`. Nothing falls back to another key, nor to a key in a file
beside the `.pub`. Put the YubiKey back, and commit again. The file key that older runs made,
`~/.ssh/id_ed25519_signing`, signs no more, and stays in place.

A key that `WS_GIT_SIGNING_KEY` names must be there already, as the script makes none. A `.pub`
that the script cannot read, or that holds more than a public key's line, as a private key does,
stops it. A key of Secretive's, in
`~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys`, signs through
`~/.config/ws/secretive-ssh-keygen`, which gives `ssh-keygen` Secretive's agent.

`~/.gitconfig` includes `~/.config/ws/gitconfig` at its top, so that its own settings win where both
set one. Where the include is missing, the script adds it at the top, or makes the file, and keeps
the old file as `~/.gitconfig.TIME~`. It names the file's own settings of identity and signing,
which then win, such as a `user.email`. Where the include sits below the file's own settings, which
it then overrides, or comes twice, the script stops: keep one, at the top. Where `~/.gitconfig` is a
link that lacks the include, the script stops: put it at the top of the file the link points to.
Each stop comes before the script changes anything for git. git reads its XDG file,
`~/.config/git/config`, before `~/.gitconfig`, so `~/.config/ws/gitconfig` wins over the settings
there. The script names those of identity and signing as it adds the include.

GitHub verifies a commit by the signing keys of its author's account. Where those of gh's login lack
the key, the script adds it with `gh ssh-key add --type signing`, titled with the Mac's name. The
`devtools` role does the same on ws ([The signing key](devtools.md#the-signing-key)). The login
needs the `write:ssh_signing_key` scope. Without a login or the scope, or on another error of gh's,
the script says so at each run, and goes on. Until a run adds the key, GitHub shows the Mac's
commits as unverified, and the ruleset on `main` refuses them. `--check` reads the account's keys,
and says what it would add, also before the key is made. Before gh is installed, it says what it
would add once gh has a login. After an add, the script checks that they list the key.
