# git on the Mac

`mise -C /opt/ws/macos bootstrap` sets up git on the Mac by [git's units](units.toml.tmpl), as the `git` role
does on ws ([Signing](../../../ubuntu/roles/git/README.md#signing)). The facts give the files'
text, from the settings that `internal/gitcfg` holds for both sides, the fetch's prune among
them ([The facts](../../facts/README.md)). The author is `gitcfg.Mac()`, `zadykian` and
`me@zadykian.dev`, as ws's is `gitcfg.WS()`, the machine account.

## The units

- `file.git-config` writes `/etc/ws/gitconfig`: git's identity, the signing key, and gh as
  GitHub's credential command, as `gh auth setup-git` writes it, so that pushes to GitHub take gh's
  login.
- `file.git-include` puts the include of `/etc/ws/gitconfig` at the top of `~/.gitconfig`.
- `file.git-signers` writes `/etc/ws/allowed_signers`, which trusts seward's signing key for the
  email in `git verify-commit`. It comes after `state.ssh-sign-key` ([ssh on the
  Mac](../ssh/README.md#the-keys)), and reads the key's `.pub` as the user by the spec's
  `ssh_pub`. Before the `.pub` exists it holds the header alone.
- `file.git-keygen` writes `/etc/ws/ssh-keygen`, 755.
- `state.gh` copies gh into `/usr/local/bin`, root's and 755.

The files in `/etc/ws` are root's and 644, as on ws, bar the script's 755. A unit writes them
through sudo and reads them as the user, so that a bootstrap that changes nothing asks for no
password.

## gh

[`mise.macos.toml`](mise.macos.toml) pins gh, which mise installs from GitHub's release by its aqua
backend. [`mise.macos.lock`](../../mise/mise.macos.lock) holds the release's URL and SHA-256 for macOS
on arm64, and the bootstrap runs locked, so mise refuses a gh the lock lacks. mise installs it for
the tree alone, on no shell's `PATH`, so `state.gh` copies it to `/usr/local/bin/gh`, which
`/etc/paths` puts on every shell's, under a temporary name first. A gh there that differs is
replaced. To move gh on, change the pin and lock it again. The lock must hold the new release's URL
and SHA-256, as gh's release lists them. `TestMacLockHoldsEachTool` checks that the lock holds each
tool.

## The signing key

git signs with `~/.ssh/ws-sign`, the key that seward makes ([The keys](../ssh/README.md#the-keys)),
and names its `.pub`, `~/.ssh/ws-sign.pub`, as `user.signingkey`. No value names it, and the facts
read none: the key comes after them, from a unit. git then signs through `/etc/ws/ssh-keygen`,
which gives `ssh-keygen` seward's agent, `~/.ssh/seward.sock`, so that a terminal, an editor and
other apps sign alike, whatever `SSH_AUTH_SOCK` names. On a real Mac the key never leaves the
Secure Enclave, and signs only while the Mac is unlocked. A virtual machine holds the key as a
file. ssh offers ws `ws-auth` alone ([ssh's hosts](../ssh/README.md#sshs-hosts)).

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
