# shell

The `shell` role, Go steps in `internal/server/shell`, writes `~/.bashrc` with ble.sh and
bash-completion, `PATH` in `~/.profile`, and `~/.gitconfig`. Each run replaces any change made to
them by hand. A machine's own settings go in files they read where they exist:

- `~/.secrets/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too, and which the role writes from the secrets file:
  [Secrets](#secrets);
- `~/.bash_aliases`, which `~/.bashrc` sources too;
- `~/.config/git/local`, which `~/.gitconfig` includes last, so that its settings win.

A link at a file the role writes as root fails the run before a step changes anything. So does a
link at a directory between the home and that file ([Links under the home](security.md#links-under-the-home)).

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

`~/.gitconfig` has gh's credential helper already, as `gh auth setup-git` writes it, so running
that changes nothing. A dry run shows no diff of `~/.gitconfig`, as one set up by hand may hold a
credential. It names instead the keys a real run would drop, without their values: move them to
`~/.config/git/local` first. A subsection shows as `*`, as in `http.*.extraheader`, since one may
be a URL, which names a host. A real run keeps the old file beside the new one, as
`~/.gitconfig.*~`: its name ends in the run's PID and time.

## Signing

`~/.gitconfig` signs each commit with an SSH key of the machine's own: `ws_git_signing_key`
in the values file, `~/.ssh/id_ed25519_signing` by default. The role makes it once, an ed25519
key with no passphrase, as tools and claude commit unattended. The key only signs. The role does not
register it with GitHub: [The signing key](devtools.md#the-signing-key).

To revoke the key, delete it in the account's settings on GitHub, then delete
`~/.ssh/id_ed25519_signing` and its `.pub`. The next run makes a new key. The Mac's wizard
registers it, at its Signing key stage, or add the `.pub` by hand. A key deleted on GitHub alone
comes back the same way.

`git verify-commit` trusts the key for `shell_git_email` through `shell_git_allowed_signers`,
`~/.config/git/allowed_signers` by default. Each run replaces a change made to it, and keeps the
old file beside it, as `allowed_signers.*~`. So `git verify-commit` fails here on a commit another
key signed, an older machine's say, though GitHub verifies it by the account's keys.

A run replaces the signing key a `~/.gitconfig` set up by hand names, and says so: to keep that key,
set it in `~/.config/git/local` first. A signing setting there, such as `user.signingkey` or
`commit.gpgsign` or `tag.gpgsign`, wins over the role's, and the run names any it finds.

`shell_git_signing: false` turns it all off: no key and no signing in `~/.gitconfig`. A key
made before stays, on the machine and on GitHub. A dry run makes no key, so on a machine without
one it shows the key's creation and skips the allowed signers file.

## Secrets

The `[env]` table of [the secrets file](#the-secrets-file), empty by default, maps names to values
for `~/.secrets/env`.

With it set, the role makes `~/.secrets` 0700 and writes `env` 0600, both owned by `ws_user`.
The file opens with a `# Written by github.com/zadykian/ws` line, then `export NAME=VALUE` for
each name, the value quoted for the shell. Each run replaces the file, so a change goes in
`[env]`. The `otelcol` role reads the file later in the same run:
[claude's metrics elsewhere](otelcol.md#claudes-metrics-elsewhere).

- A name takes letters, digits and underscores, and starts with no digit. The run refuses any
  other, and its message names no value.
- A value is a string, so `"0123"` and `"on"` stay as written.
- An `env` without that line at its top, as one written by hand, fails the run before the role
  touches `~/.secrets`, in a dry run too. Move its values into `[env]`, then remove the file.
- The role neither prints nor diffs the file, so a dry run says only that it would change.

Without `--secrets`, the role leaves `~/.secrets` alone, even a file it wrote, so that a run
without the secrets file, as CI's, changes nothing. With `--secrets` and an empty `[env]`, the run
removes `~/.secrets/env`, if ws wrote it.

## The secrets file

`ws server apply --secrets FILE` takes the secrets of the roles from a TOML file that age encrypts,
in armor. `ws` decrypts it in its process with an X25519 identity. A run without `--secrets`, as
CI's, has no secrets. [The Mac's secrets](mac-secrets.md) says the file's tables, its references
and its values.

The identity stays in the Mac's Keychain, made by `ws mac secrets init`
([The identity](mac-secrets.md#the-identity)), or in ws's credential, made by
`ws server secrets init` ([ws's file](mac-secrets.md#wss-file)). `ws server apply` reads it from
the systemd credential `ws-secrets`, in `$CREDENTIALS_DIRECTORY`, for `--secrets` and `--check`
alike. Without the credential it fails and names `LoadCredential`. It reads no standard input and
no terminal. The Mac's wizard starts the run as a unit with `LoadCredential=ws-secrets:PATH`. By
hand, on a server, as root, with ws's file:

```sh
systemd-run --pty --wait -p LoadCredentialEncrypted=ws-secrets ws server apply --secrets /etc/ws/secrets.age
```

systemd decrypts the credential for the unit alone, and drops it once the run ends.

The roles read these tables, and ws refuses any other key:

- `[env]`: names to values, as [Secrets](#secrets) says;
- `[git."URL"]`: the entries of [Git credentials](#git-credentials);
- `[file]` and `[password]`: the ikeward profile's files and the `.p12`'s password
  ([The profile](ikeward.md#the-profile)).

No message, `--check` included, shows a value. Nothing plain reaches the disk but the
files that the roles write.

## Git credentials

The `[git]` table of [the secrets file](#the-secrets-file), empty by default, gives git a token for
each server besides GitHub. Its key is the server's URL, as `[git."https://git.example.com"]`, or a
path on it, which covers the repositories below it. Each entry takes:

- `token`: a personal access token, say;
- `username`: `token` by default, as such servers take any;
- `proactive_auth`: `true` for a server that sends no 401, so that git sends the token with its
  first request, by `http.<url>.proactiveAuth`.

A URL of `http://` fails the run, as git would send its token in clear. Each token goes in a 0600
file of `~/.config/git/tokens`, named for its URL. `~/.config/git/tokens.gitconfig`, which
`~/.gitconfig` includes, gets a helper for each URL, which answers git's request for a credential
from that file. The helper ignores git's requests to store or erase one, so a token the server
refuses stays until the entry changes. Where two URLs match, the longer one wins. gh's helper still
answers for GitHub, except below an entry's URL there.

With `--secrets`, a URL that the table lacks loses its token and its helper. Without it, as in
CI's run, the role leaves the tokens and `tokens.gitconfig` as they are. No token reaches
`tokens.gitconfig`, `~/.gitconfig` or the run's output, `-v` included: a dry run shows the diff of
the credential lines, and that a token would change, not which.

To revoke a token, revoke it on the server first, then remove its entry and run again.

## Clones

`shell_git_clones` lists repositories to clone once, as the machine's `host_vars.yml` sets it, and is
empty by default. Each entry takes:

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
`dest` with `*`, `?` or `[`, as the check for a missing `dest` could take it as a pattern.
