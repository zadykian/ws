# shell

The `shell` role, Go steps in `roles/server/ubuntu/shell`, writes `~/.bashrc` with ble.sh and
bash-completion, and `PATH` in `~/.profile`. Each run replaces any change made to them by hand. A
machine's own settings go in files they read where they exist:

- `/etc/ws/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too, and which the role writes from the secrets file:
  [Secrets](#secrets);
- `~/.bash_aliases`, which `~/.bashrc` sources too.

A link at a file the role writes as root fails the step of that file alone. So does a link at a
directory between the home and that file ([Links under the home](../security/README.md#links-under-the-home)).

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

## Secrets

The `[env]` table of [the secrets file](#the-secrets-file), empty by default, maps names to values
for `/etc/ws/env`.

With it set, the role writes `/etc/ws/env` 0640, root's with the account's group: `WS_USER` reads it
and cannot change it.
The file opens with a `# Written by github.com/zadykian/ws` line, then `export NAME=VALUE` for
each name, the value quoted for the shell. Each run replaces the file, so a change goes in
`[env]`. The `otelcol` role reads the file later in the same run:
[claude's metrics elsewhere](../otelcol/README.md#claudes-metrics-elsewhere).

- A name takes letters, digits and underscores, and starts with no digit. The run refuses any
  other, and its message names no value.
- A value is a string, so `"0123"` and `"on"` stay as written.
- An `env` without that line at its top, as one written by hand, fails the run before the role
  touches it, in a dry run too. Move its values into `[env]`, then remove the file.
- The role neither prints nor diffs the file, so a dry run says only that it would change.

Without `--secrets`, the role leaves `/etc/ws/env` alone, even a file it wrote, so that a run
without the secrets file, as CI's, changes nothing. With `--secrets` and an empty `[env]`, the run
removes `/etc/ws/env`, if ws wrote it.

## The secrets file

`ws server ubuntu apply --secrets FILE` takes the secrets of the roles from a TOML file that age
encrypts, in armor. `ws` decrypts it in its process with an X25519 identity. A run without
`--secrets`, as CI's, has no secrets. [ws's secrets](../secrets/README.md) says the file's tables, its
references and its values.

The identity stays in ws's credential, made by `ws server ubuntu secrets init`, or by `ws client
macos secrets init` from the Mac ([ws's file](../secrets/README.md#wss-file)). `ws server ubuntu apply`
reads it from the systemd credential `ws-secrets`, in `$CREDENTIALS_DIRECTORY`, for `--secrets` and
`--check` alike. Without the credential it fails and names `LoadCredential`. It reads no standard
input and no terminal. The wizard starts the run as a unit with
`LoadCredentialEncrypted=ws-secrets`. By hand, on a server, as root, with ws's file:

```sh
systemd-run --pty --wait -p LoadCredentialEncrypted=ws-secrets ws server ubuntu apply --secrets /etc/ws/secrets.age
```

systemd decrypts the credential for the unit alone, and drops it once the run ends.

The roles read these tables, and ws refuses any other key:

- `[env]`: names to values, as [Secrets](#secrets) says;
- `[git."URL"]`: the entries of [Git credentials](../git/README.md#git-credentials);
- `[file]` and `[password]`: the ikeward profile's files and the `.p12`'s password
  ([The profile](../ikeward/README.md#the-profile)).

No message, `--check` included, shows a value. Nothing plain reaches the disk but the
files that the roles write.
