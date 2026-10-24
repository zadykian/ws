# shell

The `shell` role writes `~/.bashrc` with ble.sh and bash-completion, `PATH` in `~/.profile`, and
`~/.gitconfig`. Each run replaces any change made to them by hand. A machine's own settings go in
files they read where they exist:

- `~/.secrets/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too, and which the role writes from a variable:
  [Secrets](#secrets);
- `~/.bash_aliases`, which `~/.bashrc` sources too;
- `~/.config/git/local`, which `~/.gitconfig` includes last, so that its settings win.

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

`~/.gitconfig` has gh's credential helper already, as `gh auth setup-git` writes it, so running
that changes nothing. A dry run shows no diff of `~/.gitconfig`, as one set up by hand may hold a
credential. It names instead the keys a real run would drop, without their values: move them to
`~/.config/git/local` first. A subsection shows as `*`, as in `http.*.extraheader`, since one may
be a URL, which names a host. A real run keeps the old file beside the new one, as
`~/.gitconfig.*~`.

## Signing

`~/.gitconfig` signs each commit with an SSH key of the machine's own: `ws_git_signing_key` in
`group_vars/all.yml`, `~/.ssh/id_ed25519_signing` by default. The role makes it once, an ed25519
key with no passphrase, as tools and claude commit unattended. The key only signs. The `devtools`
role registers it with GitHub: [The signing key](devtools.md#the-signing-key).

To revoke the key, delete it in the account's settings on GitHub, then delete
`~/.ssh/id_ed25519_signing` and its `.pub`. The next run makes a new key and registers it. A key
deleted on GitHub alone comes back with the next run, unless `shell_git_signing` is false.

`git verify-commit` trusts the key for `shell_git_email` through `shell_git_allowed_signers`,
`~/.config/git/allowed_signers` by default. Each run replaces a change made to it, and keeps the
old file beside it, as `allowed_signers.*~`. So `git verify-commit` fails here on a commit another
key signed, an older machine's say, though GitHub verifies it by the account's keys.

A run replaces the signing key a `~/.gitconfig` set up by hand names, and says so: to keep that key,
set it in `~/.config/git/local` first. A signing setting there, such as `user.signingkey` or
`commit.gpgsign`, wins over the role's, and the run names any it finds.

`shell_git_signing: false` turns it all off: no key, no signing in `~/.gitconfig`, and no
registration. A key made before stays, on the machine and on GitHub. A dry run makes no key, so on
a machine without one it shows the key's creation and skips the allowed signers file.

## Secrets

`shell_secret_env`, empty by default, maps names to values for `~/.secrets/env`. A private
repository passes it from its Ansible Vault, with `-e @/ABS/PATH/vault.yml` and
`--vault-password-file /dev/fd/3`. `bootstrap.sh` runs from the clone, with `/dev/null` as stdin,
hence the absolute path and the password on descriptor 3.

With it set, the role makes `~/.secrets` 0700 and writes `env` 0600, both owned by `ws_user`.
The file opens with a `# Written by github.com/zadykian/ws` line, then `export NAME=VALUE` for
each name, the value quoted for the shell. Each run replaces the file, so a change goes in the
variable. The `otelcol` role reads the file later in the same run:
[claude's metrics elsewhere](otelcol.md#claudes-metrics-elsewhere).

- A name takes capitals, digits and underscores, and starts with no digit. The run refuses any
  other, and its message names no value.
- A value is a string. YAML reads a bare `0123`, `12:30`, `on` or `~` as a number, a boolean or
  null, which the run refuses, naming the name alone: quote it, as `"0123"`.
- A value that holds `{{`, `{%` or `{#` takes YAML's `!unsafe` tag, or Ansible templates it.
- An `env` without that line at its top, as one written by hand, fails the run before the role
  touches `~/.secrets`, in a dry run too. Move its values into the variable, then remove the file.
- The role neither prints nor diffs the file, so a dry run says only that it would change.

Empty, the variable leaves `~/.secrets` alone, even a file the role wrote, so that a run
without the vault, as CI's, changes nothing. A null, as from a vault whose entries are all
commented out, counts as empty. To drop the file, remove it by hand.

## Git credentials

`shell_git_credentials`, empty by default, gives git a token for each server besides GitHub. A
private repository passes it in its extra vars, from the vault of `shell_secret_env`, say:
[Secrets](#secrets). Each entry takes:

- `url`: the server, as `https://git.example.com`, or a path on it, which covers the repositories
  below it;
- `token`: a personal access token, say;
- `username`: `token` by default, as such servers take any;
- `proactive_auth`: `true` for a server that sends no 401, so that git sends the token with its
  first request, by `http.<url>.proactiveAuth`;
- `state`: `present` by default, or `absent`, which removes the URL's token and helper, and needs
  no other key.

A URL of `http://` fails the run, as git would send its token in clear. Each token goes in a 0600
file of `~/.config/git/tokens`, named for its URL. `~/.config/git/tokens.gitconfig`, which
`~/.gitconfig` includes, gets a helper for each URL, which answers git's request for a credential
from that file. The helper ignores git's requests to store or erase one, so a token the server
refuses stays until the entry changes. Where two URLs match, the longer one wins. gh's helper still
answers for GitHub, except below an entry's URL there.

A run removes a token and its helper only where its entry says `state: absent`. A URL dropped from
the list keeps both. Empty, the list leaves the tokens and `tokens.gitconfig` as they are, so that
a run without the extra vars, as CI's, keeps them. No token reaches `tokens.gitconfig`,
`~/.gitconfig` or the run's output, `-v` and `--diff` included: a dry run shows the helpers' diff,
and that a token would change, not which.

To revoke a token, revoke it on the server first. Then set `state: absent` on its entry, and keep
the entry until each machine has run with it.

## Clones

`shell_git_clones` lists repositories to clone once, as a private repository's extra vars set it,
and is empty by default. Each entry takes:

- `repo`: the repository's URL or path, with no token, which [Git credentials](#git-credentials)
  give;
- `dest`: the clone's path, absolute or relative to `ws_home`.

Each repository is cloned as `ws_user`, after the tokens are written, where its `dest` is missing.
A run never pulls, resets or removes a clone, and leaves alone a `dest` that exists, clone or not.
A private server may sit behind the VPN of the `ikeward` role, which runs before `shell`
([A run](ikeward.md#a-run)). A clone that fails, as from a server out of reach, fails the run, and
the next run tries again.

A dry run lists each missing `dest` without reaching the server. A URL of `http://` or `https://`
that holds a user name or token fails the run, as the clone's config would keep it. So does a
`dest` with `*`, `?` or `[`, as the run's test for a missing `dest` takes it as a pattern.
