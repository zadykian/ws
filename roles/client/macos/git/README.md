# git on the Mac

`mise -C /opt/ws/roles/client bootstrap` sets up git on the Mac by [git's units](units.toml.tmpl), as the `git` role
does on ws ([Signing](../../../server/ubuntu/git/README.md#signing)). The facts give the files'
text, from the settings that `roles/internal/gitcfg` holds for both sides, the fetch's prune among
them ([The facts](../../facts/README.md)). The author is `gitcfg.Mac()`, `zadykian` and
`me@zadykian.dev`, as ws's is `gitcfg.WS()`, the machine account.

## The units

- `file.git-config` writes `/etc/ws/gitconfig`: git's identity, the signing key, and gh as
  GitHub's credential command, as `gh auth setup-git` writes it, so that pushes to GitHub take gh's
  login. It comes after the keys' copies (`state.keys`, [ssh on the Mac](../ssh/README.md)).
- `file.git-include` puts the include of `/etc/ws/gitconfig` at the top of `~/.gitconfig`.
- `file.git-signers` writes `/etc/ws/allowed_signers`, which trusts `WS_GIT_SIGNING_PUB_KEY` for
  the email in `git verify-commit`.
- `file.git-keygen` writes `/etc/ws/secretive-ssh-keygen`, 755, for a key of Secretive's, and
  removes it for a file key.
- `state.gh` copies gh into `/usr/local/bin`, root's and 755.

The files in `/etc/ws` are root's and 644, as on ws, bar the script's 755. A unit writes them
through sudo and reads them as the user, so that a bootstrap that changes nothing asks for no
password.

## gh

[`mise.mac.toml`](mise.mac.toml) pins gh, which mise installs from GitHub's release by its aqua
backend. [`mise.mac.lock`](../../mise/mise.mac.lock) holds the release's URL and SHA-256 for macOS
on arm64, and the bootstrap runs locked, so mise refuses a gh the lock lacks. mise installs it for
the tree alone, on no shell's `PATH`, so `state.gh` copies it to `/usr/local/bin/gh`, which
`/etc/paths` puts on every shell's, under a temporary name first. A gh there that differs is
replaced. To move gh on, change the pin and lock it again. The lock must hold the new release's URL
and SHA-256, as gh's release lists them. `TestMacLockHoldsEachTool` checks that the lock holds each
tool.

## The signing key

`WS_GIT_SIGNING_PUB_KEY` has no default, and ws makes no key. Where `values.yml` names none, or an
empty one, the bootstrap stops with the line that would name each P-256 key of Secretive's agent. A
`.pub` that is missing or that the user cannot read stops it too, as does one that holds more than
a public key's line, as a private key does. So does a certificate, or a key of a type that ws's
OpenSSH lacks: git takes the seven that ssh takes, as GitHub takes no other signing key. A `.pub`
of Secretive's that ws cannot read, where its agent lists no key of it, stops it with the word to
open Secretive. A file key with no passphrase signs unattended, in a rebase too: `ssh-keygen -t
ed25519 -N '' -f ~/.ssh/NAME` makes one.

`ws client setup` names the key that comes during its wait as `WS_GIT_SIGNING_PUB_KEY` too, as it
names it `WS_IDENTITY_PUB_KEY` ([Secretive](../ssh/README.md#secretive)). A key made in Secretive's
app at Notify signs with no prompt while the Mac is unlocked. A key that asks for Touch ID at each
use would ask at each commit of a rebase. To sign with another
key, name ws's copy of its `.pub`, `/etc/ws/secretive/NAME.pub`, in `WS_GIT_SIGNING_PUB_KEY`.
`NAME` is that of Secretive's Public Key Path, and the keys' unit writes the copy from the agent
([Secretive's keys](#secretives-keys)). git then signs through `/etc/ws/secretive-ssh-keygen`,
which gives `ssh-keygen` Secretive's agent, so that a terminal, an editor and other apps sign
alike. The key never leaves the Secure Enclave, and signs only while the Mac is unlocked. ssh
offers ws the key of `WS_IDENTITY_PUB_KEY` alone, whichever keys Secretive holds ([ssh's
hosts](../ssh/README.md#sshs-hosts)).

GitHub verifies a commit by the signing keys of its author's account. ws adds no key there: add it
once with `gh ssh-key add --type signing`, as [the runbook](../../runbook.md#github) says. Until then,
GitHub shows the Mac's commits as unverified, and the ruleset on `main` refuses them.

## ~/.gitconfig

`~/.gitconfig` includes `/etc/ws/gitconfig` at its top, as on ws, so that its own settings win
where both set one, such as a `user.email`. Where the include is missing, the unit adds it at the
top, or makes the file, and keeps the old file as `~/.gitconfig.TIME~`. Where the include sits below
the file's own settings, which it then overrides, or comes twice, the unit stops and leaves the
file alone: keep one, at the top. Where `~/.gitconfig` is a link that lacks the include, the unit
stops: put it at the top of the file the link points to. git reads its XDG file,
`~/.config/git/config`, before `~/.gitconfig`, so `/etc/ws/gitconfig` wins over the settings
there. ws names none of the settings that win over its own, or lose to them.

## Secretive's keys

macOS 27 denies other apps the files of Secretive's container, with no prompt ([Secretive's
issue](https://github.com/maxgoedjen/secretive/issues/839)). Its socket still answers, so ws asks
the agent, with `ssh-add -L`. ssh and git need a key's `.pub` to pick it from the agent, so they
keep a copy in `/etc/ws/secretive`, under Secretive's name for it, which `ws client keys`
writes. No app then needs Full Disk Access. A `.pub` in `Data/PublicKeys` that `values.yml` names
becomes that copy where the agent lists the key.

The socket stays after Secretive quits, and nothing answers there. The bootstrap then stops, and
says to open Secretive, unless `WS_IDENTITY_PUB_KEY` names a key.
