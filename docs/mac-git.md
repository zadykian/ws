# git on the Mac

With `WS_GIT_NAME` and `WS_GIT_EMAIL` in `mac.conf`, `ws mac apply` sets up git on the Mac ([The
Mac](mac.md)), as the `shell` role does on ws ([Signing](shell.md#signing)). It writes
`~/.config/ws/gitconfig`: the author, and each commit signed with `WS_GIT_SIGNING_PUB_KEY`, which
`~/.config/ws/allowed_signers` trusts for the email in `git verify-commit`. It names gh's credential
command too, as `gh auth setup-git` does, so that pushes to GitHub take gh's login, and installs
Homebrew's gh where `PATH` has none. Without the two keys, the script leaves git alone, and emptying
them leaves the files and the include as they are.

Without `WS_GIT_SIGNING_PUB_KEY`, the key is `~/.ssh/id_ed25519_signing`, which the script makes
once with no passphrase, as git signs unattended, in a rebase too. Where its `.pub` is there without
the private key, as an agent's key leaves it, the script stops rather than replace it: name the
`.pub` in `WS_GIT_SIGNING_PUB_KEY`. A key that `WS_GIT_SIGNING_PUB_KEY` names must be there already,
as the script makes no other: `ssh-keygen -t ed25519 -N '' -f ~/.ssh/NAME` makes one. An empty
`WS_GIT_SIGNING_PUB_KEY` gives the script's own instead. A `.pub` that the script cannot read, or
that holds more than a public key's line, as a private key does, stops it too.

The wizard asks for git's key where `mac.conf` names none ([The Mac's
key](mac-wizard.md#the-macs-key)). A new one, made in Secretive's app at Notify, signs with no
prompt while the Mac is unlocked. The login key may sign too, but asks for Touch ID at each use by
default, so a rebase would ask at each commit. The wizard names ws's copy of its `.pub`,
`~/.config/ws/secretive/NAME.pub`, in `WS_GIT_SIGNING_PUB_KEY`. `NAME` is that of Secretive's Public
Key Path, and the script writes the copy from the agent ([Secretive's keys](#secretives-keys)). git
then signs through `~/.config/ws/secretive-ssh-keygen`, which gives `ssh-keygen` Secretive's agent,
so that a terminal, an editor and other apps sign alike. The key never leaves the Secure Enclave,
and signs only while the Mac is unlocked. ssh offers ws the key of `WS_IDENTITY_PUB_KEY` alone,
whichever keys Secretive holds ([ssh's hosts](mac.md#sshs-hosts)).

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
wizard does the same for ws ([Signing key](mac-wizard.md#signing-key)). The login
needs the `write:ssh_signing_key` scope. Without a login or the scope, or on another error of gh's,
the script says so at each run, and goes on. Until a run adds the key, GitHub shows the Mac's
commits as unverified, and the ruleset on `main` refuses them. `--check` reads the account's keys,
and says what it would add, also before the key is made. Before gh is installed, it says what it
would add once gh has a login. After an add, the script checks that they list the key.

## Secretive's keys

macOS 27 denies other apps the files of Secretive's container, with no prompt
([Secretive's issue](https://github.com/maxgoedjen/secretive/issues/839)). Its socket still answers,
so the script and the wizard ask the agent, with `ssh-add -L`. ssh and git need a key's `.pub` to
pick it from the agent, so they keep a copy in `~/.config/ws/secretive`, under Secretive's name for
it. No app then needs Full Disk Access. A `.pub` in `Data/PublicKeys` that `mac.conf` names becomes
that copy where the agent lists the key.

The socket stays after Secretive quits, and nothing answers there. The script then stops, and says
to open Secretive, unless `WS_IDENTITY_PUB_KEY` names a key.
